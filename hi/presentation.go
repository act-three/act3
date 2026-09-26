package hi

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"ily.dev/domi"
	"ily.dev/domi/attr"
	"ily.dev/domi/event"

	"ily.dev/act3/hi/internal/canon"
	"ily.dev/act3/hi/internal/sheet"
)

// A PresentationState controls whether a presentation,
// such as a dialog, menu, or popover, is displayed.
//
// See [View.Dialog], [View.Menu], and [View.Popover].
//
// The zero value is a valid PresentationState
// that hides the presentation.
type PresentationState struct {
	isOpen  bool
	dismiss domi.Attr
}

// Present displays a presentation while isOpen is true.
// When a gesture requests the presentation to be dismissed,
// it sends dismiss to the app.
// The app should then update its value for isOpen to false.
func Present[Msg any](isOpen bool, dismiss Msg) PresentationState {
	return PresentationState{
		isOpen:  isOpen,
		dismiss: event.Click(dismiss),
	}
}

func (v base) Popover(p PresentationState, content View) View {
	return v.modify(wrapAttachment{popover: nodePopover{
		isOpen:  p.isOpen,
		dismiss: p.dismiss,
		content: content,
		at:      Bottom,
		anchor:  Top,
	}}.modify)
}

func (v base) Menu(p PresentationState, content View) View {
	return v.modify(wrapAttachment{popover: nodePopover{
		isOpen:  p.isOpen,
		dismiss: p.dismiss,
		content: content,
		at:      BottomLeading,
		anchor:  TopLeading,
	}}.modify)
}

func (v base) Dialog(p PresentationState, content View) View {
	dialog := nodeDialog{p.isOpen, p.dismiss,
		content.
			Padding(Edges(12i)),
	}
	return v.modify(func(n node) node {
		return func(env environment) box {
			b := n(env)
			dialog := renderSubviewNode(env, dialog.render)
			b.node = domi.Fragment(b.node, dialog.content)
			return b
		}
	})
}

// wrapAttachment gives each popover its own anchor scope without letting
// the presented content participate in the receiver's layout.
type wrapAttachment struct {
	popover nodePopover
}

func (w wrapAttachment) modify(n node) node {
	return func(env environment) box {
		env.tag = cmp.Or(env.tag, "hi-attachment")
		env.style.Set("display", "grid")
		env.style.Set("grid-template-columns", "100%")
		env.style.Set("grid-template-rows", "100%")
		Center.setItemsOn(&env.style)
		env.style.Set("anchor-name", "--hi-attachment")
		env.style.Set("anchor-scope", "--hi-attachment")
		p := wrapSubview(env, n)
		popover := renderSubviewNode(env, w.popover.render)
		p.content = domi.Fragment(p.content, popover.content)
		return build(env, p)
	}
}

type nodePopover struct {
	isOpen     bool
	dismiss    domi.Attr
	content    View
	at, anchor Alignment
}

func (n nodePopover) render(env environment) box {
	v := ZStack(n.content).
		BorderStroke(0.5, borderColor).
		modify(shadowMedium).
		ThemeBackground(menuColor).
		BorderShape(RoundedRectangle(11i))
	opposite := n.at ^ n.anchor
	if opposite&(Top|Bottom) == Top|Bottom {
		v = v.Padding(EdgesLetterbox(4))
	}
	if opposite&(Leading|Trailing) == Leading|Trailing {
		v = v.Padding(EdgesPillarbox(4))
	}
	inner := presentationEnvironment(env, n.isOpen)
	b := unary(VStack, v)(inner)
	inner.nextenv = nextenv{}
	fills := b.fills &^ b.rigid
	inner.style.Merge(popoverPositionStyle(n.at, n.anchor, fills, env.sheet))
	if fill := map[AxisSet]string{
		Horizontal:            "horizontal",
		Vertical:              "vertical",
		Horizontal | Vertical: "both",
	}[fills]; fill != "" {
		inner.add(domi.Name("data-hi-fill", fill))
	}
	inner.style.Set("position", "fixed")
	inner.style.Set("display", "grid")
	inner.style.Set("grid-template-columns", "100%")
	inner.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&inner.style)
	inner.style.Set("width", "max-content")
	inner.style.Set("height", "max-content")
	if fills.hasAll(Horizontal) {
		inner.style.Set("width", "stretch")
	}
	if fills.hasAll(Vertical) {
		inner.style.Set("height", "stretch")
	}
	inner.tag = "div"
	inner.style.SetPseudo(":not(:popover-open)", "display", "none !important")
	inner.add(
		domi.Name("popover", "manual"),
		domi.Name("data-hi-presentation", "popover"),
	)
	if b.title != "" {
		inner.add(attr.Role("dialog"))
	}
	return buildPresentation(inner, b, n.dismiss)
}

type nodeDialog struct {
	isOpen  bool
	dismiss domi.Attr
	content View
}

