package ui

import (
	"context"
	"net/http"
	"net/url"
	"path"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/xui/internal/sheet"
)

// An App stores application state,
// applies updates,
// and generates the view for each state.
// One instance contains the state for a single browser page load.
// See [Handler] for instance lifecycle.
//
// The context given to each method contains the instance ID
// (see [domi.InstanceID])
// as well as values from the HTTP request context, if any.
// It is cancelled when the instance ends.
type App[Msg any] interface {
	// Update is responsible for updating the App state
	// in response to each Msg.
	//
	// Update should avoid long-running work and operations
	// that take unknown amounts of time, such as network I/O.
	// For these cases, Update should return a Cmd.
	Update(context.Context, Msg) domi.Cmd[Msg]

	// View returns the view to be displayed in the browser.
	// It is centered in the viewport, and the available space
	// is the size of the viewport.
	View(context.Context) View

	// Subscriptions returns the set of active subscriptions.
	Subscriptions(context.Context) domi.Sub[Msg]

	// Preview returns the result of a potential navigation.
	//
	// Preview must not modify the App state.
	//
	// The call to Preview represents a hypothetical
	// onURLRequest call from the browser. Same-origin links
	// omit the URL origin.
	//
	// If Preview returns a nonempty dest value, it must equal
	// the value the app would use for the PushURL command it
	// issues in response to the URL request. The value for v
	// should be the same as that returned by View after a
	// navigation to dest.
	//
	// An empty dest denotes that there is no preview available.
	// It is always safe to decline to provide a preview.
	// This method is an optimization only. Preview is called
	// to pre-render pages the user is likely to visit (e.g. on
	// link hover), so navigation appears instant when the link
	// is clicked.
	Preview(context.Context, *url.URL) (dest string, v View)
}

// Handler returns an HTTP handler that serves an [App].
//
// On initial page load, the Handler calls f
// with the request URL
// to obtain an instance of the App and an initial Cmd.
// The context contains the instance ID (see [domi.InstanceID])
// and is cancelled when the instance ends.
//
// When the user clicks a link,
// the Handler may intercept the navigation,
// as configured per link (see [LinkView.RequirePageLoad]).
// It then calls onURLRequest to produce a Msg.
// Same-origin links omit the URL origin.
// Method Update decides how to handle the request,
// typically by returning a PushURL or ReplaceURL command.
//
// When the URL changes
// (from a navigation command or the browser's back and forward buttons),
// onURLChange is called to produce a Msg.
// The app's Update method then updates its state accordingly.
//
// [Option] values provide further control over the Handler's behavior.
func Handler[Msg any, A App[Msg]](
	f func(context.Context, *url.URL) (A, domi.Cmd[Msg]),
	onURLRequest func(u *url.URL) Msg,
	onURLChange func(*url.URL) Msg,
	o ...Option,
) http.Handler {
	var styleNonce func(context.Context) string
	for _, o := range o {
		if o, ok := o.(optionStyleNonce); ok {
			styleNonce = o.f
		}
	}
	// cssLink is filled in below, after the server exists to be asked
	// about its configuration; the constructor only runs on requests.
	var cssLink domi.Node
	sv := domi.NewServer(
		func(ctx context.Context, u *url.URL) (*instance[Msg, A], domi.Cmd[Msg]) {
			app, cmd := f(ctx, u)
			in := &instance[Msg, A]{app: app, cssLink: cssLink}
			if styleNonce != nil {
				in.nonce = styleNonce(ctx)
			}
			return in, cmd
		},
		onURLRequest,
		onURLChange,
		o...,
	)
	cssPath := path.Join(sv.InternalURLPrefix(), "ui."+cssDigest+".css")
	if !sv.HasCustomDocument() {
		cssLink = html.Link(attr.Rel("stylesheet"), attr.Href(cssPath))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+cssPath, serveCSS)
	mux.Handle("/", sv)
	return mux
}

// An instance adapts App to domi.App.
// It keeps the generated CSS rules for the lifetime of the page load
// so the style element only ever grows, keeping the rendered tree stable.
// Preview shares the sheet.
// Rendering a preview adds rules but doesn't modify App state.
type instance[Msg any, A App[Msg]] struct {
	app     A
	nonce   string
	cssLink domi.Node // loads the static stylesheet; nil with a custom document
	sheet   sheet.Sheet
}

func (in *instance[Msg, A]) Update(ctx context.Context, m Msg) domi.Cmd[Msg] {
	return in.app.Update(ctx, m)
}

func (in *instance[Msg, A]) Subscriptions(ctx context.Context) domi.Sub[Msg] {
	return in.app.Subscriptions(ctx)
}

func (in *instance[Msg, A]) View(ctx context.Context) (title string, n domi.Node) {
	return in.render(in.app.View(ctx))
}

func (in *instance[Msg, A]) Preview(ctx context.Context, u *url.URL) (dest, title string, n domi.Node) {
	dest, v := in.app.Preview(ctx, u)
	if dest == "" {
		return "", "", nil
	}
	title, n = in.render(v)
	return dest, title, n
}

// render renders root as a page whose generated CSS rules are kept
// in the instance's sheet.
// The page carries all rules in the sheet,
// including those from earlier renders by the same instance.
// A non-nil cssLink is included in the page to load the static stylesheet.
func (in *instance[Msg, A]) render(root View) (title string, page domi.Node) {
	env := environment{
		sheet: &in.sheet,
		root:  rootenv{atRoot: true},
	}
	b := unary(VStack, root).render(env)
	var a domi.Attr
	if in.nonce != "" {
		a = attr.Nonce(in.nonce)
	}
	style := domi.Tag("style", a)(domi.Text("@layer xui{" + in.sheet.CSS() + "}"))
	var rootAttr domi.Attr
	if b.pageScroll != 0 {
		var axes []string
		if b.pageScroll.hasAll(Horizontal) {
			axes = append(axes, "x")
		}
		if b.pageScroll.hasAll(Vertical) {
			axes = append(axes, "y")
		}
		rootAttr = domi.Name("scroll", strings.Join(axes, " "))
	}
	// Order matters, static stylesheet, then generated style, then content.
	// Emit the style element even when empty to keep the tree stable.
	return b.title, domi.Tag("ui-root", rootAttr)(in.cssLink, style, b.node)
}

// Render returns HTML representing root.
// It is intended for tests.
// Applications serve their views with [Handler].
func Render(root View) (title string, page domi.Node) {
	var in instance[struct{}, App[struct{}]]
	return in.render(root)
}

// An Option configures a [Handler].
//
// See [domi.Option] for more option constructors.
type Option = domi.Option

// StyleNonce adds a nonce to the style element generated for each page.
// The nonce must match the one in the page's Content-Security-Policy.
//
// A Handler calls f once per App instance,
// with the context given to the instance's constructor.
func StyleNonce(f func(context.Context) string) Option {
	return optionStyleNonce{f: f}
}

// A optionStyleNonce is recognized by Handler and ignored by domi,
// which disregards options it does not know.
// The embedded Option is never set; it only marks the type an Option.
type optionStyleNonce struct {
	domi.Option
	f func(context.Context) string
}
