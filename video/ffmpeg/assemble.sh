#!/bin/bash
# Assemble a self-contained runtime tree under /out/rootfs: the ffmpeg
# and ffprobe binaries, the shared-library closure they need at runtime
# (the codec libs are static, so this is mainly musl libc, libstdc++,
# libgcc, libplacebo, libdovi, and the Vulkan loader), the musl dynamic
# linker, and the Mesa lavapipe software-Vulkan driver plus its ICD
# manifest. The deploy step copies this tree verbatim into the squashfs.
#
# pipefail is intentionally left off: copy_closure pipes through head
# and grep, which can legitimately close early or match nothing.
set -eu

ROOT=/out/rootfs
mkdir -p "$ROOT"

# Binaries live at the filesystem root: box.meta sets PATH=/.
install -Dm755 /usr/local/bin/ffmpeg "$ROOT/ffmpeg"
install -Dm755 /usr/local/bin/ffprobe "$ROOT/ffprobe"

# Copy the shared-library closure of a binary, preserving absolute
# paths so the dynamic linker finds each library where it expects it.
copy_closure() {
	ldd "$1" 2>/dev/null | awk '/=>/ {print $3} /ld-musl/ {print $1}' \
		| grep '^/' | sort -u | while read -r lib; do
		[ -e "$lib" ] && install -Dm755 "$lib" "$ROOT$lib"
	done
}

copy_closure "$ROOT/ffmpeg"
copy_closure "$ROOT/ffprobe"

# The musl program interpreter (PT_INTERP).
for interp in /lib/ld-musl-*.so.1; do
	[ -e "$interp" ] && install -Dm755 "$interp" "$ROOT$interp"
done

# Mesa lavapipe: the Vulkan loader dlopens this driver at runtime via
# the ICD manifest, so it isn't in ffmpeg's ldd output — copy it and its
# library closure (libLLVM, libgallium, …) explicitly, along with the
# canonical ICD manifest the Dockerfile wrote.
install -Dm755 /usr/lib/libvulkan_lvp.so "$ROOT/usr/lib/libvulkan_lvp.so"
copy_closure /usr/lib/libvulkan_lvp.so
install -Dm644 /usr/share/vulkan/icd.d/lvp_icd.json \
	"$ROOT/usr/share/vulkan/icd.d/lvp_icd.json"

echo "=== runtime tree ==="
find "$ROOT" \( -type f -o -type l \) | sort
echo "=== total size ==="
du -sh "$ROOT"
