package view

import (
	"fmt"
	"time"

	"ily.dev/domi"

	"ily.dev/act3/expr"
	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

func AppTrash(
	items []model.TrashItem,
	selected *model.TrashItem,
) (title string, n domi.Node) {
	title = "Trash"
	if selected != nil {
		title = trashItemTitle(*selected)
	}
	return title, FlexCol(Class("v-media-page"))(
		Split()(
			List()(
				Box(
					Size2,
					Style("background: var(--bg-surface)"),
					Style("border-bottom: 0.5px solid var(--border-strong)"),
					Style("margin: -0.5rem -0.5rem 1rem"),
					Style("padding: 1rem"),
				)(
					Label("line/info-circle", "Items in Trash are permanently deleted after 30 days.",
						TextBalance,
					),
				),
				ListItems(items, func(it model.TrashItem) bool {
					return selected != nil && it.ID == selected.ID
				}, AppTrashListItem),
			),
			expr.IfElse(selected != nil,
				func() domi.Node {
					return appTrashDetail(*selected)
				},
				func() domi.Node {
					return Center(Class("v-media-muted"))(
						domi.Text("No Item Selected"),
					)
				},
			),
		),
	)
}

func AppTrashListItem(it model.TrashItem, attrs ...domi.Attr) domi.Node {
	i2 := trashKindIcon2(it.Kind)
	return CardLink(appTrashDetailPath(it.ID), CardGhost,
		group(attrs...),
	)(
		CardContent()(
			FlexRow(Gap2, Style("align-items:baseline"))(
				FlexCol(Gap1, Style("flex-shrink:0"))(
					Icon(trashKindIcon(it.Kind)),
					iff(i2 != "", func() domi.Node {
						return Icon(i2)
					}),
				),
				FlexCol(Gap1)(
					CardTitle()(Text(trashItemTitle(it))),
					iff(it.Subtitle != "", func() domi.Node {
						return CardDescription(LineClamp1)(domi.Text(it.Subtitle))
					}),

					CardDescription(LineClamp2)(
						domi.Text(relativeTime(it.DeletedAt)),
					),
				),
			),
		),
	)
}

func appTrashDetail(it model.TrashItem) domi.Node {
	return FlexCol(Class("v-media-detail"))(
		ScrollY(Class("v-media-detail-body"))(
			SettingsPage()(
				FlexCol(Gap6)(
					SettingsContent()(
						TextNode(Size6)(domi.Text(trashItemTitle(it))),
						iff(it.Subtitle != "", func() domi.Node {
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
							Button(onClick(&msg.Restore{ID: it.ID}),
								ButtonGhost, ButtonSize2,
							)(Text("Restore")),
						),

						SettingsItem()(
							SettingsItemLabel()(
								SettingsItemLabelTitle("Purge"),
								SettingsItemLabelDescription("Permanently delete this item"),
							),
							Button(onClick(&msg.Purge{ID: it.ID}),
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

func trashForm(id string) domi.Node {
	return Button(onClick(&msg.Trash{ID: id}),
		Destructive, ButtonGhost, ButtonSize2,
	)(Text("Delete"))
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
	case model.TrashKindDownload:
		return "line/download-01"
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
	case model.TrashKindDownload:
		return "Download"
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
