package view

import (
	"fmt"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

func PlayerForEpisode(v *model.Video, ep *model.Episode, qualityOpts []model.QualityOption, captionsOpts []model.SubtitleOption, audioOpts []model.AudioOption) html.Node {
	return player(v, playerTitleForEpisode(ep), qualityOpts, captionsOpts, audioOpts)
}

func PlayerForMovie(v *model.Video, med *model.MovieEditionHead, qualityOpts []model.QualityOption, captionsOpts []model.SubtitleOption, audioOpts []model.AudioOption) html.Node {
	return player(v, playerTitleForMovie(med), qualityOpts, captionsOpts, audioOpts)
}

func playerTitleForMovie(med *model.MovieEditionHead) string {
	title := med.Title()
	if y := med.Year(); y != "" {
		return fmt.Sprintf("%s (%s)", title, y)
	}
	return title
}

func player(v *model.Video, title string, qualityOpts []model.QualityOption, captionsOpts []model.SubtitleOption, audioOpts []model.AudioOption) html.Node {
	return turbo.Frame("player")(
		html.Div(
			attr.ID("full-player"),
			Class("v-player"),
			stimulus.Controller("player"),
			stimulus.Value("player", "title")(title),
			stimulus.Value("player", "playing")("false"),
			stimulus.Value("player", "paused")("false"),
			stimulus.Value("player", "stopped")("true"),
			stimulus.Value("player", "harlow")("false"),
			stimulus.Value("player", "hide-controls")("false"),
			stimulus.Value("player", "current-quality")("Auto"),
			stimulus.Value("player", "quality-menu-open")("false"),
			stimulus.Value("player", "current-subtitle")(""),
			stimulus.Value("player", "captions-menu-open")("false"),
			stimulus.Value("player", "current-audio")(""),
			stimulus.Value("player", "audio-menu-open")("false"),

			stimulus.Action("keydown.h@window->player#toggleHarlow"),
			stimulus.Action("keydown@window->player#handleKey"),
			stimulus.Action("keyup@window->player#handleKey"),

			stimulus.Action("mousemove->player#handleControls"),
			stimulus.Action("mouseleave->player#handleControls"),
			stimulus.Action("touchstart->player#handleControls"),
			stimulus.Action("touchmove->player#handleControls"),
			stimulus.Action("enterfullscreen->player#handleControls"),
			stimulus.Action("exitfullscreen->player#handleControls"),
		)(
			html.Div(Class("v-player-video-layer"))(
				html.Video(
					Class("v-player-video"),
					Attr("playsinline"),
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

					// Display duration.
					stimulus.Action("durationchange->player#handleDuration"),
					stimulus.Action("loadeddata->player#handleDuration"),
					stimulus.Action("loadedmetadata->player#handleDuration"),

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
						attr.Src(v.PlaylistPath()),
						attr.Type("application/vnd.apple.mpegurl"),
					),
				),
				playerCaptionsTemplate(captionsOpts),
			),
			html.Div(
				Class("v-player-controls"),
				stimulus.Target("player", "controls"),
				stimulus.Action("click->player#togglePlay:self"),
				stimulus.Action("focusin->player#handleControlsFocus"),
			)(
				html.Div(Class("v-player-overlay-top"))(
					Button(stimulus.Action("click->player#dismiss"), ButtonSurface, ButtonCircle)(Icon("line/x-close")),
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
							playerCaptionsMenu(captionsOpts),
							playerAudioMenu(audioOpts),
							Button(stimulus.Action("click->player#toggleAudioDesc"), ButtonSurface, ButtonCircle)(Icon("line/recording-01")),
							playerVolumeBar(),
						),

						html.Div(Class("v-player-button-group"), Attr("data-align")("center"))(
							Button(stimulus.Action("click->player#skipBackward"), ButtonSurface, ButtonCircle)(Icon("line/refresh-ccw-01")),
							Button(stimulus.Action("click->player#togglePlay"), ButtonSurface, ButtonCircle)(Icon("solid/play")),
							Button(stimulus.Action("click->player#skipForward"), ButtonSurface, ButtonCircle)(Icon("line/refresh-cw-01")),
						),

						html.Div(Class("v-player-button-group"), Attr("data-align")("end"))(
							playerQualityMenu(qualityOpts),
							Button(stimulus.Action("click->player#toggleFullscreen"), ButtonSurface, ButtonCircle)(Icon("line/maximize-02")),
						),
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

func playerQualityMenu(opts []model.QualityOption) html.Node {
	var items []html.Node
	for _, opt := range opts {
		items = append(items,
			html.Button(
				attr.Type("button"),
				stimulus.Action("click->player#setQuality"),
				Attr("data-player-url-param")(opt.Path),
				Attr("data-player-label-param")(opt.Label),
				Class("v-player-quality-option"),
			)(Text(opt.Label)),
		)
	}
	return html.Div(Class("v-player-quality-wrapper"))(
		Button(stimulus.Action("click->player#toggleQualityMenu"), ButtonSurface, ButtonCircle)(Icon("line/settings-04")),
		html.Div(
			stimulus.Target("player", "qualityMenu"),
			Class("v-player-quality-menu"),
		)(items...),
	)
}

// playerCaptionsTemplate emits a <template> containing one <track>
// child per subtitle option. The JS clones it into <video> after the
// manifest has loaded if the HLS implementation didn't surface its
// SUBTITLES group via textTracks (Chrome's case today — see Chromium
// #383582114). When the manifest does surface them (Safari, Roku,
// AppleTV, future Chrome) the template stays unused and there are no
// duplicate TextTracks to deduplicate.
func playerCaptionsTemplate(opts []model.SubtitleOption) html.Node {
	var tracks []html.Node
	for _, opt := range opts {
		tracks = append(tracks, html.Track(
			attr.Src(opt.WebVTTPath),
			Attr("srclang")(opt.Language),
			Attr("label")(opt.Label),
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
// to switch between them. The label param is how the JS finds the
// matching TextTrack — it must equal the manifest NAME (and the
// template's label attribute) for the same row.
func playerCaptionsMenu(opts []model.SubtitleOption) html.Node {
	items := []html.Node{
		html.Button(
			attr.Type("button"),
			stimulus.Action("click->player#setSubtitle"),
			Attr("data-player-sub-id-param")(""),
			Class("v-player-quality-option"),
			Attr("data-active")(""),
		)(Text("Off")),
	}
	for _, opt := range opts {
		items = append(items,
			html.Button(
				attr.Type("button"),
				stimulus.Action("click->player#setSubtitle"),
				Attr("data-player-sub-id-param")(opt.ID),
				Attr("data-player-sub-label-param")(opt.Label),
				Class("v-player-quality-option"),
			)(Text(opt.Label)),
		)
	}
	return html.Div(Class("v-player-quality-wrapper"))(
		Button(stimulus.Action("click->player#toggleCaptionsMenu"), ButtonSurface, ButtonCircle)(Icon("line/message-text-square-02")),
		html.Div(
			stimulus.Target("player", "captionsMenu"),
			Class("v-player-captions-menu"),
		)(items...),
	)
}

// playerAudioMenu mirrors playerCaptionsMenu: a popover menu over a
// headphones-style button. Audio renditions are surfaced by the
// browser's native HLS implementation as HTMLMediaElement.audioTracks
// (Safari populates this from EXT-X-MEDIA AUDIO group; Chrome's
// AudioVideoTracks feature is currently disabled), so no template
// fallback is needed. The id param matches the HLS NAME — which is
// the AudioRendition ID — and is used by the JS to find the matching
// AudioTrack. Unlike captions there is no "off" — every video has audio.
func playerAudioMenu(opts []model.AudioOption) html.Node {
	if len(opts) == 0 {
		return nil
	}
	var items []html.Node
	for _, opt := range opts {
		display := opt.Title + " (" + model.OutputChannelLabel(opt.Channels) + ")"
		btnAttrs := []attr.Node{
			attr.Type("button"),
			stimulus.Action("click->player#setAudio"),
			Attr("data-player-audio-id-param")(opt.ID),
			Class("v-player-quality-option"),
		}
		if opt.Default {
			btnAttrs = append(btnAttrs, Attr("data-active")(""))
		}
		items = append(items, html.Button(btnAttrs...)(Text(display)))
	}
	return html.Div(Class("v-player-audio-wrapper"))(
		Button(
			stimulus.Action("click->player#toggleAudioMenu"),
			ButtonSurface, ButtonCircle,
		)(Icon("line/headphones-01")),
		html.Div(
			stimulus.Target("player", "audioMenu"),
			Class("v-player-audio-menu"),
		)(items...),
	)
}

func playerSeekBar() html.Node {
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
				html.Text("% buffered"),
			),
			html.Span(
				stimulus.Target("player", "seekTooltip"),
				Class("v-player-seek-tooltip"),
			)(
				html.Text("00:00"),
			),
		),
	)
}

func playerVolumeBar() html.Node {
	return FlexRow(Gap2, Class("v-player-volume-bar"))(
		Button(stimulus.Action("click->player#toggleMute"), ButtonSurface, ButtonCircle)(Icon("line/volume-max")),
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
