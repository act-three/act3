/*
How We Manage Video

There is a dizzying variety of codecs, containers, and protocols out there.
Here is our approach.

# Requirements

Modern support.
We don't need extreme compatibility with old or obscure devices.
Our users predominantly use recent versions of Chrome or Safari,
plus Roku and Apple TV.

# Formats

We've settled on the following short list of formats.
This covers our compatibility needs.

  - HLS with fMP4
  - H.264 and H.265
  - (audio tbd)
  - (subtitles tbd)

# Video

For video codec, we prefer H.265 since it has better quality per bitrate.
(We're willing to spend the CPU time needed to encode it.)
We always generate H.265 for reencoded renditions.
For remuxed source video
(used for the "original quality" rendition at the top of the ladder),
we allow either H.265 or H.264.
If the source video uses another codec, we reencode it.

# Audio

(tbd) prefer AAC (probably).
List of formats to allow (based on compatibility),
and reencode others to AAC.
Probably never reencode for bitrate purposes.

# Subtitles

Convert everything to UTF-8.

Format tbd. Probably always VTT (reencoding others).

# Bit-Rate Ladder

We encode each source video to the following renditions:

  - (tbd)

We also support hand-encoded renditions.
Usually, we prefer high-quality, high-bitrate releases.
Occasionally, someone will upload a release of a video source
encoded at a lower bitrate.
It's possible this video will be higher quality
than what we would generate automatically.
If an admin adds a release of a video at a lower bitrate,
this lower-bitrate source will replace the next-highest rendition
in the auto-generated ladder.
It won't be used as an encoding source for lower-bitrate renditions,
and it won't replace the original, higher bitrate source
(in case there is no auto-generated rendition
between the high-bitrate source and the lower-bitrate source).

# Downloadable Renditions

We also generate one rendition in a downloadable format.
This is the same encode as the top of the bitrate ladder for streaming,
just repackaged.
We offer two download options:
this rendition and the actual original source video file.
(Usually, this rendition is just remuxed from the source video,
but it is usually in a different container format.
Most source video is mkv;
we generate mp4.)

We do this for each added release.

# Example

We add a high-bitrate 4k UHD Bluray remux of Dune (2021) at 65.2 Mb/s (71 GB),
plus a much lower bitrate 1080p Web-DL at 6,478 kb/s (7 GB).

This results in N streaming renditions:

  - 65.2 MB/s 4k (remux from original 4k)
  - (tbd) (reencode from original 4k)
  - 6,478 kb/s 1080p (remux from original 1080p)
  - (tbd) (reencode from original 4k)
  - (tbd) (reencode from original 4k)
  - ...

Plus four download options:

  - 4k Original Quality MP4
  - 4k Original Source File MKV
  - 1080p Original Quality MP4
  - 1080p Original Source File MKV
*/
package video
