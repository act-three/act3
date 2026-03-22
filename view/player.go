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

func PlayerForEpisode(v *model.Video, ep *model.Episode, qualityOpts []model.QualityOption) html.Node {
	return player(v, playerTitleForEpisode(ep), qualityOpts)
}

func PlayerForMovie(v *model.Video, med *model.MovieEditionHead, qualityOpts []model.QualityOption) html.Node {
	return player(v, playerTitleForMovie(med), qualityOpts)
}

func playerTitleForMovie(med *model.MovieEditionHead) string {
	title := med.Title()
	if y := med.Year(); y != 0 {
		return fmt.Sprintf("%s (%d)", title, y)
	}
	return title
}

func player(v *model.Video, title string, qualityOpts []model.QualityOption) html.Node {
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
					attr.Attr("playsinline"),
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
						attr.Src(v.PlaylistURL()),
						attr.Type("application/vnd.apple.mpegurl"),
					),
				),
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
						html.Div(Class("v-player-button-group"), attr.Attr("data-align")("start"))(
							Button(stimulus.Action("click->player#toggleCaptions"), ButtonSurface, ButtonCircle)(Icon("line/message-text-square-02")),
							Button(stimulus.Action("click->player#toggleAudioDesc"), ButtonSurface, ButtonCircle)(Icon("line/recording-01")),
							playerVolumeBar(),
						),

						html.Div(Class("v-player-button-group"), attr.Attr("data-align")("center"))(
							Button(stimulus.Action("click->player#skipBackward"), ButtonSurface, ButtonCircle)(Icon("line/refresh-ccw-01")),
							Button(stimulus.Action("click->player#togglePlay"), ButtonSurface, ButtonCircle)(Icon("solid/play")),
							Button(stimulus.Action("click->player#skipForward"), ButtonSurface, ButtonCircle)(Icon("line/refresh-cw-01")),
						),

						html.Div(Class("v-player-button-group"), attr.Attr("data-align")("end"))(
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
	if ep == nil {
		return "Episode " + ep.ID()
	}
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
				attr.Attr("data-player-url-param")(opt.URL),
				attr.Attr("data-player-label-param")(opt.Label),
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
				attr.Attr("min")("0"),
				attr.Attr("max")("100"),
				attr.Attr("step")("0.01"),
				attr.Attr("value")("0"),
				attr.Attr("autocomplete")("off"),
				attr.Attr("aria-label")("Seek"),
				attr.Attr("style")("--value: 0%"),
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
				attr.Attr("min")("0"),
				attr.Attr("max")("100"),
				attr.Attr("value")("0"),
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
				attr.Attr("min")("0"),
				attr.Attr("max")("1"),
				attr.Attr("step")("0.05"),
				attr.Attr("value")("1"),
				attr.Attr("autocomplete")("off"),
				attr.Attr("aria-label")("Volume"),
				attr.Attr("style")("--value: 100%"),
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
