package ui

import (
	"encoding/json/v2"

	"ily.dev/act3/html"
	"ily.dev/act3/html/attr"
	"ily.dev/act3/ui/stimulus"
)

// ActionButton renders a button that POSTs to action
// with form-encoded params on click.
// The controller handles redirects (via Turbo.visit)
// and turbo-stream responses (via Turbo.renderStreamMessage).
func ActionButton(action string, params map[string]string, attrs ...attr.Node) html.Element {
	return func(children ...html.Node) html.Node {
		as := []attr.Node{
			stimulus.Controller("action-button"),
			stimulus.Value("action-button", "url")(action),
			stimulus.Action("click->action-button#call"),
		}
		if len(params) > 0 {
			paramsJSON, _ := json.Marshal(params)
			as = append(as, stimulus.Value("action-button", "params")(string(paramsJSON)))
		}
		as = append(as, group(attrs...))
		return Button(as...)(children...)
	}
}
