package view

import (
	"fmt"
	"net/url"
	"slices"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/web/jassub"
)

func PlayerContainer(id string, player domi.Node) domi.Node {
	return domi.Keyed("div")(attr.ID("player"))(
		func(yield func(string, domi.Node) bool) {
			if player == nil {
				return
			}
			yield(id, player)
		},
	)
}

// PlayerForEpisode and PlayerForMovie render the player for an episode
// or movie edition's video.
func PlayerForEpisode(v *model.Video, ep *model.Episode, qualityOpts []model.QualityOption, captionsOpts []model.SubtitleOption, audioOpts []model.AudioOption, initAudio, initSubtitle string, pinAudio bool) domi.Node {
	return player(v, playerTitleForEpisode(ep), qualityOpts, captionsOpts, audioOpts, initAudio, initSubtitle, pinAudio)
}

func PlayerForMovie(v *model.Video, med *model.MovieEditionHead, qualityOpts []model.QualityOption, captionsOpts []model.SubtitleOption, audioOpts []model.AudioOption, initAudio, initSubtitle string, pinAudio bool) domi.Node {
	return player(v, playerTitleForMovie(med), qualityOpts, captionsOpts, audioOpts, initAudio, initSubtitle, pinAudio)
}

func playerTitleForMovie(med *model.MovieEditionHead) string {
	title := med.Title()
	if y := med.Year(); y != "" {
		return fmt.Sprintf("%s (%s)", title, y)
	}
	return title
}

