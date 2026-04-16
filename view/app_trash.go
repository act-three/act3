package view

import (
	"fmt"
	"time"

	"ily.dev/act3/expr"
	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
)

const AppTrashListItems = "trash-list-items"

func AppTrash(
	title string,
	items []model.TrashItem,
	detail ...html.Node,
) (string, html.Node) {
	return title, FlexCol(attr.Class("v-media-page"))(
		Split()(
			List("/app/trash/", "detail")(
				Box(
					Size2,
					attr.Style("background: var(--bg-surface)"),
					attr.Style("border-bottom: 0.5px solid var(--border-strong)"),
					attr.Style("margin: -0.5rem -0.5rem 1rem"),
					attr.Style("padding: 1rem"),
				)(
					Label("line/info-circle", "Items in Trash are permanently deleted after 30 days.",
						TextBalance,
					),
				),
				turbo.StreamTarget(AppTrashListItems)(
					ListItems(items, AppTrashListItem),
				),
			),
			turbo.Frame("detail", turbo.Advance())(
				expr.IfElse(detail != nil,
					func() html.Node {
						return Group(detail...)
					},
					func() html.Node {
						return Center(Class("v-media-muted"))(
							html.Text("No Item Selected"),
						)
					},
				),
			),
		),
	)
}

func AppTrashListItem(it model.TrashItem, attrs ...attr.Node) html.Node {
	i2 := trashKindIcon2(it.Kind)
	return Card(CardGhost,
		attr.Group(attrs...),
		ListID(it.ID),
		ListURL(appTrashDetailPath(it.ID)),
	)(
		CardContent()(
			FlexRow(Gap2, attr.Style("align-items:baseline"))(
				FlexCol(Gap1, attr.Style("flex-shrink:0"))(
					Icon(trashKindIcon(it.Kind)),
					html.If(i2 != "", func() html.Node {
						return Icon(i2)
					}),
				),
				FlexCol(Gap1)(
					CardTitle()(Text(trashItemTitle(it))),
					html.If(it.Subtitle != "", func() html.Node {
						return CardDescription(LineClamp1)(html.Text(it.Subtitle))
					}),
					CardDescription(LineClamp2)(
						html.Text(relativeTime(it.DeletedAt)),
					),
				),
			),
		),
	)
}

func AppTrashDetail(it model.TrashItem) html.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(html.Text(trashItemTitle(it))),
						html.If(it.Subtitle != "", func() html.Node {
							return Text(it.Subtitle, Size3,
								Class("u-settings-label-description"),
							)
						}),
						Text(trashKindLabel(it.Kind), Size3,
							Class("u-settings-label-description"),
						),
						Text(relativeTime(it.DeletedAt), Size2,
							Class("u-settings-label-description"),
						),
					),

					SettingsGroup()(
						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Restore"),
								SettingsItemLabelDescription("Return this item to the library"),
							),
							ActionButton("/-/do/restore", trashParams(it.ID),
								ButtonGhost, ButtonSize2,
							)(Text("Restore")),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Purge"),
								SettingsItemLabelDescription("Permanently delete this item"),
							),
							ActionButton("/-/do/purge", trashParams(it.ID),
								Destructive, ButtonGhost, ButtonSize2,
							)(Text("Purge")),
						),
					),
				),
			),
		),
	)
}

func appTrashDetailPath(id string) string { return "/app/trash/" + id }

// TrashListAppend is a Turbo Stream fragment that inserts a newly
// trashed item at the top of the trash list.
func TrashListAppend(item *model.TrashItem) html.Node {
	if item == nil {
		return html.Group()
	}
	return turbo.Prepend(AppTrashListItems,
		ListItems([]model.TrashItem{*item}, AppTrashListItem),
	)
}

// TrashListRemove is a Turbo Stream fragment that removes an item
// from the trash list (fired after restore or purge).
func TrashListRemove(id string) html.Node {
	return turbo.RemoveTargets(`[data-list-id-param="` + id + `"]`)
}

