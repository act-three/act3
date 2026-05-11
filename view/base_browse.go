package view

import (
	crand "crypto/rand"
	"math/rand/v2"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/web/static"
)

func browse(title string, washImages ...model.Image) html.Element {
	return func(child ...html.Node) html.Node {
		return base(title)(Style("padding:var(--nav-h) 0 8rem"))(
			browseWash(washImages),
			browseContainer(child...),
			browseNavigationMenu(),
			turbo.Frame("player"),
		)
	}
}

func browseContainer(child ...html.Node) html.Node {
	return FlexCol(Class("v-media-container"))(
		Group(child...),
	)
}

func browseWash(images []model.Image) html.Node {
	var urls []string
	for _, im := range images {
		if im.IsPlaceholder() {
			continue
		}
		urls = append(urls, im.SmallestURL())
	}
	url := static.Path("/static/cb.jpeg")
	if len(urls) > 0 {
		url = urls[rand.IntN(len(urls))]
	}
	return Box(Class("v-media-wash"))(
		PosterImg(AspectPoster, PosterFill, attr.Src(url)),
	)
}

func browseDownloadButton(dls []*model.RenditionForDownload) html.Node {
	id := "dl-" + crand.Text()[:8]
	anchor := "--" + id
	return FlexCol()(
		Button(ButtonGhost, ButtonSize3, ButtonCircle,
			Disabled(len(dls) == 0),
			attr.Popovertarget(id),
			Style("anchor-name:"+anchor),
		)(Icon("line/download-01")),
		html.Div(
			attr.ID(id),
			attr.Popover("auto"),
			Class("u-menu"),
			Style("position-anchor:"+anchor),
			Style("top:anchor(bottom)"),
			Style("left:anchor(center)"),
			Style("translate:-50% 0"),
		)(
			html.Range(dls, func(dl *model.RenditionForDownload) html.Node {
				return html.A(
					Href(dl.Path()),
					Class("u-menu-item"),
				)(Text(dl.Label(), Size3))
			}),
		),
	)
}

func browseNavigationMenu() html.Node {
	return FlexCol(
		Class("v-media-nav"),
	)(
		browseContainer(
			Grid12()(
				FlexRow(
					ColSpan12,
					Gap8,
					Class("v-media-nav-row"),
				)(
					Link("/")(wordmark()),
					Button(Href("/collections"), ButtonGhost)(Text("Collections")),
					Box(Class("v-media-nav-spacer")),
					html.If(isUserAdmin(), func() html.Node {
						return Button(Href("/app/profile"), ButtonGhost, ButtonCircle)(Icon("line/settings-01"))
					}),
				),
			),
		),
	)
}
