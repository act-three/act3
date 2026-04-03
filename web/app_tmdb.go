package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) appTMDB(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		settings, err := tx.SettingGetByGroup(ctx, "tmdb")
		if err != nil {
			return nil, err
		}
		title, body := view.AppTMDB(settings)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) doTMDBSettingsUpdate(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		err := tx.SettingSetString(ctx, model.SettingKeyTMDBAccessToken, req.FormValue("token"))
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/tmdb", http.StatusSeeOther)
		return nil, nil
	})
}
