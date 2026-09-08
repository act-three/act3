package fixture

import (
	"ily.dev/domi"

	. "ily.dev/act3/xui"
)

var buttonStyles = []struct {
	name  string
	style ButtonStyle
}{
	{"Bordered", Bordered},
	{"Prominent", Prominent},
	{"Subtle", Subtle},
	{"Borderless", Borderless},
	{"Destructive", Destructive},
	{"DestructiveSubtle", DestructiveSubtle},
}

var buttonControlSizes = []struct {
	name string
	size ControlSize
}{
	{"Mini", Mini},
	{"Small", Small},
	{"Regular", Regular},
	{"Large", Large},
}

func sampleButton(label View) ButtonView { return Button(Msg{Edit: true}, label) }

func buttonSection(title string, body View) View {
	return VStack(
		Text(title).TextForeground(Secondary).TextFont(Caption),
		body,
	)
}

func buttonGallery() View {
	return VStack(
		Text("Inspection fixture; geometry is provisional. Reserved edges, clipped fills, "+
			"alpha rings, shadows, focus outlines, and return transitions remain pending.").Font(Caption),
		buttonSection("Styles and sizes — text and square labels", buttonSizes()),
		buttonSection("States — hover, hold, or Tab to inspect browser interaction", buttonStates()),
		buttonSection("Label content and layout", buttonLabels()),
		buttonSection("Nested themes, styles, and sizes", buttonContexts()),
	).
		Gap(24).
		Alignment(Leading).
		Class("button-gallery")
}

func buttonSizes() View {
	var rows []View
	for _, style := range buttonStyles {
		var sizes []View
		for _, size := range buttonControlSizes {
			sizes = append(sizes, buttonSection(size.name, HStack(
				sampleButton(Text("Save")),
				sampleButton(Primary.Frame(Width(16), Height(16))).
					Attr(domi.Name("aria-label", "Square label")).Class("button-square"),
			).Gap(8).ControlSize(size.size)))
		}
		rows = append(rows, buttonSection(style.name,
			Grid(Columns(4), sizes...).ButtonStyle(style.style)))
	}
	return VStack(rows...).Gap(16).Class("button-sizes")
}

func buttonStates() View {
	var rows []View
	for _, style := range buttonStyles {
		var states []View
		for _, state := range []struct {
			name                     string
			selected, open, disabled bool
		}{
			{"Rest", false, false, false},
			{"Selected", true, false, false},
			{"Menu open", false, true, false},
			{"Selected + menu", true, true, false},
			{"Disabled", false, false, true},
			{"Selected + disabled", true, false, true},
			{"Menu + disabled", false, true, true},
			{"Selected + menu + disabled", true, true, true},
		} {
			states = append(states, buttonSection(state.name,
				sampleButton(Text("Save")).Selected(state.selected).
					MenuOpen(state.open).Disabled(state.disabled)))
		}
		rows = append(rows, buttonSection(style.name,
			Grid(Columns(4), states...).ButtonStyle(style.style)))
	}
	return VStack(rows...).Gap(16).Class("button-states")
}

func buttonLabels() View {
	const long = "Save all changes to this collection and update every episode in the library"
	return Grid(Columns(3),
		buttonSection("Composite label", sampleButton(
			HStack(Icon("film"), Text("Movies")).Gap(6))),
		buttonSection("Arranged lines", sampleButton(
			VStack(Text("Save changes"), Text("Current collection")).Gap(2))),
		buttonSection("Explicit label font and color", sampleButton(
			Text("Save").Font(Title).Foreground(Red))),
		buttonSection("Two-line limit", sampleButton(
			Text(long).LineLimit(2).Frame(Width(120)))),
		buttonSection("Unlimited wrapping", sampleButton(
			Text(long).LineLimit(0).Frame(Width(120)))),
		buttonSection("Adjacent labels under pressure", HStack(
			sampleButton(Text(long)),
			sampleButton(Text(long)),
		).Gap(8).Frame(Width(220))),
		buttonSection("Horizontal fill", sampleButton(
			HStack(Text("Save"), Spacer(), Icon("check")).Gap(8),
		).Frame(Width(220))),
		buttonSection("Vertical fill", sampleButton(
			VStack(Icon("arrow-up"), Spacer(), Icon("arrow-down")).Gap(8),
		).Frame(Height(100))),
		buttonSection("Rectangular label", sampleButton(
			Primary.Frame(Width(48), Height(16)),
		).Attr(domi.Name("aria-label", "Rectangular label"))),
	).Class("button-labels")
}

func buttonContextStyles() View {
	var samples []View
	for _, style := range buttonStyles {
		samples = append(samples, buttonSection(style.name, HStack(
			sampleButton(Text("Save")),
			sampleButton(Text("Save")).MenuOpen(true),
		).Gap(8).ButtonStyle(style.style)))
	}
	return Grid(Columns(3), samples...)
}

func buttonContexts() View {
	var panels []View
	for _, theme := range []struct {
		name string
		base Color
	}{
		{"Light", OKLCH(0.98, 0.005, 80)},
		{"Dark", OKLCH(0.22, 0.015, 270)},
	} {
		panels = append(panels, VStack(
			buttonSection(theme.name+" — rest and menu open", buttonContextStyles()),
			buttonSection("Nested elevated surface — rest and menu open", buttonContextStyles()).
				Padding(Edges(16)).ThemeBackground(ThemeColor(0.08, 0, BackgroundScale)),
			buttonSection("Inherited Small / Subtle with nearer overrides", HStack(
				sampleButton(Text("Inherited")),
				VStack(
					sampleButton(Text("Prominent / Large")),
					sampleButton(Text("Bordered / Mini")).ButtonStyle(Bordered).ControlSize(Mini),
				).ButtonStyle(Prominent).ControlSize(Large),
				sampleButton(Text("Inherited sibling")),
			)),
		).Gap(16).Padding(Edges(16)).
			ButtonStyle(Subtle).ControlSize(Small).ThemeBackground(theme.base))
	}
	return VStack(panels...).Gap(16).Class("button-contexts")
}
