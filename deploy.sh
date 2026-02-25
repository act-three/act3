#!/bin/sh
set -eo pipefail

cd $(dirname $0)
mkdir -p deploy
dir=$(mktemp -d /tmp/act3.XXXXXX)
trap "rm -rf '$dir'" EXIT

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $dir/act3

## Build static ffmpeg and ffprobe from source via Docker.
## See video/ffmpeg/Dockerfile for the build configuration.
## Docker layer caching makes rebuilds fast when only the Go app changes.
ffmpeg_image=act3-ffmpeg
if ! docker image inspect "$ffmpeg_image" >/dev/null 2>&1; then
    docker build -t "$ffmpeg_image" video/ffmpeg/
fi
docker run --rm "$ffmpeg_image" cat /out/ffmpeg > "$dir/ffmpeg"
docker run --rm "$ffmpeg_image" cat /out/ffprobe > "$dir/ffprobe"
chmod +x "$dir/ffmpeg" "$dir/ffprobe"

today=$(date +%Y%m%d)
n=1
while [ -f "deploy/act3.$today.$(printf %02d $n).app" ]
do n=$((n + 1))
done
version="$today.$(printf %02d $n)"
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
    -force-uid 500\
    -force-gid 500\
    -default-mode 0755

ls -l "$dir" deploy/$image
ssh root@pepper app update act3 $version <deploy/$image
ssh root@pepper boxdown act3
ssh root@pepper boxup act3
