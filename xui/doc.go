/*
Package ui is a UI toolkit for [ily.dev/domi] applications.

Application code composes views,
then serves the View graph with a [Handler].

	func (app *App) View(ctx context.Context) View {
		return VStack(
			Text("Movies").
				Title("Movies").
				Font(SizeCap(20i, 1.2), Bold),
			Image(bannerURL).
				ScaledToFill().
				Frame(Width(800), Height(200)),
			For(movies, movie.id, func(m *movie) View {
				return HStack(
					Text(m.title),
				)
			}),
		)
	}

View modifiers affect the appearance, sizing, and other properties
of the views they modify.

	Text("This is blue").
		Foreground(Blue)

In this example,
the Foreground modifier changes the text color of the Text view.

Views are immutable.
Every modifier returns a modified copy of its underlying view.
This makes views safe to reuse and share.

# Layout

Each view in a layout is situated within some amount of "available space".
The available space is determined independently for each axis.
It can be a definite amount or unbounded.

For instance, consider the following view graph.

	ScrollView(Vertical|Horizontal,
		ZStack(
			Red,
		).
			Frame(Width(100), Height(100)),
	)

The available space for the ZStack is 100px square,
determined by its enclosing frame.
The available space for the color red is also 100px square,
determined by its enclosing ZStack,
which provides the 100px it gets from the frame to the Color.
The available space for the frame is unbounded,
which is determined by its enclosing scroll view.

An enclosing view determines the available space
for each of its subviews.

  - Some views, like the 100px frame above,
    provide a fixed amount of available space to their subviews.
  - Some views, like ZStack,
    provide their own available space to their subviews.
  - Some views, like an HStack or VStack,
    have more complex behavior.

Each view determines its own size.

  - Some views, such as Color, expand to fill available space.
  - Some views, like VStack, HStack, and ZStack,
    adopt a size based on their subviews.
  - Some views, such as the 100px frame above,
    occupy a fixed area regardless of the available space.
  - Some views, such as Text,
    have more complex sizing behavior.

It is possible for a view to take a size
that exceeds its available space.

	HStack(
		Blue.
			Frame(Width(100), Height(100)).
			Frame(Width(50), Height(50)),
		Text("Hello"),
	)

The available space for the inner frame is a 50px square,
determined by the outer frame.
But the inner frame is a 100px square,
which exceeds its available space.
Because the inner frame is larger than the outer frame,
the blue square overlaps the word "Hello",
which is an adjacent sibling of the outer frame.

# Length Units

Lengths can be specified in scaled or unscaled pixel units.
Unscaled lengths map directly to the CSS px unit.
Scaled lengths are scaled in proportion
to the root font size (CSS rem unit) with a 16px basis:

  - When the root font size is 16px, a scaled length of 1 is the CSS length 1px.
  - When the root font size is 24px, a scaled length of 1 is the CSS length 1.5px.

In most web browsers, 16px is the default root font size.

Lengths support linear arithmetic:

  - Any two lengths can be added together (or subtracted),
    including a mix of scaled and unscaled lengths.
  - Any length can be multiplied (or divided) by a real-valued scalar.

Do not multiply or divide two length values,
or take the reciprocal of a length.
These operations are not closed over lengths.
Although they compile,
they do not generally produce useful results.

Lengths are represented by complex128 values.
The real coefficient encodes unscaled length,
and the imaginary coefficient encodes scaled length.

Unscaled lengths are denoted by real number literals,
such as 2, 0.5, and 0x1p-2.
To add padding that is always 8 CSS pixels wide,
use a real-valued (unscaled) length:

	view.Padding(Edges(8))

This produces padding that is always 8px.

Scaled lengths are denoted by [imaginary literals],
such as 2i, 0.5i, and 0x1p-2i.
To add padding that gets bigger as the root font size increases,
use an imaginary (scaled) length:

	view.Padding(Edges(8i))

This produces 8px padding at a 16px root font size,
and 12px padding at a 24px root font size.

Scaled and unscaled lengths can be added,
and it is valid to combine them:

	view.Padding(Edges(3 + 8i))

This produces 11px padding at a 16px root font size,
and 15px padding at a 24px root font size.

# Serving Client Assets

A xui page requires two static assets:
a CSS stylesheet and a JavaScript module.
The default behavior of [Handler] includes both.

Apps that provide their own document shell (see [domi.Document])
must serve these assets themselves.
There are two ways to do it.

  - Serve each asset directly,
    using [Stylesheet] and [domi.ClientModule].
  - Bundle the assets with additional CSS and JavaScript.

Apps that serve their own CSS and JavaScript
might wish to bundle the assets into those files.
Obtain the filesystem paths of the asset sources by running:

	go list -f '{{.Dir}}/ui.css' ily.dev/act3/xui
	go list -f '{{.Dir}}/client.js' ily.dev/domi

Include ui.css in the app's CSS bundle.
Include client.js in the app's JavaScript bundle,
then import the module and call run:

	import * as Domi from "path/to/bundle.js";
	Domi.run();

# CSS Layer

There are two types of CSS rules defined in this package.
A static stylesheet is documented in Serving Client Assets.
[Handler] also emits a dynamically-generated stylesheet
in each rendered page.

All CSS rules of both types are declared in the "xui" cascade layer.

[imaginary literals]: https://go.dev/ref/spec#Imaginary_literals
*/
package ui
