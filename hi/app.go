package hi

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/html"

	"ily.dev/act3/hi/internal/sheet"
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

	// View returns the page to be displayed in the browser.
	//
	// It should construct a View, then call its PageRenderer
	// to produce a page. The View is centered in the viewport,
	// and the available space is the size of the viewport.
	View(context.Context, PageRenderer) Page

	// Subscriptions returns the set of active subscriptions.
	Subscriptions(context.Context) domi.Sub[Msg]

	// Preview returns the result of a potential navigation.
	// See PreviewRenderer.
	//
	// Preview must not modify the App state.
	//
	// The call to Preview represents a hypothetical
	// onURLRequest call from the browser. Same-origin links
	// omit the URL origin.
	//
	// Returning a zero Preview denotes that there is no preview.
	// It is always safe to decline to provide a preview. This
	// method is an optimization only. Preview is called to
	// pre-render pages the user is likely to visit (e.g. on
	// link hover), so navigation appears instant when the link
	// is clicked.
	Preview(context.Context, *url.URL, PreviewRenderer) Preview
}

// A PageRenderer renders a View to produce a Page.
// See [App.View].
type PageRenderer func(View) Page

// A Page is produced by a [PageRenderer].
// See [App.View].
type Page struct {
	title string
	page  domi.Node
}

// A PreviewRenderer renders a View to produce a Preview.
// See [App.Preview].
//
// The value of dest must equal the value the app would use
// for the PushURL command it issues when navigating
// to the previewed page.
// The dest must be a host-relative URL,
// like "/settings/profile" or "/".
//
// The view should be the same as that returned by [App.View]
// after a navigation to dest.
//
// If dest invalid, the PreviewRenderer panics.
type PreviewRenderer func(dest string, v View) Preview

// A Preview is produced by a [PreviewRenderer].
// See [App.Preview].
//
// The zero value denotes that no preview is available.
type Preview struct {
	dest string
	view Page
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
// the Handler may intercept the navigation (see [LinkPolicy]).
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
	th, styleNonce, icons := configure(o)
	// cssLink is filled in below, after the server exists to be asked
	// about its configuration; the constructor only runs on requests.
	var cssLink domi.Node
	sv := domi.NewServer(
		func(ctx context.Context, u *url.URL) (*instance[Msg, A], domi.Cmd[msg[Msg]]) {
			app, cmd := f(ctx, u)
			in := &instance[Msg, A]{
				app:     app,
				path:    urlPath(u),
				cssLink: cssLink,
				theme:   th,
				icons:   icons,
			}
			if styleNonce != nil {
				in.nonce = styleNonce(ctx)
			}
			return in, domi.MapCmd(wrapMsg[Msg], cmd)
		},
		func(u *url.URL) msg[Msg] { return wrapMsg(onURLRequest(u)) },
		func(u *url.URL) msg[Msg] {
			return msg[Msg]{
				appmsg: onURLChange(u),
				path:   urlPath(u),
			}
		},
		o...,
	)
	cssPath := path.Join(sv.InternalURLPrefix(), "hi."+cssDigest+".css")
	if !sv.HasCustomDocument() {
		cssLink = html.Link(attr.Rel("stylesheet"), attr.Href(cssPath))
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+cssPath, serveCSS)
	mux.Handle("/", sv)
	return mux
}

// msg is the domi msg type for instance.
// Field appmsg is currently required.
// Hi doesn't have any msg cases that it handles entirely itself.
// They all wrap app messages.
type msg[Msg any] struct {
	appmsg Msg
	path   []string // non-nil (even if empty) for URLChange, else nil
}

func wrapMsg[Msg any](m Msg) msg[Msg] {
	return msg[Msg]{appmsg: m}
}

// An instance adapts App to domi.App.
// It keeps the generated CSS rules for the lifetime of the page load
// so the style element only ever grows, keeping the rendered tree stable.
// Preview shares the sheet.
// Rendering a preview adds rules but doesn't modify App state.
type instance[Msg any, A App[Msg]] struct {
	app     A
	path    []string
	nonce   string
	cssLink domi.Node // loads the static stylesheet; nil with a custom document
	theme   theme
	icons   func(string) domi.Node
	sheet   sheet.Sheet
}

