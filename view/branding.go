package view

import (
	"ily.dev/domi"

	. "ily.dev/act3/ui"
)

func wordmark() domi.Node {
	return FlexRow(Style("align-items:center;gap:0.5rem"))(
		Box(Class("v-wordmark")),
		Box()(domi.Safe("&beta;")),
	)
}
