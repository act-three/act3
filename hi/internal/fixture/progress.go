package fixture

import . "ily.dev/act3/hi"

func indeterminateProgressDemo() View {
	var controls []View
	for _, size := range []struct {
		name string
		size ControlSize
	}{
		{"Mini", Mini},
		{"Small", Small},
		{"Regular", Regular},
		{"Large", Large},
	} {
		controls = append(
			controls,
			HStack(
				Progress(),
				Text(size.name),
			).
				Alignment(FirstBaseline).
				ControlSize(size.size),
		)
	}
	return HStack(controls...).
		Gap(40).
		Alignment(FirstBaseline).
		Font(SizeCap(12, 2)).
		Padding(EdgesLetterbox(12))
}

func determinateProgressDemo() View {
	return VStack(
		section("0%", Progress(0)),
		section("25%", Progress(0.25)),
		section("100%", Progress(1)),
	).
		Gap(16).
		Frame(Width(320))
}
