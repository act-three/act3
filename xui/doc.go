/*
Package ui is a UI toolkit for [ily.dev/domi] applications.

Application code composes views,
then serves the View graph with a [Handler].

	func (app *App) View(ctx context.Context) View {
		return VStack(
			Text("Movies").
				Title("Movies").
				Font(Title),
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
		Foreground(CSSColor("blue"))

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
			CSSColor("red"),
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
		CSSColor("blue").
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
*/
package ui
