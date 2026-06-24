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

gen_buildinfo() {
	# The working copy (@) is usually empty at build time — we land
	# changes on main, then build.  So stamp the commit that was
	# actually built: the latest non-empty ancestor of @ (e.g. @
	# itself when it holds changes, otherwise the commit it sits on).
	rev=$(jj log -r 'latest(::@ ~ empty())' --no-graph --color=never -T change_id) || return 1
	[ -n "$rev" ] || return 1

	printf "%s" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >buildinfo/embed-date.txt
	printf "%s" "$rev" >buildinfo/embed-changeid.txt
	jj log -r "$rev" --no-graph --color=never -T commit_id \
		>buildinfo/embed-commitid.txt
	jj log --color=never -r "fork_point($rev | main)::($rev | main)" \
		-T 'separate(" ", change_id.shortest(8), commit_id.shortest(8), bookmarks, description.first_line())' \
		>buildinfo/embed-log.txt
}

case "${1:-}" in
	deploy)
		mkdir -p deploy
		dir=$(mktemp -d /tmp/act3.XXXXXX)
		trap "rm -rf '$dir'" EXIT

		# Vendor jassub for the high-fidelity ASS subtitle path
		# (libass via WebAssembly). Idempotent; the network fetch
		# only happens when dist/ is empty or out of date with the
		# pinned version in gen.go. The package detects the bundle
		# at runtime via the embed.FS, so no build tag is needed —
		# without this step dist/ stays at the placeholder Readme
		# and jassub.Path returns "" at runtime.
		go run web/jassub/gen.go

		gen_buildinfo
		CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags prod,nodynamic,buildinfoembed -o $dir/act3

		## Build ffmpeg and ffprobe from source via Docker.
		## See Dockerfile.ffmpeg for the build configuration.
		## Docker layer caching makes rebuilds fast when only the Go app changes.
		##
		## The build emits a runtime tree under /out/rootfs: the binaries
		## plus the shared libraries, musl loader, and Mesa lavapipe
		## Vulkan driver they need (ffmpeg can't be fully static because
		## the Vulkan loader dlopens its driver). Stage the whole tree,
		## plus the app binary and box.meta, into one root directory so
		## everything lands at the paths the dynamic linker and box
		## runtime expect.
		ffmpeg_image=act3-ffmpeg
		if ! docker image inspect "$ffmpeg_image" >/dev/null 2>&1; then
			docker build -t "$ffmpeg_image" -f Dockerfile.ffmpeg video/ffmpeg
		fi
		root="$dir/root"
		mkdir -p "$root"
		docker run --rm "$ffmpeg_image" tar -C /out/rootfs -cf - . \
			| tar -C "$root" -xf -
		cp "$dir/act3" "$root/act3"
		cp box.meta "$root/box.meta"

		version=$(build_version)
		image="act3.$version.app"

		## Combines the staged root directory into a squashfs file system
		## image. A single source directory has its contents placed at the
		## image root (multiple sources would instead nest each by name).
		mksquashfs \
			"$root" \
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
	run)
		shift
		gen_buildinfo
		bin="$(mktemp -d /tmp/act3-run.XXXXXX)/act3"
		trap 'rm -rf "$(dirname "$bin")"' EXIT
		go build -tags buildinfoembed -o "$bin" .
		"$bin" "$@"
		;;
	regen)
		# Run go generate on each commit in the revset, so any drift in
		# generated files (bundles, sqlc output, html/tag.go) lands on the
		# commit that introduced the source change. Useful after rewriting
		# history with jj, since the post-write Edit/Write hook doesn't
		# fire on jj operations.
		revset="${2:-$(jj config get revsets.fix 2>/dev/null || echo 'reachable(@, mutable())')}"
		here=$(jj log -r '@' --no-graph -T 'change_id')
		trap 'jj edit "$here" >/dev/null 2>&1 || true' EXIT
		for cid in $(jj log --reversed -r "$revset" --no-graph -T 'change_id ++ "\n"'); do
			echo "regen $cid"
			jj edit "$cid"
			go generate ./...
		done
		;;
	"")
		echo "Usage: $0 [command]"
		echo
		echo "Commands:"
		echo
		echo "    container  Build & run container for dev"
		echo "    run        Run the server from the working copy, stamped with build info"
		echo "    deploy     Deploy the image to the USB stick"
		echo "    regen      Run go generate on each commit in revset (default: same as jj fix)"
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
