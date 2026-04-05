package view

import (
	"fmt"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/stimulus"
	"ily.dev/act3/ui/turbo"
)

func Collections(cols []*model.CollectionHead) html.Node {
	var washURLs []string
	for _, c := range cols {
		washURLs = append(washURLs, c.BannerPath())
	}
	return browse("Collections", washURLs...)(
		FlexCol(attr.Style("padding-top:1rem"))(
			html.Range(cols, collectionBannerLink),
		),
	)
}

func TheaterCollection(c *model.Collection, itemCount, runtimeMinutes int64) html.Node {
	return browse("Collections", c.BannerPath())(
		FlexCol(
			Gap8,
			attr.Style("padding-top:1rem"),
			stimulus.Controller("collection"),
			stimulus.Value("collection", "mode")("overview"),
		)(
			collectionBanner(&c.CollectionHead),
			FlexRow(attr.Style("justify-content:space-between;align-items:baseline"))(
				FlexRow(Gap4, ButtonSize3)(
					collectionTabButton(
						"/-/part/collection-overview/"+c.ID(),
						"Overview",
						stimulus.Action("click->collection#setOverview"),
						stimulus.Target("collection", "overview"),
						attr.Attr("data-selected"),
					),
					collectionTabButton(
						"/-/part/collection-playlist/"+c.ID(),
						"Playlist",
						stimulus.Action("click->collection#setPlaylist"),
						stimulus.Target("collection", "playlist"),
					),
				),
				collectionStatus(itemCount, runtimeMinutes),
			),
			turbo.Frame("collection-content",
				stimulus.Target("collection", "frame"),
				turbo.FrameOption(attr.Attr("target")("_top")),
			)(
				TheaterCollectionOverview(c),
			),
		),
	)
}

func TheaterCollectionOverview(c *model.Collection) html.Node {
	return posterGrid(c.Works())
}

func TheaterCollectionPlaylist(c *model.Collection, ps []model.Playable) html.Node {
	return FlexCol(Class("v-collection-playlist"))(
		html.Range(ps, theaterCollectionPlayable),
	)
}

const imageWidth = 8 * 16 // px

func theaterCollectionPlayable(p model.Playable) html.Node {
	n, d := p.ImageAspect()
	h := imageWidth * d / n
	return FlexRow(
		Gap4,
		Class("v-collection-playlist-row"),
		attr.Stylef("height:%dpx", h),
	)(
		html.A(attr.Href(p.TheaterPath()), Class("v-collection-playlist-row-link")),
		theaterCollectionPlayableImage(p, h),
		playButtonForList(p),
		FlexCol(Class("v-collection-playlist-text"), Size3)(
			Text(p.Title(), Class("v-collection-playlist-title")),
			FlexRow(Gap2)(
				html.Range(p.Info(), func(s string) html.Node {
					return Text(s, Class("v-collection-playlist-info"))
				}),
			),
		),
		Text(p.ReleaseDate(), Class("v-collection-playlist-release-date")),
		Text(p.Runtime()+"m"),
		Button(ButtonGhost, ButtonCircle)(Icon("line/check")), // placeholder for watched status & button
	)
}

func theaterCollectionPlayableImage(p model.Playable, h int) html.Node {
	return html.Img(
		Class("v-collection-playlist-row-image"),
		attr.Stylef("width:%dpx", imageWidth),
		attr.Stylef("height:%dpx", h),
		attr.Src(p.ImagePath()),
	)
}

func collectionStatus(itemCount, runtimeMinutes int64) html.Node {
	s := fmt.Sprintf("%d episodes & movies, %dm", itemCount, runtimeMinutes)
	return Text(s, attr.Style("padding-right:0.25rem"))
}

func collectionTabButton(url, label string, attrs ...attr.Node) html.Node {
	return Group(
		Button(
			attr.Attr("data-url")(url),
			stimulus.Action("mouseenter->collection#prefetch"),
			attr.Group(attrs...),
		)(Text(label)),
		// This exists only so that we can dispatch a synthetic mouseenter
		// event to cause Turbo to prefetch the correct URL on hover.
		html.A(
			attr.Href(url),
			turbo.DataFrame("collection-content"),
			stimulus.Target("collection", "prefetch"),
			attr.Style("display:none"),
		),
	)
}

func collectionBannerLink(c *model.CollectionHead) html.Node {
	return Box(HoverOverlay, Class("v-collection-banner-x"))(
		html.A(
			Class("v-collection-banner-link"),
			attr.Href(c.TheaterPath()),
		)(
			collectionBanner(c),
		),
	)
}

func collectionBanner(c *model.CollectionHead) html.Node {
	return Box(Class("v-collection-banner"))(
		PosterImg(PosterFill, PosterAspect1000185, attr.Src(c.BannerPath())),
		html.If(c.BannerID() == "", func() html.Node {
			return Text(c.Title(), Class("v-collection-title"))
		}),
	)
}
