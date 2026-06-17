package view

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"

	"ily.dev/domi"
	"ily.dev/domi/event"

	"ily.dev/act3/model"
	"ily.dev/act3/msg"
	. "ily.dev/act3/ui"
)

// onClick delivers m when the element is clicked.
var onClick = event.Click[msg.Msg]

// The generic ui components that take a message value, specialized
// to the application message type. (Handlers must return exactly
// msg.Msg, so inference from a concrete message would panic at
// render; components that take a func(value) msg.Msg infer correctly
// and need no specialization. See domi.On.)
var (
	dialog                = Dialog[msg.Msg]
	imageDialog           = ImageDialog[msg.Msg]
	popover               = Popover[msg.Msg]
	settingsToggle        = SettingsToggle[msg.Msg]
	settingsButtonRowItem = SettingsButtonRowItem[msg.Msg]
)

// errNotAMove discards a change event that did not come from the
// sortable bridge (see onEpisodeMove).
var errNotAMove = errors.New("not a sortable move")

// onEpisodeMove delivers the episode move described by a synthetic
// change event from the sortable controller (see ui/sortable.js):
// drag stays a client gesture, and the drop arrives here as an
// ordinary event whose detail carries the move.
func onEpisodeMove() domi.Attr {
	return domi.On("change", func(v jsontext.Value) (msg.Msg, error) {
		var e struct {
			Detail struct {
				EpisodeID    string `json:"episodeId"`
				FromSeasonID string `json:"fromSeasonId"`
				SeasonID     string `json:"seasonId"`
				Index        int    `json:"index"`
			} `json:"detail"`
		}
		if err := json.Unmarshal(v, &e); err != nil {
			return nil, err
		}
		d := e.Detail
		if d.EpisodeID == "" || d.FromSeasonID == "" || d.SeasonID == "" {
			return nil, errNotAMove
		}
		return &msg.EpisodeMove{
			EpisodeID:    d.EpisodeID,
			FromSeasonID: d.FromSeasonID,
			SeasonID:     d.SeasonID,
			Index:        d.Index,
		}, nil
	},
		[]string{"detail", "episodeId"},
		[]string{"detail", "fromSeasonId"},
		[]string{"detail", "seasonId"},
		[]string{"detail", "index"},
	)
}

// onSubmit calls f with the value of the form's named field on
// submit and delivers the resulting message.
func onSubmit(field string, f func(value string) msg.Msg) domi.Attr {
	return domi.On("submit", func(v jsontext.Value) (msg.Msg, error) {
		var e struct {
			Target struct {
				Elements map[string]struct {
					Value string `json:"value"`
				} `json:"elements"`
			} `json:"target"`
		}
		err := json.Unmarshal(v, &e)
		return f(e.Target.Elements[field].Value), err
	}, []string{"target", "elements", field, "value"})
}

// onPlay opens the player for the given video ids
// when the play form is submitted.
func onPlay(ids model.PlayIDs) domi.Attr {
	return domi.On("submit", func(v jsontext.Value) (msg.Msg, error) {
		var e struct {
			Target struct {
				Elements struct {
					A struct {
						Dataset struct {
							SelectCurrentValue string `json:"selectCurrentValue"`
						} `json:"dataset"`
					} `json:"a"`
					S struct {
						Dataset struct {
							SelectCurrentValue string `json:"selectCurrentValue"`
						} `json:"dataset"`
					} `json:"s"`
					PinAudio struct {
						Value string `json:"value"`
					} `json:"pin_audio"`
				} `json:"elements"`
			} `json:"target"`
		}
		err := json.Unmarshal(v, &e)
		el := e.Target.Elements
		return &msg.Play{
			IDs:      ids,
			Audio:    el.A.Dataset.SelectCurrentValue,
			Subtitle: el.S.Dataset.SelectCurrentValue,
			PinAudio: el.PinAudio.Value == "1",
		}, err
	},
		[]string{"target", "elements", "a", "dataset", "selectCurrentValue"},
		[]string{"target", "elements", "s", "dataset", "selectCurrentValue"},
		[]string{"target", "elements", "pin_audio", "value"},
	)
}

// onPlayerClose closes the open player.
func onPlayerClose() domi.Attr {
	return domi.On("click", func(jsontext.Value) (msg.Msg, error) {
		return &msg.PlayerClose{}, nil
	})
}
