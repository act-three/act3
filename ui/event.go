package ui

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"

	"ily.dev/domi"
)

// errOtherKey discards a keydown event for a key the handler doesn't
// care about. (A handler error drops the event; see domi.On.)
var errOtherKey = errors.New("other key")

// onEscape delivers m when the Escape key is pressed.
func onEscape[Msg any](m Msg) domi.Attr {
	return domi.On("keydown", func(v jsontext.Value) (Msg, error) {
		var e struct {
			Key string `json:"key"`
		}
		if err := json.Unmarshal(v, &e); err != nil {
			var zero Msg
			return zero, err
		}
		if e.Key != "Escape" {
			var zero Msg
			return zero, errOtherKey
		}
		return m, nil
	}, []string{"key"})
}

// errChildClick discards a click that landed on a descendant rather
// than the handler's own element.
var errChildClick = errors.New("child click")

// onClickSelf delivers m when the element itself — not a descendant
// — is clicked. The element must carry a unique id.
func onClickSelf[Msg any](m Msg) domi.Attr {
	return domi.On("click", func(v jsontext.Value) (Msg, error) {
		var e struct {
			Target struct {
				ID string `json:"id"`
			} `json:"target"`
			CurrentTarget struct {
				ID string `json:"id"`
			} `json:"currentTarget"`
		}
		if err := json.Unmarshal(v, &e); err != nil {
			var zero Msg
			return zero, err
		}
		if e.CurrentTarget.ID == "" || e.Target.ID != e.CurrentTarget.ID {
			var zero Msg
			return zero, errChildClick
		}
		return m, nil
	}, []string{"target", "id"}, []string{"currentTarget", "id"})
}

// onChangeValue calls f with the element's committed value
// and delivers the resulting message.
func onChangeValue[Msg any](f func(value string) Msg) domi.Attr {
	return domi.On("change", func(v jsontext.Value) (Msg, error) {
		var e struct {
			CurrentTarget struct {
				Value string `json:"value"`
			} `json:"currentTarget"`
		}
		err := json.Unmarshal(v, &e)
		return f(e.CurrentTarget.Value), err
	}, []string{"currentTarget", "value"})
}
