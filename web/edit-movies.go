package web

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/model"
	"ily.dev/act3/service/tmdb"
	. "ily.dev/act3/ui"
	"ily.dev/act3/ui/turbo"
	"ily.dev/act3/view"
	"ily.dev/act3/xstrings"
)

func (c *Config) editMovies(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		all, err := tx.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaMovies("All Movies", all), nil
	})
}

func (c *Config) editMoviesDetail(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		_, selID, _ := xstrings.LastCut(req.PathValue("id"), "-")

		mo, err := tx.Movie(ctx, selID)
		if err == sql.ErrNoRows {
			http.Redirect(w, req, "/app/movies", http.StatusSeeOther)
			return nil, nil
		} else if err != nil {
			return nil, err
		}

		detail := view.EditMediaMoviesDetail(mo)
		if req.Header.Get("turbo-frame") == "detail" {
			return view.PageFrame(mo.Title(), "detail", detail), nil
		}

		all, err := tx.MovieHeadList(ctx)
		if err != nil {
			return nil, err
		}
		return view.EditMediaMovies(mo.Title(), all, detail), nil
	})
}

func (c *Config) doAddMovie(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		title := req.FormValue("title")
		if title == "" {
			return nil, &model.ValidationError{
				Op:  "add movie",
				Err: errNotFound,
			}
		}
		mo, err := tx.MovieCreate(ctx, title, 0)
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, mo.EditURL(), http.StatusSeeOther)
		return nil, nil
	})
}

func (c *Config) doAddMovieTMDB(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		mo, err := tx.MovieCreateByTMDBID(ctx, req.FormValue("id"))
		if err != nil {
			return nil, err
		}
		return turbo.Frame("tmdb-"+strconv.FormatInt(*mo.TMDBID(), 10))(
			movieResultLink(mo),
			turbo.Prepend(view.EditMediaMoviesListItems,
				ListItems([]*model.MovieHead{mo}, view.EditMediaMoviesListItem),
			),
		), nil
	})
}

func (c *Config) movieAddDialogReq(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return view.Dialog(
		html.Div(
			attr.Class(`
				w-2xl
				h-full
				flex
				flex-col
				gap-2
			`),
		)(
			html.Div(
				attr.Class("flex-none"),
			)(
				html.Text("Add Movie"),
			),
			html.Form(
				attr.Action("/-/part/movie-search"),
				attr.Attr("data-turbo-frame")("results"),
			)(
				InputText(
					attr.Attr("autofocus"),
					attr.Class("flex-none"),
					attr.Name("q"),
				),
			),
			html.Div(
				attr.Class(`
					flex-initial
					overflow-auto
					overscroll-contain
					h-dvh
					max-h-full
					border
					rounded-sm
				`),
			)(
				turbo.Frame("results"),
			),
		),
	), nil
}

func (c *Config) movieSearch(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	type result struct {
		TMDB  tmdb.SearchResult
		Local *model.MovieHead
	}
	ctx := req.Context()
	query := req.FormValue("q")
	slog.InfoContext(ctx, "movie search", "q", query)
	if strings.TrimSpace(query) == "" {
		return turbo.Frame("results"), nil
	}
	movies, err := c.TMDB.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "movie results",
		"len", len(movies))
	results := make([]result, len(movies))
	ids := make([]*int64, len(movies))
	for i := range movies {
		id := int64(movies[i].ID)
		ids[i] = &id
	}
	m := make(map[int64]*model.MovieHead)

	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		existing, err := tx.MovieHeadListByTMDBID(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, mo := range existing {
			m[*mo.TMDBID()] = mo
		}
		for i, res := range movies {
			results[i] = result{
				TMDB:  res,
				Local: m[int64(res.ID)],
			}
		}

		return turbo.Frame("results")(
			FlexCol(Gap4, Class("p-4"))(
				html.Range(results, func(t result) html.Node {
					frameID := "tmdb-" + strconv.Itoa(t.TMDB.ID)
					return Card(CardSurface, CardSize3,
						Class("h-[200px]"),
					)(
						FlexRow(Gap4, Class("h-full"))(
							Inset(InsetSideLeft,
								Class("flex-none"),
							)(
								html.Img(
									Class("block h-full aspect-2/3 object-cover"),
									attr.Src(tmdb.ImageURL(t.TMDB.PosterPath)),
								),
							),
							FlexCol(Gap2)(
								movieSearchTitle(t.TMDB),
								movieSearchAction(frameID, t),
								TextNode(LineClamp3)(
									html.Text(t.TMDB.Overview),
								),
							),
						),
					)
				}),
			),
		), nil
	})
}

func movieSearchTitle(m tmdb.SearchResult) html.Node {
	title := m.Title
	if len(m.ReleaseDate) >= 4 {
		title += " (" + m.ReleaseDate[:4] + ")"
	}
	return html.Text(title)
}

func movieSearchAction(frameID string, t struct {
	TMDB  tmdb.SearchResult
	Local *model.MovieHead
}) html.Node {
	if t.Local != nil {
		return movieResultLink(t.Local)
	}
	return turbo.Frame(frameID)(
		html.Form(
			attr.Method("post"),
			attr.Action("/-/do/add-movie-tmdb"),
			turbo.DataFrame(frameID),
		)(
			html.Input(
				attr.Type("hidden"),
				attr.Name("id"),
				attr.Value(strconv.Itoa(t.TMDB.ID)),
			),
			Button(ButtonSurface)(html.Text("Add")),
		),
	)
}

func movieResultLink(mo *model.MovieHead) html.Node {
	return FlexRow(Gap2)(
		Label("line/check-circle", "In Library"),
		Button(
			Href(mo.EditURL()),
			Attr("data-turbo-frame")("detail"),
			Attr("data-action")("click->dialog#dismiss"),
		)(
			Text("Edit"),
		),
	)
}
