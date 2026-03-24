#!/bin/bash
set -euo pipefail
cd $(dirname $0)

# Images are files stored in the deploy dir,
# named like act3-20241231.01.app.
# The last two digits are a counter.

latest_image() {
	ls -1 deploy | fgrep .app | sort | tail -n 1
}

latest_deployed_image() {
	cat deploy/latest 2>/dev/null || true
}

next_version() {
	today=$(date +%Y%m%d)
	n=1
	while [ -f "deploy/act3.$today.$(printf %02d $n).app" ]; do
		n=$((n + 1))
	done
	echo "$today.$(printf %02d $n)"
}

fixup_version() {
	today=$(date +%Y%m%d)
	n=1
	while [ -f "deploy/act3.$today.$(printf %02d $((n + 1))).app" ]; do
		n=$((n + 1))
	done
	echo "$today.$(printf %02d $n)"
}

build_version() {
	v=$(fixup_version)
	if [ "act3.$v.app" = "$(cat deploy/latest)" ]; then
		next_version
	else
		echo $v
	fi
}

case "${1:-}" in
	container)
		container="act3-dev"
		image="act3-dev"
		chrome_container="act3-chrome"
		chrome_image="act3-chrome"

		if docker container inspect $container &>/dev/null; then
			echo "Container $container exists; aborting."
			exit 1
		fi

		echo "Building images..."
		docker build -t $chrome_image -f Dockerfile.chrome .
		docker build -t $image -f Dockerfile.dev .

		echo "Starting dev container..."
		docker run -d \
			--name $container \
			--hostname $container \
			--add-host=host.docker.internal:host-gateway \
			-p 2222:22 \
			-p 4444:4444 \
			-p 4445:4445 \
			-v "$HOME/Downloads:/Downloads" \
			-v "$HOME/.gitconfig:/home/dev/.gitconfig:ro" \
			-v "$HOME/.config/git:/home/dev/.config/git:ro" \
			$image

		# Chrome shares the dev container's network namespace,
		# so CDP on localhost:9222 is reachable from the dev container.
		docker rm -f $chrome_container 2>/dev/null || true
		echo "Starting Chrome container..."
		docker run -d \
			--name $chrome_container \
			--network container:$container \
			--restart unless-stopped \
			$chrome_image

		# Fix ownership of dirs auto-created by bind mounts
		docker exec $container chown dev:dev /home/dev/.config

		# Machine-specific env vars (single source of truth: /etc/profile.d/)
		docker exec -i $container sh -c "cat >> /etc/profile.d/act3.sh" <<-EOF
			export A3TMDBTOKEN=${A3TMDBTOKEN:-}
			export A3TRANSMISSION=http://host.docker.internal:9091/transmission/rpc
			export A3FFMPEGVIDEOPRESET=${A3FFMPEGVIDEOPRESET:-}
		EOF

		# Aliases (in .bashrc so they apply to all interactive shells, not just login)
		docker exec -i $container sh -c "cat >> /home/dev/.bashrc" <<-EOF
			alias claude='claude --permission-mode bypassPermissions'
		EOF

		# Make interactive login shell source ~/.bashrc
		docker exec -i $container sh -c "cat >> /home/dev/.profile" <<-EOF
			[ -f ~/.bashrc ] && . ~/.bashrc
		EOF

		# Copy host Claude settings (OAuth credentials, preferences)
		docker cp "$HOME/.claude.json" $container:/home/dev/.claude.json
		docker exec $container chown dev:dev /home/dev/.claude.json

		# act3-mcp — starts and stops dev server reliably
		docker exec -u dev $container /home/dev/.local/bin/claude mcp add --scope user act3-mcp -- ./cmd/act3-mcp/run.sh

		# gopls MCP — Go language server for rename, references, diagnostics
		docker exec -u dev $container /home/dev/.local/bin/claude mcp add --scope user gopls -- /home/dev/go/bin/gopls mcp

		# Playwright MCP — connects to Chromium in the Chrome container
		docker exec -u dev $container /home/dev/.local/bin/claude mcp add --scope user playwright \
			-- npx @playwright/mcp@latest --cdp-endpoint http://localhost:9222

		# Install SSH key
		docker cp "$HOME/.ssh/id_ed25519.pub" $container:/home/dev/.ssh/authorized_keys
		docker exec $container chown dev:dev /home/dev/.ssh/authorized_keys
		docker exec $container chmod 600 /home/dev/.ssh/authorized_keys

		# Use SSH here (instead of docker exec) for agent forwarding.
		echo "Cloning repo..."
		ssh act3-dev "git clone git@github.com:em-ily-dev/act3.git ~/act3 && mkdir -p ~/.local/act3 && cd ~/act3 && ./mk git-setup"

		# Copy gitignored files not included in the clone
		docker cp ui/icon/untitled-icons.zip "$container:/home/dev/act3/ui/icon/untitled-icons.zip"
		docker exec $container sh -c "cd /home/dev/act3/ui/icon && go run gen.go"

		echo "Container $container is running."
		echo "Connect with: zed ssh://act3-dev/home/dev/act3"
		;;
	deploy)
		mkdir -p deploy
		dir=$(mktemp -d /tmp/act3.XXXXXX)
		trap "rm -rf '$dir'" EXIT

		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags prod -o $dir/act3

		## Build static ffmpeg and ffprobe from source via Docker.
		## See Dockerfile.ffmpeg for the build configuration.
		## Docker layer caching makes rebuilds fast when only the Go app changes.
		ffmpeg_image=act3-ffmpeg
		if ! docker image inspect "$ffmpeg_image" >/dev/null 2>&1; then
			docker build -t "$ffmpeg_image" -f Dockerfile.ffmpeg video/ffmpeg
		fi
		docker run --rm "$ffmpeg_image" cat /out/ffmpeg >"$dir/ffmpeg"
		docker run --rm "$ffmpeg_image" cat /out/ffprobe >"$dir/ffprobe"
		chmod +x "$dir/ffmpeg" "$dir/ffprobe"

		version=$(build_version)
		image="act3.$version.app"

		## Combines given files and directories into a squashfs file system image.
		mksquashfs \
			box.meta \
			"$dir/act3" \
			"$dir/ffmpeg" \
			"$dir/ffprobe" \
			deploy/$image \
			-p '/data d 0555 0 0' \
			-p '/database d 0555 0 0' \
			-p '/storage d 0555 0 0' \
			-p '/tmp d 0755 0 0' \
			-p '/dev d 0555 0 0' \
			-p '/etc d 0555 0 0' \
			-p '/etc/ssl d 0555 0 0' \
			-p '/etc/ssl/cert.pem f 0444 0 0 cat /etc/ssl/cert.pem' \
			-p '/etc/resolv.conf f 0444 0 0 cat /dev/null' \
			-p '/proc d 0555 0 0' \
			-p '/sys d 0555 0 0' \
			-force-uid 500 -force-gid 500 -default-mode 0755

		ls -l "$dir" deploy/$image
		ssh root@pepper app update act3 $version <deploy/$image
		ssh root@pepper boxdown act3
		ssh root@pepper boxup act3
		echo $image >deploy/latest
		;;
	git-setup)
		go build -o .git/hooks/act3vet ./cmd/act3vet
		go build -o .git/hooks/commit-msg ./cmd/commit-msg
		cp lib/pre-commit.sh .git/hooks/pre-commit
		chmod +x .git/hooks/pre-commit
		echo "Installed git hooks"
		;;
	"")
		echo "Usage: $0 [command]"
		echo
		echo "Commands:"
		echo
		echo "    container  Build & run container for dev"
		echo "    deploy     Deploy the image to the USB stick"
		echo "    git-setup  Install git hooks"
		echo
		echo "Last deployed: $(latest_deployed_image)"
		echo "Latest image:  $(latest_image)"
		echo
		;;
	*)
		echo "Error: Unknown subcommand '$1'"
		exit 1
		;;
esac