func (n nodeDialog) render(env environment) box {
	root := env.theme
	root.bgbase = root.bgroot
	background := newColor(dialogColor.color().colorCoords(root))
	v := ZStack(n.content).
		BorderStroke(0.5, borderColor).
		modify(shadowDialog).
		ThemeBackground(background).
		BorderShape(RoundedRectangle(21i))
	inner := presentationEnvironment(env, n.isOpen)
	b := unary(VStack, v)(inner)
	inner.nextenv = nextenv{}
	inner.tag = "dialog"
	inner.style.Set("position", "fixed")
	Edges(12).setOn(&inner.style, "inset")
	inner.style.Set("display", "grid")
	inner.style.Set("grid-template-columns", "100%")
	inner.style.Set("grid-template-rows", "100%")
	Center.setItemsOn(&inner.style)
	inner.style.SetPseudo(":not([open])", "display", "none !important")
	var backdrop sheet.StyleSet
	backdrop.SetPseudo("::backdrop", "background-color", "rgb(0 0 0 / 0.3)")
	inner.add(attr.Class(env.sheet.ClassFor(backdrop)), domi.Name("data-hi-presentation", "dialog"))
	return buildPresentation(inner, b, n.dismiss)
}

func presentationEnvironment(env environment, open bool) environment {
	env.lc = layoutContext{}
	env.container = containerGrid
	env.unbounded = 0
	env.canPresent = env.canPresent && open
	env.root = rootenv{}
	return env
}

func buildPresentation(env environment, content box, dismiss domi.Attr) box {
	env.add(domi.Name("data-hi-open", fmt.Sprint(env.canPresent)))
	if content.title != "" {
		env.add(domi.Name("aria-label", content.title))
	}
	// Domi delegates click events. A hidden proxy lets native dismissal
	// gestures dispatch the application's handler without intercepting
	// unrelated events from the presentation's content.
	p := plan{content: domi.Fragment(
		domi.Tag("hi-dismiss", domi.Name("hidden", ""), dismiss,
			attr.Class(env.sheet.ClassFor(sheet.Style("display", "none !important"))))(),
		content.node,
	)}
	b := build(env, p)
	b.rigid = Horizontal | Vertical
	return b
}

// popoverPositionStyle attaches a top-layer box to the receiver's
// scoped anchor. Insets describe the available interval on each axis,
// while self-alignment selects the box's attachment point within it.
// Unlike a percentage translation, both can change in @position-try.
func popoverPositionStyle(at, anchor Alignment, fills AxisSet, sh *sheet.Sheet) canon.StyleSet {
	ss := popoverAttachmentStyle(at, anchor)
	ss.Set("position-anchor", "--hi-attachment")
	var fallbacks []string
	for _, pair := range popoverAlternatives(at, anchor)[1:] {
		fallbacks = append(fallbacks, sh.PositionTryFor(popoverAttachmentStyle(pair[0], pair[1]).Decls()))
	}
	if len(fallbacks) > 0 {
		ss.Set("position-try-fallbacks", strings.Join(fallbacks, ","))
	}
	switch {
	case fills.hasAny(Vertical):
		ss.Set("position-try-order", "most-height")
	case fills.hasAny(Horizontal):
		ss.Set("position-try-order", "most-width")
	}
	return ss
}

func popoverAttachmentStyle(at, anchor Alignment) canon.StyleSet {
	if (at|anchor)&FirstBaseline != 0 {
		panic("hi: presentations do not support FirstBaseline attachment")
	}
	var ss canon.StyleSet
	popoverAxisStyle(&ss, "inline", "justify-self", at.horizontal(), anchor.horizontal())
	popoverAxisStyle(&ss, "block", "align-self", at.vertical(), anchor.vertical())
	return ss
}

func popoverAxisStyle(ss *canon.StyleSet, axis, alignment string, at, anchor Alignment) {
	coordinate := "anchor(" + popoverAlignment(at) + ")"
	start, end := "0px", "0px"
	switch anchor.point() {
	case 0:
		start = coordinate
	case 100:
		end = coordinate
	case 50:
		// anchor() measures from the inset property's own edge. These
		// matching expressions leave the largest symmetric interval
		// centered on the chosen coordinate and inside the viewport.
		start = fmt.Sprintf("max(0px, calc(2 * %s - 100%%))", coordinate)
		end = start
	}
	ss.Set("inset-"+axis+"-start", start)
	ss.Set("inset-"+axis+"-end", end)
	ss.Set(alignment, "unsafe "+popoverAlignment(anchor))
}

func popoverAlignment(a Alignment) string {
	if a == Center {
		return "center"
	}
	// The top-layer containing block has the viewport's direction,
	// which can differ from the direction inherited by the attachment.
	return "self-" + a.keyword()
}

// Reflect the complete attachment relationship, preserving its preferred
// distance from the base and deduplicating centered-axis reflections.
func popoverAlternatives(at, anchor Alignment) [][2]Alignment {
	var result [][2]Alignment
	for _, reflect := range []AxisSet{0, Vertical, Horizontal, Horizontal | Vertical} {
		pair := [2]Alignment{reflectAttachment(at, reflect), reflectAttachment(anchor, reflect)}
		if !slices.Contains(result, pair) {
			result = append(result, pair)
		}
	}
	return result
}

func reflectAttachment(a Alignment, axes AxisSet) Alignment {
	if axes.hasAny(Horizontal) && a.horizontal() != Center {
		a ^= Leading | Trailing
	}
	if axes.hasAny(Vertical) && a.vertical() != Center {
		a ^= Top | Bottom
	}
	return a
}
