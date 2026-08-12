package ui

import (
	"ily.dev/domi"
	"ily.dev/domi/attr"

	"ily.dev/act3/xui/internal/sheet"
)

// A Renderer renders Views.
// Use one Renderer for the lifetime of each domi app instance.
//
// The output of a Renderer includes generated CSS rules
// used by the views it renders.
//
// A Renderer must not be copied.
// The zero value is ready to use.
type Renderer struct {
	// Nonce is added to the generated style element when set.
	// It must match the nonce in the page's Content-Security-Policy.
	// Set it before calling Render for the first time.
	Nonce string

	sheet sheet.Sheet
}

// Render returns an HTML page representing root.
// It displays root centered in the browser viewport.
// The available space for root is the size of the viewport.
//
// The returned page should be placed directly inside the "body" element.
//
//	func (app *App) View(ctx context.Context) (string, domi.Node) {
//	    return "Greeting", app.ui.Render(Text("Hello, world!"))
//	}
func (r *Renderer) Render(root View) (page domi.Node) {
	if len(root.nodes()) > 1 {
		root = VStack(root)
	}
	content, _, _ := subviewsRendered(environment{sheet: &r.sheet}, root)
	var nonce domi.Attr
	if r.Nonce != "" {
		nonce = attr.Nonce(r.Nonce)
	}
	// Put the stylesheet before the content
	// so its rules are available before the content is painted.
	// Emit it even when empty to keep the tree stable.
	style := domi.Tag("style", nonce)(domi.Text("@layer xui{" + r.sheet.CSS() + "}"))
	return domi.Tag("ui-root")(style, content)
}
