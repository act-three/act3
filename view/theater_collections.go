package view

import (
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
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

func collectionBannerLink(c *model.CollectionHead) html.Node {
	return Box(HoverOverlay, Class("v-collection-banner"))(
		html.A(
			Class("v-collection-banner-link"),
			attr.Href(c.TheaterPath()),
		)(
			PosterImg(PosterFill, PosterAspect1000185, attr.Src(c.BannerPath())),
		),
		html.If(c.BannerID() == "", func() html.Node {
			return Text(c.Title(), Class("v-collection-title"))
		}),
	)
}
