package ui

// Accent is the theme's accent color.
var Accent Color = accent

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
// The context-dependent colors are tuned separately
// for light mode and dark mode.
var (
	// Red is a context-dependent red color.
	Red Color = red

	// Orange is a context-dependent orange color.
	Orange Color = orange

	// Yellow is a context-dependent yellow color.
	Yellow Color = yellow

	// Green is a context-dependent green color.
	Green Color = green

	// Mint is a context-dependent mint color.
	Mint Color = mint

	// Teal is a context-dependent teal color.
	Teal Color = teal

	// Cyan is a context-dependent cyan color.
	Cyan Color = cyan

	// Blue is a context-dependent blue color.
	Blue Color = blue

	// Indigo is a context-dependent indigo color.
	Indigo Color = indigo

	// Purple is a context-dependent purple color.
	Purple Color = purple

	// Pink is a context-dependent pink color.
	Pink Color = pink

	// Brown is a context-dependent brown color.
	Brown Color = brown

	// Gray is a gray color.
	Gray Color = gray

	// Black is a black color.
	Black Color = black

	// White is a white color.
	White Color = white
)

var (
	red    = ModeColor(OKLCH(0.654, 0.232, 28.7), OKLCH(0.663, 0.224, 28.3))
	orange = ModeColor(OKLCH(0.765, 0.175, 62.6), OKLCH(0.782, 0.171, 67.2))
	yellow = ModeColor(OKLCH(0.865, 0.177, 90.4), OKLCH(0.885, 0.181, 94.8))
	green  = ModeColor(OKLCH(0.730, 0.194, 147.4), OKLCH(0.756, 0.208, 147.0))
	mint   = ModeColor(OKLCH(0.748, 0.130, 189.0), OKLCH(0.851, 0.115, 192.4))
	teal   = ModeColor(OKLCH(0.700, 0.111, 212.7), OKLCH(0.771, 0.118, 212.0))
	cyan   = ModeColor(OKLCH(0.707, 0.133, 233.9), OKLCH(0.817, 0.119, 227.7))
	blue   = ModeColor(OKLCH(0.603, 0.218, 257.4), OKLCH(0.624, 0.206, 255.5))
	indigo = ModeColor(OKLCH(0.529, 0.191, 278.3), OKLCH(0.556, 0.203, 278.1))
	purple = ModeColor(OKLCH(0.615, 0.213, 312.4), OKLCH(0.656, 0.227, 312.4))
	pink   = ModeColor(OKLCH(0.650, 0.238, 17.9), OKLCH(0.658, 0.232, 16.0))
	brown  = ModeColor(OKLCH(0.632, 0.064, 72.8), OKLCH(0.665, 0.064, 73.0))
	gray   = OKLCH(0.648, 0.007, 286.2)
	black  = OKLCH(0, 0, 0)
	white  = OKLCH(1, 0, 0)
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
	shadeColor Color = ModeColor(
		ThemeColor(0.047, 0, BackgroundScale),
		ThemeColor(0.017, 0.0017, BackgroundScale),
	)
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
	controlSecondary Color = ModeColor(
		ThemeColor(-0.052, 0, ControlScale),
		ThemeColor(0.103, 0.0025, ControlScale),
	)
	controlSecondaryHover Color = ModeColor(
		ThemeColor(0.052, -0.0033, ControlScale),
		ThemeColor(0.207, 0.0058, ControlScale),
	)
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
	accentTextColor Color = textOn(themeAccent{})

	// accentHover is the face of a hovered accent control.
	accentHover Color = hoverOf(themeAccent{})

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

	accent = newColor(themeAccent{})
)
