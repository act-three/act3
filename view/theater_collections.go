package view

import (
	"fmt"

	"ily.dev/domi"
	"ily.dev/domi/html"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
)

func Collections(cols []*model.CollectionHead, uploads []model.Upload) (title string, n domi.Node) {
	var washImages []model.Image
	for _, c := range cols {
		washImages = append(washImages, c.Banner())
	}
	return "Collections", browse(uploads, washImages...)(
		FlexCol(Style("padding-top:1rem"))(
			rangeNodes(cols, func(c *model.CollectionHead) domi.Node {
				return collectionBannerLink(c)
			}),
		),
	)
}

// TheaterCollection renders the collection page with either the
// overview or the playlist tab selected; ps carries the playlist
// items when that tab is shown. The tabs are links: the playlist
// lives at its own URL under the collection's.
func TheaterCollection(c *model.Collection, itemCount, runtimeMinutes int64, playlist bool, ps []model.Playable, uploads []model.Upload) (title string, n domi.Node) {
	return c.Title(), browse(uploads, c.Banner())(
		FlexCol(
			Gap8,
			Style("padding-top:1rem"),
		)(
			collectionBanner(&c.CollectionHead),
			iff(isUserAdmin(), func() domi.Node {
				return FlexRow()(
					Link(
						c.EditorPath(),
					)(Text("View in Editor", Size3,
						Style("display: inline-block"),
					)),
				)
			}),

			FlexRow(Style("justify-content:space-between;align-items:baseline"))(
				FlexRow(Gap4, ButtonSize3)(
					collectionTabButton("Overview", c.TheaterPath(), !playlist),
					collectionTabButton("Playlist", c.PlaylistPath(), playlist),
				),
				collectionStatus(c, itemCount, runtimeMinutes),
			),
			expr.IfElse(playlist,
				func() domi.Node {
					return theaterCollectionPlaylist(ps)
				},
				func() domi.Node {
					return theaterCollectionOverview(c)
				},
			),
		),
	)
}

func theaterCollectionOverview(c *model.Collection) domi.Node {
	return posterGrid(c.Works())
}

func theaterCollectionPlaylist(ps []model.Playable) domi.Node {
	return FlexCol(Class("v-collection-playlist"))(
		rangeNodes(ps, theaterCollectionPlayable),
	)
}

const imageWidth = 8 * 16 // px

func theaterCollectionPlayable(p model.Playable) domi.Node {
	n, d := p.ImageAspect()
	h := imageWidth * d / n
	return FlexRow(
		Gap4,
		Class("v-collection-playlist-row"),
		Stylef("height:%dpx", h),
	)(
		html.A(Href(p.TheaterPath()), Class("v-collection-playlist-row-link")),
		theaterCollectionPlayableImage(p, h),
		playButtonForList(p),
		FlexCol(Class("v-collection-playlist-text"), Size3)(
			Text(p.Title(), Class("v-collection-playlist-title")),
			FlexRow(Gap2)(
				rangeNodes(p.Info(), func(s string) domi.Node {
					return Text(s, Class("v-collection-playlist-info"))
				}),
			),
		),
		Text(p.ReleaseDate(), Class("v-collection-playlist-release-date")),
		Text(p.Runtime()+"m"),
	)
}

func theaterCollectionPlayableImage(p model.Playable, h int) domi.Node {
	im, _ := p.ImageField()
	return html.Img(
		Class("v-collection-playlist-row-image"),
		Stylef("width:%dpx", imageWidth),
		Stylef("height:%dpx", h),
		imgAttrs(im),
	)
}

func collectionStatus(c *model.Collection, itemCount, runtimeMinutes int64) domi.Node {
	hasMovies := len(c.Movies()) > 0
	hasSeries := len(c.Series()) > 0
	var kind string
	switch {
	case hasMovies && hasSeries:
		kind = "episodes & movies"
	case hasMovies:
		kind = "movies"
	case hasSeries:
		kind = "episodes"
	default:
		kind = "episodes or movies"
	}
	s := fmt.Sprintf("%d %s, %dm", itemCount, kind, runtimeMinutes)
	return Text(s, Style("padding-right:0.25rem"))
}

func collectionTabButton(label, url string, selected bool) domi.Node {
	return ButtonLink(url,
		BoolAttr("data-selected", selected),
	)(Text(label))
}

func collectionBannerLink(c *model.CollectionHead, attrs ...domi.Attr) domi.Node {
	return Box(
		HoverOverlay,
		Class("v-collection-banner-x"),
		Attr("data-title")(c.Title()),
		group(attrs...),
	)(
		html.A(
			Class("v-collection-banner-link"),
			Href(c.TheaterPath()),
		)(
			collectionBanner(c),
		),
	)
}

func collectionBanner(c *model.CollectionHead) domi.Node {
	return Box(Class("v-collection-banner"))(
		PosterImg(AspectBanner, PosterFill, imgAttrs(c.Banner())),
		iff(c.Banner().IsPlaceholder(), func() domi.Node {
			return Text(c.Title(), Class("v-collection-title"))
		}),
	)
}
