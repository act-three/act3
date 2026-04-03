package web

import (
	"net/http"

	"ily.dev/act3/html"
	"ily.dev/act3/model"
	"ily.dev/act3/view"
)

func (c *Config) appTransmission(_ http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxR(func(tx *model.TxR) (html.Node, error) {
		ctx := req.Context()
		settings, err := tx.SettingGetByGroup(ctx, "transmission")
		if err != nil {
			return nil, err
		}
		title, body := view.AppTransmission(settings)
		return c.app(ctx, tx, title, body)
	})
}

func (c *Config) doTransmissionSettingsUpdate(w http.ResponseWriter, req *http.Request) (html.Node, error) {
	return c.withTxRW(func(tx *model.TxRW) (html.Node, error) {
		ctx := req.Context()
		err := tx.SettingSetString(ctx, model.SettingKeyTransmissionBaseURL, req.FormValue("url"))
		if err != nil {
			return nil, err
		}
		err = tx.SettingSetString(ctx, model.SettingKeyTransmissionPath, req.FormValue("path"))
		if err != nil {
			return nil, err
		}
		http.Redirect(w, req, "/app/transmission", http.StatusSeeOther)
		return nil, nil
	})
}
