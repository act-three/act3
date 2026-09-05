package ui

// Accent is the theme's accent color.
var Accent Color = newColor(themeAccent{})

// These colors are useful for foreground elements, such as text.
var (
	// Primary is a color suitable for primary content.
	Primary Color = primary

	// Secondary is a color suitable for secondary content.
	Secondary Color = secondary

	// Tertiary is a color suitable for tertiary content.
	Tertiary Color = tertiary

	// Headline is a color suitable for title and headline content.
	Headline Color = headline
)

// These are basic named colors.
var (
	// Red is a red color.
	Red Color = OKLCH(0.576, 0.209, 29.5)

	// Blue is a blue color.
	Blue Color = OKLCH(0.576, 0.209, 263)

	// Pink is a pink color.
	Pink Color = OKLCH(0.576, 0.209, 354)

	// Black is a black color.
	Black Color = OKLCH(0, 0, 0)

	// White is a white color.
	White Color = OKLCH(1, 0, 0)
)

var (
	// backgroundColor is the background of the current context.
	backgroundColor Color = newColor(themeBackground{})

	// baseHover Color = ModeColor(
	//	ThemeColor(0.030, 0, BackgroundScale),
	//	ThemeColor(0.037, 0.0017, BackgroundScale),
	// )

	// subColor is the background of a region set back from the background,
	// such as a sidebar.
	// subColor Color = ModeColor(
	//	ThemeColor(0.030, 0, BackgroundScale),
	//	ThemeColor(-0.028, 0, BackgroundScale),
	// )

	// shadeColor is the background of an inset region,
	// such as a card or a text field.
	// shadeColor Color = ModeColor(
	//	ThemeColor(0.047, 0, BackgroundScale),
	//	ThemeColor(0.017, 0.0017, BackgroundScale),
	// )
	// shadeHover Color = ModeColor(
	//	ThemeColor(0.060, 0, BackgroundScale),
	//	ThemeColor(0.026, 0.0033, BackgroundScale),
	// )

	// focusColor is the background of the item with keyboard focus
	// in a list or menu.
	// focusColor Color = ModeColor(
	//	ThemeColor(0.043, 0, BackgroundScale),
	//	ThemeColor(0.078, 0.0017, BackgroundScale),
	// )

	// elevatedColor is the background of a region raised above the page,
	// such as a dialog.
	// It is lighter than the background in both modes.
	// elevatedColor Color = ModeColor(
	//	ThemeColor(-0.069, 0, BackgroundScale),
	//	ThemeColor(0.036, 0.0017, BackgroundScale),
	// )

	// menuColor is the background of a menu.
	// It is lighter than the background in both modes.
	// menuColor Color = ModeColor(
	//	ThemeColor(-0.069, 0, BackgroundScale),
	//	ThemeColor(0.069, 0.0017, BackgroundScale),
	// )

	// selectedBackground is the background of a selected item.
	// It is the background tinted toward the accent.
	selectedBackground Color = newColor(selectedColor{})
	// selectedHover Color = ModeColor(
	//	newColor(themeColor{selectedColor{}, 0.017, 0, BackgroundScale}),
	//	newColor(themeColor{selectedColor{}, 0.022, 0.0067, BackgroundScale}),
	// )
	// selectedBorder Color = ModeColor(
	//	newColor(themeColor{selectedColor{}, 0.030, -0.0033, BorderScale}),
	//	newColor(themeColor{selectedColor{}, 0.030, 0.0033, BorderScale}),
	// )

	// borderFaint Color = ModeColor(
	//	ThemeColor(0.0086, -0.0033, BorderScale),
	//	ThemeColor(0.017, 0.0017, BorderScale),
	// )
	// borderFaintHover Color = ModeColor(
	//	ThemeColor(0.017, -0.0033, BorderScale),
	//	ThemeColor(0.024, 0.0017, BorderScale),
	// )
	borderColor Color = ModeColor(
		ThemeColor(0.030, -0.0033, BorderScale),
		ThemeColor(0.034, 0.0017, BorderScale),
	)
	// borderHover Color = ModeColor(
	//	ThemeColor(0.039, -0.0033, BorderScale),
	//	ThemeColor(0.043, 0.0017, BorderScale),
	// )
	// borderSolid Color = ModeColor(
	//	ThemeColor(0.043, -0.0033, BorderScale),
	//	ThemeColor(0.043, 0.0017, BorderScale),
	// )
	// borderSolidHover Color = ModeColor(
	//	ThemeColor(0.078, -0.0033, BorderScale),
	//	ThemeColor(0.060, 0.0017, BorderScale),
	// )
	// borderStrong Color = ModeColor(
	//	ThemeColor(0.147, -0.0033, BorderScale),
	//	ThemeColor(0.172, 0.0017, BorderScale),
	// )
	// borderStrongHover Color = ModeColor(
	//	ThemeColor(0.181, -0.0033, BorderScale),
	//	ThemeColor(0.207, 0.0017, BorderScale),
	// )

	// Faces of secondary and tertiary controls.
	// controlSecondary Color = ModeColor(
	//	ThemeColor(-0.052, 0, ControlScale),
	//	ThemeColor(0.103, 0.0025, ControlScale),
	// )
	// controlSecondaryHover Color = ModeColor(
	//	ThemeColor(0.052, -0.0033, ControlScale),
	//	ThemeColor(0.207, 0.0058, ControlScale),
	// )
	// controlSecondarySelected Color = ModeColor(
	//	ThemeColor(0.078, -0.0033, ControlScale),
	//	ThemeColor(0.293, 0.0058, ControlScale),
	// )
	// controlTertiary Color = ModeColor(
	//	ThemeColor(-0.052, 0, ControlScale),
	//	ThemeColor(0.103, 0.0017, ControlScale),
	// )
	// controlTertiaryHover Color = ModeColor(
	//	ThemeColor(0.078, 0, ControlScale),
	//	ThemeColor(0.190, 0.0017, ControlScale),
	// )
	// controlTertiarySelected Color = ModeColor(
	//	ThemeColor(0.112, 0, ControlScale),
	//	ThemeColor(0.250, 0.0050, ControlScale),
	// )

	// accentTextColor is the foreground color for text
	// set on top of the accent color.
	accentTextColor Color = newColor(accentText{})

	// accentHover Color = ModeColor(
	//	newColor(themeColor{themeAccent{}, 0.052, -0.0067, BackgroundScale}),
	//	newColor(themeColor{themeAccent{}, 0.043, 0.0067, BackgroundScale}),
	// )

	headline = newColor(compositeColor{
		l: ModeColor(
			ThemeColor(0.086, 0, ForegroundScale),
			ThemeColor(0, 0, ForegroundScale),
		).color(),
		c: oklch{},
		h: themeBackground{},
		a: oklch{a: 1},
	})
	primary = ModeColor(
		ThemeColor(0.172, 0.0033, ForegroundScale),
		ThemeColor(0.086, 0.0033, ForegroundScale),
	)
	secondary = ThemeColor(0.345, 0.0033, ForegroundScale)
	tertiary  = ThemeColor(0.569, 0.0033, ForegroundScale)
	linkColor = newColor(compositeColor{
		l: ThemeColor(0.388, 0, ForegroundScale).color(),
		c: oklch{c: 0.233},
		h: themeAccent{},
		a: oklch{a: 1},
	})
)