func (in *instance[Msg, A]) Update(ctx context.Context, m msg[Msg]) domi.Cmd[msg[Msg]] {
	if m.path != nil {
		in.path = m.path
	}
	return domi.MapCmd(wrapMsg[Msg], in.app.Update(ctx, m.appmsg))
}

func (in *instance[Msg, A]) Subscriptions(ctx context.Context) domi.Sub[msg[Msg]] {
	return domi.MapSub(wrapMsg[Msg], in.app.Subscriptions(ctx))
}

func (in *instance[Msg, A]) View(ctx context.Context) (title string, n domi.Node) {
	r := in.app.View(ctx, func(root View) Page {
		return in.render(root, in.path)
	})
	return r.title, domi.MapNode(wrapMsg[Msg], r.page)
}

func (in *instance[Msg, A]) Preview(ctx context.Context, u *url.URL) (dest, title string, n domi.Node) {
	r := in.app.Preview(ctx, u, func(d string, v View) Preview {
		dest, err := url.Parse(d)
		if err != nil || d == "" {
			panic(fmt.Errorf("hi: invalid preview destination %q", d))
		}
		if dest.Scheme != "" || dest.Host != "" || !strings.HasPrefix(d, "/") {
			panic(fmt.Errorf("hi: preview destination must be host-relative: %q", d))
		}
		return Preview{d, in.render(v, urlPath(dest))}
	})
	return r.dest, r.view.title, domi.MapNode(wrapMsg[Msg], r.view.page)
}

// render renders root as a page whose generated CSS rules are kept
// in the instance's sheet.
// The page carries all rules in the sheet,
// including those from earlier renders by the same instance.
// A non-nil cssLink is included in the page to load the static stylesheet.
func (in *instance[Msg, A]) render(root View, path []string) Page {
	env := environment{
		renv:       resenv{path: path},
		theme:      in.theme,
		iconSource: in.icons,
		sheet:      &in.sheet,
		root:       rootenv{atRoot: true},
	}
	b := unary(VStack, root)(env)
	rootAttr := attr.Class(in.sheet.ClassFor(in.theme.styles()))
	var a domi.Attr
	if in.nonce != "" {
		a = attr.Nonce(in.nonce)
	}
	style := domi.Tag("style", a)(domi.Text("@layer hi{" + in.sheet.CSS() + "}"))
	if b.pageScroll != 0 {
		var axes []string
		if b.pageScroll.hasAll(Horizontal) {
			axes = append(axes, "x")
		}
		if b.pageScroll.hasAll(Vertical) {
			axes = append(axes, "y")
		}
		rootAttr = domi.Group(rootAttr, domi.Name("scroll", strings.Join(axes, " ")))
	}
	return Page{
		title: b.title,
		// Order matters, static stylesheet, then generated style, then content.
		page: domi.Tag("hi-root", rootAttr)(in.cssLink, style, b.node),
	}
}

// Render returns HTML representing root.
//
// Option values that are inapplicable are ignored.
// For options that take a context, Render uses context.Background().
//
// Render is intended for tests.
// Applications serve their views with [Handler].
func Render(root View, o ...Option) (title string, page domi.Node) {
	th, styleNonce, icons := configure(o)
	in := instance[struct{}, App[struct{}]]{theme: th, icons: icons}
	if styleNonce != nil {
		in.nonce = styleNonce(context.Background())
	}
	r := in.render(root, nil)
	return r.title, r.page
}

// configure resolves the options in o.
// Options it does not know are for domi.
func configure(o []Option) (th theme, styleNonce func(context.Context) string, icons func(string) domi.Node) {
	th = defaultTheme
	icons = defaultIconSource
	for _, o := range o {
		switch o := o.(type) {
		case optionStyleNonce:
			styleNonce = o.f
		case optionTheme:
			th = o.theme
		case optionIconSource:
			icons = o.f
		}
	}
	return th, styleNonce, icons
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

// urlPath splits before unescaping so an escaped slash stays in its segment.
// It returns a non-nil slice even for the root path.
func urlPath(u *url.URL) []string {
	p := strings.TrimPrefix(u.EscapedPath(), "/")
	if p == "" {
		return []string{}
	}
	parts := strings.Split(p, "/")
	for i, part := range parts {
		// EscapedPath always returns a valid encoding.
		parts[i], _ = url.PathUnescape(part)
	}
	return parts
}
