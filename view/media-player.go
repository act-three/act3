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

func MediaPlayerForEpisode(v *model.Video, ep *model.Episode, qualityOpts []model.QualityOption) html.Node {
	return mediaPlayer(v, mediaPlayerTitleForEpisode(ep), qualityOpts)
}

func mediaPlayer(v *model.Video, title string, qualityOpts []model.QualityOption) html.Node {
	return turbo.Frame("player")(
		html.Div(
			attr.ID("full-player"),
			Class("fixed inset-0 bg-black tabular-nums"),
			stimulus.Controller("player"),
			stimulus.Value("player", "icon-url")(PlyrIconURL),
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
		)(
			html.Div(Class("absolute inset-0"))(
				html.Video(
					Class("w-full h-full"),
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
				Class("absolute inset-0"),
				stimulus.Target("player", "controls"),
				stimulus.Action("click->player#togglePlay:self"),
				stimulus.Action("focusin->player#handleControlsFocus"),
			)(
				FlexRow(Class(`
					overlay--top
					absolute
					left-0
					right-0
					top-0
					px-4
					pt-2
					pb-24
					items-center
					gap-4
				`))(
					Button(stimulus.Action("click->player#dismiss"))(Icon("x")),
					Box(Class("plyr__title"))(Text(title)),
				),
				FlexCol(
					Class(`
						overlay--bottom
						absolute
						left-0
						right-0
						bottom-0
						p-2
						pt-32
						gap-2
					`),
				)(
					FlexRow(Class(`
						px-4
						items-center
						gap-4
					`))(
						Box(stimulus.Target("player", "currentTime"))(Text("0:00")),
						mediaPlayerSeekBar(),
						Box(stimulus.Target("player", "duration"))(Text("0:00")),
					),
					FlexRow(Class(`
						px-4
						items-center
						gap-4
					`))(
						FlexRow(Class(`
							basis-1/3
							flex-initial
							items-center
							justify-start
							gap-4
						`))(
							Button(stimulus.Action("click->player#toggleCaptions"))(Icon("message-square-text")),
							Button(stimulus.Action("click->player#toggleAudioDesc"))(Icon("audio-lines")),
							mediaPlayerVolumeBar(),
						),

						FlexRow(Class(`
							basis-1/3
							flex-initial
							items-center
							justify-center
							gap-4
						`))(
							Button(stimulus.Action("click->player#skipBackward"))(Icon("rotate-ccw")),
							Button(stimulus.Action("click->player#togglePlay"))(Icon("play")),
							Button(stimulus.Action("click->player#skipForward"))(Icon("rotate-cw")),
						),

						FlexRow(Class(`
							basis-1/3
							flex-initial
							justify-end
							items-center
							gap-4
						`))(
							mediaPlayerQualityMenu(qualityOpts),
							Button(stimulus.Action("click->player#toggleFullscreen"))(Icon("maximize")),
						),
					),
				),
			),
		).
			With(ButtonLG).
			With(ButtonBordered).
			With(ButtonCircle),
	)
}

func mediaPlayerTitleForEpisode(ep *model.Episode) string {
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

func mediaPlayerQualityMenu(opts []model.QualityOption) html.Node {
	var items []html.Node
	for _, opt := range opts {
		items = append(items,
			html.Button(
				attr.Type("button"),
				stimulus.Action("click->player#setQuality"),
				attr.Attr("data-player-url-param")(opt.URL),
				attr.Attr("data-player-label-param")(opt.Label),
				Class(`
					w-full text-left px-3 py-1.5
					text-sm text-white/80 hover:text-white hover:bg-white/10
					cursor-pointer
					data-[active]:text-white data-[active]:font-bold
				`),
			)(Text(opt.Label)),
		)
	}
	return html.Div(Class("relative"))(
		Button(stimulus.Action("click->player#toggleQualityMenu"))(Icon("settings-2")),
		html.Div(
			stimulus.Target("player", "qualityMenu"),
			Class(`
				absolute bottom-full right-0 mb-2
				bg-black/80 backdrop-blur
				rounded-lg
				py-1
				min-w-[120px]
				hidden
			`),
		)(items...),
	)
}

func mediaPlayerSeekBar() html.Node {
	return html.Div(Class("flex-1 min-w-0"))(
		html.Div(
			Class("relative ml-[6.5px] mr-[13px]"),
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
				Class(`
					appearance-none bg-transparent border-0
					rounded-[26px] text-white block
					h-[19px] m-0 min-w-0 p-0
					w-[calc(100%+13px)] ml-[-6.5px] mr-[-6.5px]
					relative z-[2]
					transition-shadow duration-300 ease-out

					[&::-webkit-slider-runnable-track]:h-[5px]
					[&::-webkit-slider-runnable-track]:rounded-[2.5px]
					[&::-webkit-slider-runnable-track]:border-0
					[&::-webkit-slider-runnable-track]:bg-transparent
					[&::-webkit-slider-runnable-track]:[background-image:linear-gradient(to_right,currentColor_var(--value,0%),transparent_var(--value,0%))]
					[&::-webkit-slider-runnable-track]:select-none
					[&::-webkit-slider-runnable-track]:transition-shadow
					[&::-webkit-slider-runnable-track]:duration-300

					[&::-webkit-slider-thumb]:appearance-none
					[&::-webkit-slider-thumb]:bg-white
					[&::-webkit-slider-thumb]:border-0
					[&::-webkit-slider-thumb]:rounded-full
					[&::-webkit-slider-thumb]:shadow-[0_1px_1px_rgba(35,40,47,0.15),0_0_0_1px_rgba(35,40,47,0.2)]
					[&::-webkit-slider-thumb]:size-[13px]
					[&::-webkit-slider-thumb]:mt-[-4px]
					[&::-webkit-slider-thumb]:relative
					[&::-webkit-slider-thumb]:transition-all
					[&::-webkit-slider-thumb]:duration-200
					[&::-webkit-slider-thumb]:ease-out

					[&::-moz-range-track]:h-[5px]
					[&::-moz-range-track]:rounded-[2.5px]
					[&::-moz-range-track]:border-0
					[&::-moz-range-track]:bg-transparent
					[&::-moz-range-track]:select-none

					[&::-moz-range-thumb]:bg-white
					[&::-moz-range-thumb]:border-0
					[&::-moz-range-thumb]:rounded-full
					[&::-moz-range-thumb]:shadow-[0_1px_1px_rgba(35,40,47,0.15),0_0_0_1px_rgba(35,40,47,0.2)]
					[&::-moz-range-thumb]:size-[13px]
					[&::-moz-range-thumb]:relative
					[&::-moz-range-thumb]:transition-all
					[&::-moz-range-thumb]:duration-200
					[&::-moz-range-thumb]:ease-out

					[&::-moz-range-progress]:bg-current
					[&::-moz-range-progress]:rounded-[2.5px]
					[&::-moz-range-progress]:h-[5px]
				`),
			),
			html.Progress(
				attr.Attr("min")("0"),
				attr.Attr("max")("100"),
				attr.Attr("value")("0"),
				stimulus.Target("player", "buffer"),
				Class(`
					appearance-none
					bg-white/25 text-white/25
					border-0 rounded-full
					h-[5px] left-0 -mt-[2.5px] p-0
					absolute top-1/2
					w-[calc(100%+13px)] ml-[-6.5px] mr-[-6.5px]

					[&::-webkit-progress-bar]:bg-transparent
					[&::-webkit-progress-value]:bg-current
					[&::-webkit-progress-value]:rounded-full
					[&::-webkit-progress-value]:min-w-[5px]
					[&::-webkit-progress-value]:transition-[width]
					[&::-webkit-progress-value]:duration-200
					[&::-webkit-progress-value]:ease-out

					[&::-moz-progress-bar]:bg-current
					[&::-moz-progress-bar]:rounded-full
					[&::-moz-progress-bar]:min-w-[5px]
					[&::-moz-progress-bar]:transition-[width]
					[&::-moz-progress-bar]:duration-200
					[&::-moz-progress-bar]:ease-out
				`),
			)(
				html.Text("% buffered"),
			),
			html.Span(
				stimulus.Target("player", "seekTooltip"),
				Class(`
					absolute bottom-full left-0
					mb-2 px-2 py-1
					bg-white text-gray-12 text-xs
					rounded shadow
					pointer-events-none
					opacity-0 transition-opacity duration-150
					whitespace-nowrap
					-translate-x-1/2
				`),
			)(
				html.Text("00:00"),
			),
		),
	)
}

func mediaPlayerVolumeBar() html.Node {
	return FlexRow(Class("items-center gap-2"))(
		Button(stimulus.Action("click->player#toggleMute"))(Icon("volume-2")),
		html.Div(Class("w-20"))(
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

					// Volume input.
					stimulus.Action("input->player#handleVolumeInput"),

					// Mouse wheel for volume.
					stimulus.Action("wheel->player#handleVolumeWheel"),

					Class(`
						appearance-none bg-transparent border-0
						rounded-[26px] text-white block
						h-[19px] m-0 min-w-0 p-0
						w-full
						relative z-[2]
						cursor-pointer
						transition-shadow duration-300 ease-out

						[&::-webkit-slider-runnable-track]:h-[5px]
						[&::-webkit-slider-runnable-track]:rounded-[2.5px]
						[&::-webkit-slider-runnable-track]:border-0
						[&::-webkit-slider-runnable-track]:bg-white/20
						[&::-webkit-slider-runnable-track]:[background-image:linear-gradient(to_right,currentColor_var(--value,0%),transparent_var(--value,0%))]
						[&::-webkit-slider-runnable-track]:select-none
						[&::-webkit-slider-runnable-track]:transition-shadow
						[&::-webkit-slider-runnable-track]:duration-300

						[&::-webkit-slider-thumb]:appearance-none
						[&::-webkit-slider-thumb]:bg-white
						[&::-webkit-slider-thumb]:border-0
						[&::-webkit-slider-thumb]:rounded-full
						[&::-webkit-slider-thumb]:shadow-[0_1px_1px_rgba(35,40,47,0.15),0_0_0_1px_rgba(35,40,47,0.2)]
						[&::-webkit-slider-thumb]:size-[13px]
						[&::-webkit-slider-thumb]:mt-[-4px]
						[&::-webkit-slider-thumb]:relative
						[&::-webkit-slider-thumb]:transition-all
						[&::-webkit-slider-thumb]:duration-200
						[&::-webkit-slider-thumb]:ease-out

						[&::-moz-range-track]:h-[5px]
						[&::-moz-range-track]:rounded-[2.5px]
						[&::-moz-range-track]:border-0
						[&::-moz-range-track]:bg-white/20
						[&::-moz-range-track]:select-none

						[&::-moz-range-thumb]:bg-white
						[&::-moz-range-thumb]:border-0
						[&::-moz-range-thumb]:rounded-full
						[&::-moz-range-thumb]:shadow-[0_1px_1px_rgba(35,40,47,0.15),0_0_0_1px_rgba(35,40,47,0.2)]
						[&::-moz-range-thumb]:size-[13px]
						[&::-moz-range-thumb]:relative
						[&::-moz-range-thumb]:transition-all
						[&::-moz-range-thumb]:duration-200
						[&::-moz-range-thumb]:ease-out

						[&::-moz-range-progress]:bg-current
						[&::-moz-range-progress]:rounded-[2.5px]
						[&::-moz-range-progress]:h-[5px]
					`),
				),
			),
		),
	)
}
