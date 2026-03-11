package ui

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// PlayerQualityOption is a quality/rendition choice for the player menu.
type PlayerQualityOption struct {
	Label string // e.g. "Auto", "1080p", "720p"
	URL   string // playlist URL
}

// Player renders a full-screen media player with HLS playback,
// seek/volume controls, and an optional quality menu.
func Player(sourceURL, sourceType, title string, qualityOpts []PlayerQualityOption, attrs ...attr.Node) html.Node {
	return html.Div(
		attr.Class("u-player"),
		stimulus.Controller("player"),
		stimulus.Value("player", "title")(title),
		stimulus.Value("player", "playing")("false"),
		stimulus.Value("player", "paused")("false"),
		stimulus.Value("player", "stopped")("true"),
		stimulus.Value("player", "harlow")("false"),
		stimulus.Value("player", "hide-controls")("false"),
		stimulus.Value("player", "current-quality")("Auto"),

		stimulus.Action("keydown.h@window->player#toggleHarlow"),
		stimulus.Action("keydown@window->player#handleKey"),
		stimulus.Action("keyup@window->player#handleKey"),

		stimulus.Action("mousemove->player#handleControls"),
		stimulus.Action("mouseleave->player#handleControls"),
		stimulus.Action("touchstart->player#handleControls"),
		stimulus.Action("touchmove->player#handleControls"),
		stimulus.Action("enterfullscreen->player#handleControls"),
		stimulus.Action("exitfullscreen->player#handleControls"),

		group(attrs...),
	)(
		html.Div(attr.Class("u-player-video-area"))(
			html.Video(
				attr.Class("u-player-video"),
				attr.Attr("playsinline"),
				stimulus.Target("player", "video"),

				stimulus.Action("playing->player#handlePlaying"),
				stimulus.Action("play->player#handlePlaying"),
				stimulus.Action("pause->player#handlePlaying"),
				stimulus.Action("ended->player#handlePlaying"),
				stimulus.Action("emptied->player#handlePlaying"),
				stimulus.Action("timeupdate->player#handlePlaying"),

				stimulus.Action("timeupdate->player#handleTimeUpdate"),
				stimulus.Action("seeking->player#handleTimeUpdate"),
				stimulus.Action("seeked->player#handleTimeUpdate"),

				stimulus.Action("durationchange->player#handleDuration"),
				stimulus.Action("loadeddata->player#handleDuration"),
				stimulus.Action("loadedmetadata->player#handleDuration"),

				stimulus.Action("progress->player#handleProgress"),
				stimulus.Action("playing->player#handleProgress"),
				stimulus.Action("seeking->player#handleProgress"),
				stimulus.Action("seeked->player#handleProgress"),

				stimulus.Action("volumechange->player#handleVolume"),

				stimulus.Action("waiting->player#handleLoading"),
				stimulus.Action("canplay->player#handleLoading"),
				stimulus.Action("seeked->player#handleLoading"),
				stimulus.Action("playing->player#handleLoading"),

				stimulus.Action("ratechange->player#handleRate"),

				stimulus.Action("contextmenu->player#handleContextMenu"),
			)(
				html.Source(
					attr.Src(sourceURL),
					attr.Type(sourceType),
				),
			),
		),
		html.Div(
			attr.Class("u-player-controls"),
			stimulus.Target("player", "controls"),
			stimulus.Action("click->player#togglePlay:self"),
			stimulus.Action("focusin->player#handleControlsFocus"),
		)(
			playerOverlayTop(title),
			playerOverlayBottom(qualityOpts),
		),
	)
}

func playerOverlayTop(title string) html.Node {
	return FlexRow(Class("u-player-overlay-top items-center gap-4"))(
		Button(stimulus.Action("click->player#dismiss"), ButtonSurface, ButtonCircle)(Icon("line/x-close")),
		Box(Class("u-player-title"))(Text(title)),
	)
}

