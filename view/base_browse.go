package view

import (
	crand "crypto/rand"
	"math/rand/v2"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/web/static"
)

func browse(uploads []model.Upload, washImages ...model.Image) domi.Element {
	return func(child ...domi.Node) domi.Node {
		return domi.Fragment(
			browseWash(washImages),
			// The padding kept content clear of the fixed nav by
			// living on <body> pre-domi; domi owns <body> now.
			html.Div(Style("padding:var(--nav-h) 0 8rem"))(
				browseContainer(child...),
			),
			browseNavigationMenu(uploads),
		)
	}
}

func browseContainer(child ...domi.Node) domi.Node {
	return FlexCol(Class("v-media-container"))(
		Group(child...),
	)
}

func browseWash(images []model.Image) domi.Node {
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

func browseDownloadButton(dls []*model.RenditionForDownload) domi.Node {
	id := "dl-" + crand.Text()[:8]
	anchor := "--" + id
	return FlexCol()(
		Button(ButtonGhost, ButtonSize3, ButtonCircle,
			attr.Type("button"),
			Disabled(len(dls) == 0),
			attr.PopoverTarget(id),
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
			rangeNodes(dls, func(dl *model.RenditionForDownload) domi.Node {
				return html.A(
					Href(dl.Path()),
					attr.Download(""),
					Class("u-menu-item"),
				)(Text(dl.Label(), Size3))
			}),
		),
	)
}

// browseUploadProgress shows the oldest in-flight upload's progress
// in the theater nav, so an upload started in the editor stays
// visible while browsing.
func browseUploadProgress(uploads []model.Upload) domi.Node {
	if len(uploads) == 0 {
		return nil
	}
	return html.Div(
		Class("v-nav-upload-progress"),
	)(
		Progress(uploads[0].Frac),
	)
}

func browseNavigationMenu(uploads []model.Upload) domi.Node {
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
					ButtonLink("/collections", ButtonGhost)(Text("Collections")),
					Box(Class("v-media-nav-spacer")),
					browseUploadProgress(uploads),
					iff(isUserAdmin(), func() domi.Node {
						return ButtonLink("/app/profile", ButtonGhost, ButtonCircle)(Icon("line/settings-01"))
					}),
				),
			),
		),
	)
}
