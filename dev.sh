#!/bin/bash
set -euo pipefail

CONTAINER_NAME="act3-dev"
IMAGE_NAME="act3-dev"

# Stop any existing container
if docker container inspect "$CONTAINER_NAME" &>/dev/null
then
	echo "Container $CONTAINER_NAME exists; aborting."
	exit 1
fi

# Build the image
echo "Building image..."
docker build -t "$IMAGE_NAME" -f Dockerfile .

# Start the container
echo "Starting container..."
docker run -d \
	--name "$CONTAINER_NAME" \
	--add-host=host.docker.internal:host-gateway \
	-p 2222:22 \
	-p 4444:4444 \
	-v "$HOME/Downloads:/Downloads" \
	-v "$HOME/.gitconfig:/home/dev/.gitconfig:ro" \
	-v "$HOME/.config/git:/home/dev/.config/git:ro" \
	-v "$HOME/.claude:/home/dev/.claude" \
	"$IMAGE_NAME"

# Fix ownership of dirs auto-created by bind mounts
docker exec "$CONTAINER_NAME" chown dev:dev /home/dev/.config

# Append machine-specific env vars to .profile
docker exec -i "$CONTAINER_NAME" sh -c "cat >> /home/dev/.profile" <<EOF
export A3TMDBTOKEN="${A3TMDBTOKEN:-}"
export A3TRANSMISSION="http://host.docker.internal:9091/transmission/rpc"
export A3FFMPEGVIDEOPRESET="${A3FFMPEGVIDEOPRESET:-}"
EOF

# Install SSH key
docker cp "$HOME/.ssh/id_ed25519.pub" "$CONTAINER_NAME:/home/dev/.ssh/authorized_keys"
docker exec "$CONTAINER_NAME" chown dev:dev /home/dev/.ssh/authorized_keys
docker exec "$CONTAINER_NAME" chmod 600 /home/dev/.ssh/authorized_keys

# Clone the repo and create storage dir (via SSH for agent forwarding)
echo "Cloning repo..."
ssh act3-dev "git clone git@github.com:em-ily-dev/act3.git ~/act3 && mkdir -p ~/.local/act3"

# Copy gitignored files not included in the clone
docker cp ui/icon/untitled-icons.zip "$CONTAINER_NAME:/home/dev/act3/ui/icon/untitled-icons.zip"
docker exec "$CONTAINER_NAME" chown dev:dev /home/dev/act3/ui/icon/untitled-icons.zip

echo "Container '$CONTAINER_NAME' is running."
echo "Connect with: zed ssh://act3-dev/home/dev/act3"