func playerOverlayBottom(qualityOpts []PlayerQualityOption) html.Node {
	return FlexCol(Class("u-player-overlay-bottom gap-2"))(
		FlexRow(Class("u-player-bar items-center gap-4"))(
			Box(stimulus.Target("player", "currentTime"))(Text("0:00")),
			playerSeekBar(),
			Box(stimulus.Target("player", "duration"))(Text("0:00")),
		),
		FlexRow(Class("u-player-bar items-center gap-4"))(
			FlexRow(Class("u-player-buttons-start items-center gap-4"))(
				Button(stimulus.Action("click->player#toggleCaptions"), ButtonSurface, ButtonCircle)(Icon("line/message-text-square-02")),
				Button(stimulus.Action("click->player#toggleAudioDesc"), ButtonSurface, ButtonCircle)(Icon("line/recording-01")),
				playerVolumeBar(),
			),

			FlexRow(Class("u-player-buttons-center items-center gap-4"))(
				Button(stimulus.Action("click->player#skipBackward"), ButtonSurface, ButtonCircle)(Icon("line/refresh-ccw-01")),
				Button(stimulus.Action("click->player#togglePlay"), ButtonSurface, ButtonCircle)(Icon("solid/play")),
				Button(stimulus.Action("click->player#skipForward"), ButtonSurface, ButtonCircle)(Icon("line/refresh-cw-01")),
			),

			FlexRow(Class("u-player-buttons-end items-center gap-4"))(
				playerQualityMenu(qualityOpts),
				Button(stimulus.Action("click->player#toggleFullscreen"), ButtonSurface, ButtonCircle)(Icon("line/maximize-02")),
			),
		),
	)
}

func playerQualityMenu(opts []PlayerQualityOption) html.Node {
	var items []html.Node
	for _, opt := range opts {
		items = append(items,
			html.Button(
				attr.Type("button"),
				stimulus.Action("click->player#setQuality"),
				attr.Attr("data-player-url-param")(opt.URL),
				attr.Attr("data-player-label-param")(opt.Label),
				attr.Class("u-player-quality-option"),
			)(Text(opt.Label)),
		)
	}
	return html.Div(Class("relative"))(
		Button(stimulus.Action("click->player#toggleQualityMenu"), ButtonSurface, ButtonCircle)(Icon("line/settings-04")),
		html.Div(
			stimulus.Target("player", "qualityMenu"),
			attr.Class("u-player-quality-menu hidden"),
		)(items...),
	)
}

func playerSeekBar() html.Node {
	return html.Div(attr.Class("u-player-seek"))(
		html.Div(
			attr.Class("u-player-seek-track"),
			stimulus.Target("player", "progress"),

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

				stimulus.Action("mousedown->player#handleSeekMouse"),
				stimulus.Action("mousemove->player#handleSeekMouse"),

				stimulus.Action("mousedown->player#handleSeekPause"),
				stimulus.Action("mouseup->player#handleSeekPause"),
				stimulus.Action("keydown->player#handleSeekPause"),
				stimulus.Action("keyup->player#handleSeekPause"),
				stimulus.Action("touchstart->player#handleSeekPause"),
				stimulus.Action("touchend->player#handleSeekPause"),
				attr.Class("u-player-range-input"),
			),
			html.Progress(
				attr.Attr("min")("0"),
				attr.Attr("max")("100"),
				attr.Attr("value")("0"),
				stimulus.Target("player", "buffer"),
				attr.Class("u-player-seek-buffer"),
			)(
				html.Text("% buffered"),
			),
			html.Span(
				stimulus.Target("player", "seekTooltip"),
				attr.Class("u-player-seek-tooltip"),
			)(
				html.Text("00:00"),
			),
		),
	)
}

func playerVolumeBar() html.Node {
	return FlexRow(Gap2, Class("items-center"))(
		Button(stimulus.Action("click->player#toggleMute"), ButtonSurface, ButtonCircle)(Icon("line/volume-max")),
		html.Div(attr.Class("u-player-volume"))(
			html.Div(Class("relative"))(
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

					stimulus.Action("input->player#handleVolumeInput"),
					stimulus.Action("wheel->player#handleVolumeWheel"),

					attr.Class("u-player-range-input"),
				),
			),
		),
	)
}
