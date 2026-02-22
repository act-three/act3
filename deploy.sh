#!/bin/sh
set -eo pipefail

cd $(dirname $0)
mkdir -p deploy
dir=$(mktemp -d /tmp/act3.XXXXXX)
trap "rm -rf '$dir'" EXIT

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $dir/act3

## Fetch static ffmpeg and ffprobe binaries for linux-amd64.
## https://github.com/shaka-project/static-ffmpeg-binaries/releases/tag/n7.1-2
ffmpeg_url=https://github.com/shaka-project/static-ffmpeg-binaries/releases/download/n7.1-2/ffmpeg-linux-x64
ffmpeg_sha256=c656811eecb083e3e0e8ee706a572fd919d275ba5ab34eb9f4762302afce4270
ffprobe_url=https://github.com/shaka-project/static-ffmpeg-binaries/releases/download/n7.1-2/ffprobe-linux-x64
ffprobe_sha256=31c308c383fc0be13c5b8d83e98eb4acca7011f2f79eec8dc57a2a04796255e5

fetch() {
    local url=$1 sha256=$2 dest=$3
    if [ -f "$dest" ] && echo "$sha256  $dest" | shasum -a 256 -c -; then
        return
    fi
    curl -fsSL -o "$dest" "$url"
    echo "$sha256  $dest" | shasum -a 256 -c -
}

cache=$HOME/.cache/act3
mkdir -p "$cache"
fetch "$ffmpeg_url" "$ffmpeg_sha256" "$cache/ffmpeg"
fetch "$ffprobe_url" "$ffprobe_sha256" "$cache/ffprobe"
chmod +x "$cache/ffmpeg" "$cache/ffprobe"

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
    "$cache/ffmpeg" \
    "$cache/ffprobe" \
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