func trashForm(id string) html.Node {
	return ActionButton("/-/do/trash", trashParams(id),
		Destructive, ButtonGhost, ButtonSize2,
	)(Text("Delete"))
}

func trashParams(id string) map[string]string {
	return map[string]string{"id": id}
}

// MediaListRemove is a Turbo Stream fragment that removes a newly
// trashed entity from its parent list page (All Movies, All Series,
// All Collections). Sub-containers (editions, seasons, episodes,
// videos) don't have their own top-level list page; callers relying
// on them to refresh a detail frame should emit their own updates.
func MediaListRemove(kind model.TrashKind, id string) html.Node {
	switch kind {
	case model.TrashKindMovie, model.TrashKindSeries, model.TrashKindCollection:
		return turbo.RemoveTargets(`[data-list-id-param="` + id + `"]`)
	}
	return html.Group()
}

// MoviesListAppend emits a Turbo stream that re-appends a restored
// movie to the All Movies list page.
func MoviesListAppend(mw *model.MovieWork) html.Node {
	return turbo.Append(AppMoviesListItems,
		ListItems([]*model.MovieWork{mw}, AppMoviesListItem),
	)
}

// SeriesListAppend emits a Turbo stream that re-appends a restored
// series to the All Series list page.
func SeriesListAppend(sw *model.SeriesWork) html.Node {
	return turbo.Append(AppSeriesListItems,
		ListItems([]*model.SeriesWork{sw}, AppSeriesListItem),
	)
}

// CollectionsListAppend emits a Turbo stream that re-appends a
// restored collection to the All Collections list page.
func CollectionsListAppend(col *model.CollectionHead) html.Node {
	return turbo.Append(AppCollectionsListItems,
		ListItems([]*model.CollectionHead{col}, AppCollectionsListItem),
	)
}

func trashKindIcon(k model.TrashKind) string {
	switch k {
	case model.TrashKindMovie, model.TrashKindMovieEdition:
		return "line/film-01"
	case model.TrashKindSeries, model.TrashKindSeriesEdition:
		return "line/tv-03"
	case model.TrashKindSeason:
		return "line/ticket-02"
	case model.TrashKindEpisode:
		return "line/play-square"
	case model.TrashKindVideo:
		return "line/video-recorder"
	case model.TrashKindCollection:
		return "line/layers-three-01"
	}
	return "line/trash-01"
}

func trashKindIcon2(k model.TrashKind) string {
	switch k {
	case model.TrashKindMovieEdition, model.TrashKindSeriesEdition:
		return "line/clapperboard"
	}
	return ""
}

func trashKindLabel(k model.TrashKind) string {
	switch k {
	case model.TrashKindMovie:
		return "Movie"
	case model.TrashKindMovieEdition:
		return "Movie Edition"
	case model.TrashKindSeries:
		return "Series"
	case model.TrashKindSeriesEdition:
		return "Series Edition"
	case model.TrashKindSeason:
		return "Season"
	case model.TrashKindEpisode:
		return "Episode"
	case model.TrashKindVideo:
		return "Video"
	case model.TrashKindCollection:
		return "Collection"
	}
	return "Unknown"
}

func trashItemTitle(it model.TrashItem) string {
	if it.Title != "" {
		return it.Title
	}
	return it.ID
}

// relativeTime renders a past time as a coarse "N units ago" string.
func relativeTime(t time.Time) string {
	if t.IsZero() {
		return "recently"
	}
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return pluralize(int(d/time.Minute), "minute") + " ago"
	}
	if d < 24*time.Hour {
		return pluralize(int(d/time.Hour), "hour") + " ago"
	}
	return pluralize(int(d/(24*time.Hour)), "day") + " ago"
}

func pluralize(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", unit)
	}
	return fmt.Sprintf("%d %ss", n, unit)
}
