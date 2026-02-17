/*
How We Manage Video

There is a dizzying variety of codecs, containers, and protocols out there.
Here is our approach.

# General Considerations & Requirements

Modern support.
We don't need extreme compatibility with old or obscure devices.
Our users predominantly use recent versions of Chrome or Safari,
on Windows, macOS, and iOS,
plus Roku and Apple TV.
In cases where software or hardware impacts compatibility,
we support only recent versions, up to 2 years old.

We prioritize higher quality in ideal viewing conditions,
and we're willing to sacrifice quality in non-ideal conditions
when those are in conflict.
For example, HLS requires all renditions in a playlist to use the same coded.
If a source video is encoded in H.264,
we use H.264 for all renditions,
which means lower quality per bitrate for non top-tier renditions
(which are primarily constrained by bitrate, unlike the top tier).
The alternative would be to transcode the top tier to HEVC,
allowing the lower tiers to achieve a higher quality for the same bitrate,
but would lose quality for the top tier.

# Formats

We've settled on the following short list of formats.
This covers our compatibility needs.

  - HLS with fMP4
  - H.264 and HEVC
  - AAC
  - (subtitles tbd)

See [HTTP Live Streaming (HLS) authoring specification for Apple devices][HLS]
for technical requirements.
We use only a small subset of the formats and settings allowed by that document.
We don't distinguish between wi-fi delivery, cellular delivery,
and other network types.

# Video

For video codec, we prefer HEVC since it has better quality per bitrate.
If the source video is H.264 or HEVC, we use the source codec.
Otherwise, we reencode to HEVC for all renditions (even the top).

Note that HLS requires all renditions in a playlist to use the same codec.

We generally prioritize quality. This includes:

  - 2-pass encoding (faster preset for pass 1 analysis, medium for pass 2)
  - using the highest profile and level permitted by the [HLS spec][HLS]
    (ignoring requirements that don't apply to devices and OS versions
    outside our supported range)

If we experience compatibility problems with other devices (Roku, Android),
we might have to revisit some of these choices.

# Audio

We encode to AAC stereo (2-channel) for all streaming renditions.
Stereo is used because non-standard surround channel layouts
(such as 5.1(side)) require a PCE (Program Config Element)
in the AAC bitstream, which CoreMedia's HLS fMP4 parser rejects.

If the source audio is already AAC stereo (≤2 channels),
we copy the audio stream as-is.
Otherwise, we reencode to AAC with a stereo downmix.

When the source has surround audio (>2 channels),
we add one extra rendition with the same video as the best rendition
but with 5.1(back) audio instead of stereo.
The 5.1(back) layout maps to the standard MPEG-4 channel configuration 6,
which does not require PCE and is accepted by all HLS clients.
This converts non-standard layouts like 5.1(side)
to the compatible 5.1 rear-surround layout.

# Subtitles

For now, we omit subtitles entirely.

> WIP notes for future work -- do not implement yet:
>
> Format tbd.
>
> Let's see if we can get ASS rendering working reliably in web browsers,
> and maybe also serve a minimal set of other formats (ideally just one)
> for non-browser devices.
>
> Always use UTF-8. Convert other encodings as needed.

# Bit-Rate Ladder

We use the following bitrate ladder:

  - 500000 kbit/s; source resolution; source fps
  - 20000 kbit/s; source resolution; source fps
  - 5000 kbit/s; min(1080p, source); source fps
  - 3000 kbit/s; min(720p, source); source fps
  - 1500 kbit/s; min(540p, source); min(30fps, source)
  - 420 kbit/s; min(540p; source); min(30fps, source)

The bitrate targets above are reduced by about 20% for 24fps video,
according to the note in the [HLS spec][HLS].

## Best Rendition

If the source video is H.264 or HEVC
and its bitrate is less than the top tier of the ladder (500 Mbit/s),
then we remux it, using its video stream as-is
for the highest-bitrate rendition.

If the source video is some other codec
and its bitrate is less than ~90% of the top tier of the ladder (450 Mbit/s),
we set the target bitrate of the best rendition to
about 10% higher than the source video bitrate,
and reencode to HEVC.

Otherwise (that is, if the source video has a very high bitrate),
we reencode the source video to HEVC at a target bitrate of 500 Mbit/s.

## Remaining Renditions

We generate one for every bitrate in the ladder lower than the best rendition,
by re-encoding the source video.

> Possible future work -- do not implement (YAGNI):
>
> We might also support hand-encoded renditions.
> Usually, we prefer high-quality, high-bitrate releases.
> Occasionally, someone will upload a release of a video source
> encoded at a lower bitrate.
> It's possible this video will be higher quality
> than what we would generate automatically.
> If an admin adds a release of a video at a lower bitrate,
> this lower-bitrate source will replace the next-highest rendition
> in the auto-generated ladder.
> It won't be used as an encoding source for lower-bitrate renditions,
> and it won't replace the original, higher bitrate source
> (in case there is no auto-generated rendition
> between the high-bitrate source and the lower-bitrate source).

# Downloadable Renditions

> Future work:
>
> We also generate one rendition in a downloadable format.
> This is the same encode as the top of the bitrate ladder for streaming,
> just repackaged into a "fast-start" mp4 container.
>
> We offer two download options:
>   - this generated rendition
>   - the actual original source video file
>
> (Usually, this rendition is just remuxed from the source video,
> but it is usually in a different container format.
> Most source video is mkv;
> we generate mp4.)

# Example

We add a high-bitrate 4k UHD Bluray remux of Dune (2021) at 65.2 Mbit/s (71 GB),
plus a much lower bitrate 1080p Web-DL at 6,478 kbit/s (7 GB).

This results in N streaming renditions:

  - 65.2 MB/s 4k (remux from original 4k)
  - (tbd) (reencode from original 4k)
  - 6,478 kbit/s 1080p (remux from original 1080p)
  - (tbd) (reencode from original 4k)
  - (tbd) (reencode from original 4k)
  - ...

Plus four download options:

  - 4k Original Quality MP4
  - 4k Original Source File MKV
  - 1080p Original Quality MP4
  - 1080p Original Source File MKV

[HLS]: https://developer.apple.com/documentation/http-live-streaming/hls-authoring-specification-for-apple-devices
*/
package video