// player renders the play surface. An empty initAudio falls back to
// the video's default audio; an empty initSubtitle leaves captions off.
func player(
	v *model.Video,
	title string,
	qualityOpts []model.QualityOption,
	captionsOpts []model.SubtitleOption,
	audioOpts []model.AudioOption,
	initAudio, initSubtitle string,
	pinAudio bool,
) domi.Node {
	if initAudio == "" && len(audioOpts) > 0 {
		initAudio = audioOpts[0].ID
	}
	return html.Div(
		domi.Opaque,
		attr.ID("full-player"),
		Class("v-player"),
		stimulus.Controller("player"),
		stimulus.Value("player", "title")(title),
		stimulus.Value("player", "playing")("false"),
		stimulus.Value("player", "paused")("true"),
		stimulus.Value("player", "stopped")("true"),
		stimulus.Value("player", "harlow")("false"),
		stimulus.Value("player", "hide-controls")("false"),
		stimulus.Value("player", "video-id")(v.ID()),
		stimulus.Value("player", "current-quality")(""),
		stimulus.Value("player", "quality-menu-open")("false"),
		stimulus.Value("player", "current-subtitle")(initSubtitle),
		stimulus.Value("player", "captions-menu-open")("false"),
		stimulus.Value("player", "current-audio")(initAudio),
		stimulus.Value("player", "audio-menu-open")("false"),

		// jassub URLs are empty in default builds; the
		// player JS treats an empty host URL as "feature off"
		// and stays on the native textTracks path. In
		// -tags jassub builds these resolve to the digest-
		// cache-busted artifacts under /-/jassub/.
		stimulus.Value("player", "jassub-host-url")(jassub.Path("jassub.js")),
		stimulus.Value("player", "jassub-worker-url")(jassub.Path("jassub-worker.js")),
		stimulus.Value("player", "jassub-wasm-url")(jassub.Path("jassub-worker-modern.wasm")),
		stimulus.Value("player", "jassub-font-url")(jassub.Path("default.woff2")),

		stimulus.Action("keydown.h@window->player#toggleHarlow"),
		stimulus.Action("keydown@window->player#handleKey"),
		stimulus.Action("keyup@window->player#handleKey"),

		stimulus.Action("mousemove->player#handleControls"),
		stimulus.Action("mouseleave->player#handleControls"),
		stimulus.Action("touchstart->player#handleControls"),
		stimulus.Action("touchmove->player#handleControls"),
		stimulus.Action("enterfullscreen->player#handleControls"),
		stimulus.Action("exitfullscreen->player#handleControls"),

		// Any click anywhere in the player closes any open menu.
		// Capture-phase so it runs before togglePlay:self on
		// .v-player-controls — the bg-click on the video
		// dismisses the menu without also flipping playback.
		// All other controls (volume, play button, other menu
		// toggles, etc.) handle their own click on bubble-up
		// after the menu has been dismissed.
		stimulus.Action("click->player#closeMenusOnClick:capture"),
	)(
		html.Div(Class("v-player-video-layer"))(
			html.Video(
				Class("v-player-video"),
				Attr("playsinline")(""),
				stimulus.Target("player", "video"),

				stimulus.Action("playing->player#handlePlaying"),
				stimulus.Action("play->player#handlePlaying"),
				stimulus.Action("pause->player#handlePlaying"),
				stimulus.Action("ended->player#handlePlaying"),
				stimulus.Action("emptied->player#handlePlaying"),
				stimulus.Action("timeupdate->player#handlePlaying"),

				// Handle time change on media.
				stimulus.Action("timeupdate->player#handleTimeUpdate"),
				stimulus.Action("seeking->player#handleTimeUpdate"),
				stimulus.Action("seeked->player#handleTimeUpdate"),
				stimulus.Action("durationchange->player#handleDurationChange"),

				// Wire menus to manifest tracks. Per spec, textTracks
				// and audioTracks are populated by loadedmetadata —
				// earlier events can fire before manifest parsing
				// is complete (ACT-169).
				stimulus.Action("loadedmetadata->player#handleLoadedMetadata"),

				// Enable the seekbar once segments are available.
				// Setting currentTime before HAVE_CURRENT_DATA breaks
				// Safari's native HLS pipeline (ACT-171).
				stimulus.Action("loadeddata->player#handleLoadedData"),

				// Buffer progress.
				stimulus.Action("progress->player#handleProgress"),
				stimulus.Action("playing->player#handleProgress"),
				stimulus.Action("seeking->player#handleProgress"),
				stimulus.Action("seeked->player#handleProgress"),

				// Volume changes.
				stimulus.Action("volumechange->player#handleVolume"),

				// Loading state.
				stimulus.Action("waiting->player#handleLoading"),
				stimulus.Action("canplay->player#handleLoading"),
				stimulus.Action("seeked->player#handleLoading"),
				stimulus.Action("playing->player#handleLoading"),

				// Speed change.
				stimulus.Action("ratechange->player#handleRate"),

				// Disable right click.
				stimulus.Action("contextmenu->player#handleContextMenu"),
			)(
				html.Source(
					attr.Src(initialPlaylistPath(v, initAudio, pinAudio)),
					attr.Type("application/vnd.apple.mpegurl"),
				),
			),
			playerCaptionsTemplate(captionsOpts),
		),
		html.Div(
			Class("v-player-controls"),
			stimulus.Target("player", "controls"),
			stimulus.Action("click->player#togglePlay:self"),
		)(
			html.Div(Class("v-player-overlay-top"))(
				// TODO: optimistic commit to close (or hide) the player faster.
				Button(onPlayerClose(), stimulus.Target("player", "dismiss"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/x-close")),
				Box(Class("v-player-title"))(Text(title)),
			),
			html.Div(Class("v-player-overlay-bottom"))(
				html.Div(Class("v-player-time-row"))(
					Box(stimulus.Target("player", "currentTime"))(Text("0:00")),
					playerSeekBar(),
					Box(stimulus.Target("player", "duration"))(Text("0:00")),
				),
				html.Div(Class("v-player-button-row"))(
					html.Div(Class("v-player-button-group"), Attr("data-align")("start"))(
						playerCaptionsMenu(captionsOpts, initSubtitle),
						playerAudioMenu(audioOpts, initAudio),
						playerVolumeBar(),
					),

					html.Div(Class("v-player-button-group"), Attr("data-align")("center"))(
						Button(stimulus.Action("click->player#skipBackward"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/refresh-ccw-01")),
						Button(stimulus.Action("click->player#togglePlay"), ButtonSurface, ButtonCircle, ButtonSize3)(
							Box(Class("v-player-icon-play"))(Icon("solid/play")),
							Box(Class("v-player-icon-pause"))(Icon("free/pause")),
						),
						Button(stimulus.Action("click->player#skipForward"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/refresh-cw-01")),
					),

					html.Div(Class("v-player-button-group"), Attr("data-align")("end"))(
						playerQualityMenu(qualityOpts),
						Button(stimulus.Action("click->player#toggleFullscreen"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/maximize-02")),
					),
				),
			),
		),
	)
}

func playerTitleForEpisode(ep *model.Episode) string {
	year := ""
	if d := ep.Airdate(); d != "" {
		y, _, _ := strings.Cut(d, "-")
		year = " (" + y + ")"
	}
	return fmt.Sprintf("%s %s %s%s",
		ep.SeriesHead().Title(),
		ep.SnnEnn(),
		ep.Title(),
		year,
	)
}

// initialPlaylistPath is the player's initial <source> URL: the video's
// multivariant playlist, with the audio rendition pinned when pinAudio
// is set.
func initialPlaylistPath(v *model.Video, initAudio string, pinAudio bool) string {
	if !pinAudio || initAudio == "" {
		return v.PlaylistPath()
	}
	q := url.Values{"a": {initAudio}}
	return v.PlaylistPath() + "?" + q.Encode()
}

// playerQualityMenu emits the quality popover. The "Auto" entry has
// an empty quality-id (full MV playlist with all renditions); other
// entries pin to a specific video rendition. The JS composes the
// final URL by combining the picked quality-id with the current
// audio-id (in Chrome where audio switching also requires a
// source-swap; in Safari, audio is switched via the audioTracks API
// and the URL only carries the quality-id).
func playerQualityMenu(opts []model.QualityOption) domi.Node {
	// Render Auto at the bottom — visually it's the dynamic default
	// and the per-rendition pins read top-down by resolution.
	opts = autoLast(opts)
	labels := qualityLabels(opts)
	var items []domi.Node
	for i, opt := range opts {
		btnAttrs := []domi.Attr{
			attr.Type("button"),
			stimulus.Action("click->player#setQuality"),
			Attr("data-player-quality-id-param")(opt.RenditionID),
			Class("v-player-menu-item"),
		}
		if opt.RenditionID == "" {
			btnAttrs = append(btnAttrs, Attr("data-active")(""))
		}
		items = append(items, html.Button(btnAttrs...)(LabelNode("line/check", TruncTail(labels[i]))))
	}
	return html.Div(Class("v-player-menu-wrapper"), Attr("data-player-menu")("quality"))(
		Button(stimulus.Action("click->player#toggleQualityMenu"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/settings-04")),
		html.Div(
			stimulus.Target("player", "qualityMenu"),
			Class("v-player-menu v-player-quality-menu"),
		)(items...),
	)
}

// qualityLabels formats one display string per option. Each label
// starts with the resolution. FPS is appended to every rendition
// when the set has more than one distinct frame rate. Bitrate is
// appended within runs of adjacent renditions that would otherwise
// produce the same label, so users can tell same-resolution variants
// apart.
func qualityLabels(opts []model.QualityOption) []string {
	showFPS := fpsVaries(opts)
	out := make([]string, len(opts))
	for i, opt := range opts {
		out[i] = qualityBaseLabel(opt, showFPS)
	}
	for i := 0; i < len(out); {
		j := i + 1
		for j < len(out) && out[j] == out[i] {
			j++
		}
		if j-i > 1 {
			for k := i; k < j; k++ {
				out[k] += " — " + bitrateLabel(opts[k].TargetBitrate)
			}
		}
		i = j
	}
	return out
}

func qualityBaseLabel(opt model.QualityOption, showFPS bool) string {
	if opt.RenditionID == "" {
		return "Auto"
	}
	if showFPS {
		return fmt.Sprintf("%dp %dfps", opt.Height, opt.FPS)
	}
	return fmt.Sprintf("%dp", opt.Height)
}

// fpsVaries reports whether the encoded renditions in opts have more
// than one distinct FPS.
func fpsVaries(opts []model.QualityOption) bool {
	fps := map[int]bool{}
	for _, opt := range opts {
		if opt.RenditionID != "" {
			fps[opt.FPS] = true
		}
	}
	return len(fps) > 1
}

// bitrateLabel renders a kbit/s value as "X MB/s" or "X kB/s".
func bitrateLabel(kbps int) string {
	if kbps < 1000 {
		return fmt.Sprintf("%d kB/s", kbps)
	}
	return fmt.Sprintf("%.1f", float64(kbps)/1000) + " MB/s"
}

// autoLast returns a copy of opts with the Auto entry (RenditionID
// "") moved to the end. Other entries keep their relative order. The
// model returns Auto first; the player UI shows it at the bottom.
func autoLast(opts []model.QualityOption) []model.QualityOption {
	return slices.SortedStableFunc(slices.Values(opts), func(a, b model.QualityOption) int {
		if a.RenditionID == "" {
			return 1
		}
		if b.RenditionID == "" {
			return -1
		}
		return 0
	})
}

// playerCaptionsTemplate emits a <template> containing one <track>
// child per subtitle option. The JS clones it into <video> after the
// manifest has loaded if the HLS implementation didn't surface its
// SUBTITLES group via textTracks (Chrome's case today — see Chromium
// #383582114). When the manifest does surface them (Safari, Roku,
// AppleTV, future Chrome) the template stays unused and there are no
// duplicate TextTracks to deduplicate.
//
// The label attribute carries the SubtitleTrack ID, matching the HLS
// EXT-X-MEDIA NAME — both surface as TextTrack.label, and the player
// JS keys on it. Visible menu text comes from the menu's Text node.
func playerCaptionsTemplate(opts []model.SubtitleOption) domi.Node {
	var tracks []domi.Node
	for _, opt := range opts {
		tracks = append(tracks, html.Track(
			attr.Src(opt.WebVTTPath),
			Attr("srclang")(opt.Language),
			Attr("label")(opt.ID),
			Attr("kind")("subtitles"),
		))
	}
	if len(tracks) == 0 {
		return nil
	}
	return html.Template(stimulus.Target("player", "captionsTemplate"))(tracks...)
}

// playerCaptionsMenu mirrors playerQualityMenu: a popover menu over a
// settings-style button. Subtitle tracks come from either the HLS
// manifest (Safari, Roku, AppleTV) or the playerCaptionsTemplate
// fallback inserted by the JS (Chrome). The JS toggles TextTrack.mode
// to switch between them. The id param matches the HLS NAME (which
// is the SubtitleTrack ID) and the template's label attribute, and
// is what the JS uses to find the matching TextTrack.
//
// Each non-Off entry also carries the original codec and a URL to
// the original-format blob, so the JS can route ass/ssa picks to
// jassub when a jassub-tagged build provides the host URL. Default
// builds ignore those attributes and stay on the WebVTT path.
func playerCaptionsMenu(opts []model.SubtitleOption, initSubtitle string) domi.Node {
	items := []domi.Node{playerCaptionsItem("", "Off", "", "", initSubtitle)}
	for _, opt := range opts {
		items = append(items, playerCaptionsItem(opt.ID, opt.Label, opt.OriginalCodec, opt.OriginalPath, initSubtitle))
	}
	return html.Div(Class("v-player-menu-wrapper"), Attr("data-player-menu")("captions"))(
		Button(stimulus.Action("click->player#toggleCaptionsMenu"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/message-text-square-02")),
		html.Div(
			stimulus.Target("player", "captionsMenu"),
			Class("v-player-menu v-player-captions-menu"),
		)(items...),
	)
}

func playerCaptionsItem(id, label, codec, original, initSubtitle string) domi.Node {
	attrs := []domi.Attr{
		attr.Type("button"),
		stimulus.Action("click->player#setSubtitle"),
		Attr("data-player-sub-id-param")(id),
		Attr("data-player-sub-codec-param")(codec),
		Attr("data-player-sub-original-param")(original),
		Class("v-player-menu-item"),
	}
	if id == initSubtitle {
		attrs = append(attrs, Attr("data-active")(""))
	}
	return html.Button(attrs...)(LabelNode("line/check", TruncTail(label)))
}

// playerAudioMenu mirrors playerCaptionsMenu: a popover menu over a
// headphones-style button. Audio renditions are surfaced by the
// browser's native HLS implementation as HTMLMediaElement.audioTracks
// (Safari populates this from EXT-X-MEDIA AUDIO group; Chrome's
// AudioVideoTracks feature is currently disabled), so no template
// fallback is needed. The id param matches the HLS NAME — which is
// the AudioRendition ID — and is used by the JS to find the matching
// AudioTrack.
func playerAudioMenu(opts []model.AudioOption, initAudio string) domi.Node {
	if len(opts) == 0 {
		return nil
	}
	var items []domi.Node
	for _, opt := range opts {
		btnAttrs := []domi.Attr{
			attr.Type("button"),
			stimulus.Action("click->player#setAudio"),
			Attr("data-player-audio-id-param")(opt.ID),
			Class("v-player-menu-item"),
		}
		if opt.ID == initAudio {
			btnAttrs = append(btnAttrs, Attr("data-active")(""))
		}
		items = append(items, html.Button(btnAttrs...)(LabelNode("line/check", TruncTail(audioOptionLabel(opt)))))
	}
	return html.Div(Class("v-player-menu-wrapper"), Attr("data-player-menu")("audio"))(
		Button(
			stimulus.Action("click->player#toggleAudioMenu"),
			ButtonSurface, ButtonCircle, ButtonSize3,
		)(Icon("line/recording-01")),
		html.Div(
			stimulus.Target("player", "audioMenu"),
			Class("v-player-menu v-player-audio-menu"),
		)(items...),
	)
}

func playerSeekBar() domi.Node {
	return html.Div(Class("v-player-seek"))(
		html.Div(
			Class("v-player-seek-bar"),
			stimulus.Target("player", "progress"),

			// Seek tooltip.
			stimulus.Action("mouseenter->player#handleSeekTooltip"),
			stimulus.Action("mouseleave->player#handleSeekTooltip"),
			stimulus.Action("mousemove->player#handleSeekTooltip"),
		)(
			html.Input(
				attr.Type("range"),
				Attr("min")("0"),
				Attr("max")("100"),
				Attr("step")("0.01"),
				Attr("value")("0"),
				Attr("autocomplete")("off"),
				Attr("aria-label")("Seek"),
				Attr("style")("--value: 0%"),
				Attr("disabled")(""),
				stimulus.Target("player", "seek"),
				stimulus.Action("input->player#handleSeek"),

				// Set seek-value attribute for tooltip consistency.
				stimulus.Action("mousedown->player#handleSeekMouse"),
				stimulus.Action("mousemove->player#handleSeekMouse"),

				// Pause while seeking.
				stimulus.Action("mousedown->player#handleSeekPause"),
				stimulus.Action("mouseup->player#handleSeekPause"),
				stimulus.Action("keydown->player#handleSeekPause"),
				stimulus.Action("keyup->player#handleSeekPause"),
				stimulus.Action("touchstart->player#handleSeekPause"),
				stimulus.Action("touchend->player#handleSeekPause"),
				Class("v-player-seek-input"),
			),
			html.Progress(
				Attr("min")("0"),
				Attr("max")("100"),
				Attr("value")("0"),
				stimulus.Target("player", "buffer"),
				Class("v-player-buffer"),
			)(
				domi.Text("% buffered"),
			),
			html.Span(
				stimulus.Target("player", "seekTooltip"),
				Class("v-player-seek-tooltip"),
			)(
				domi.Text("00:00"),
			),
		),
	)
}

func playerVolumeBar() domi.Node {
	return FlexRow(Gap4, Class("v-player-volume-bar"))(
		Button(stimulus.Action("click->player#toggleMute"), ButtonSurface, ButtonCircle, ButtonSize3)(Icon("line/volume-max")),
		html.Div(Class("v-player-volume"))(
			html.Input(
				attr.Type("range"),
				Attr("min")("0"),
				Attr("max")("1"),
				Attr("step")("0.05"),
				Attr("value")("1"),
				Attr("autocomplete")("off"),
				Attr("aria-label")("Volume"),
				Attr("style")("--value: 100%"),
				stimulus.Target("player", "volume"),

				// Volume input.
				stimulus.Action("input->player#handleVolumeInput"),

				// Mouse wheel for volume.
				stimulus.Action("wheel->player#handleVolumeWheel"),

				Class("v-player-volume-input"),
			),
		),
	)
}
