/*
Package ui is a UI toolkit for [ily.dev/domi] applications.

Application code composes views,
then renders the View graph with a [Renderer].

	func (app *App) View(ctx context.Context) (string, domi.Node) {
		return "Movies", app.ui.Render(
			VStack(
				Text("Movies").
					Font(Title),
				Image(bannerURL).
					FramedAs(ScaleToFill).
					Frame(Width(800), Height(200)),
				For(movies, movie.id, func(m *movie) View {
					return HStack(
						Text(m.title),
					)
				}),
			),
		)
	}

View modifiers affect the appearance, sizing, and other properties
of the views they modify.

	Text("This is blue").
		Foreground("blue")

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

	ScrollView(
		ZStack(
			Color("red"),
		).
			Frame(Width(100), Height(100)),
	)

The available space for the ZStack is 100px square,
determined by its enclosing frame view.
The available space for the color red is also 100px square,
determined by its enclosing ZStack,
which grants its own available space to all its subviews.
The available space for the frame is unbounded,
which is determined by its enclosing scroll view.

When determining available space,
some views, like the frame above,
grant a fixed amount of available space to their subviews.
Other views, like a ZStack,
grant whatever available space they have to their subviews.
Still other views, like an HStack or VStack,
have more complex behavior.
In general, any enclosing view determines the available space
for each of its subviews,
possibly by incorporating its own available space.

When determining a view's size,
some views, such as Color,
expand to fill available space.
Other views, such as the 100px frame above,
occupy a fixed area regardless of the available space.
Still other views, such as HStack, VStack, and Text,
have more complex sizing behavior.
In general, each view determines its own size,
possibly by responding in some way to the available space.

It is possible for a view to exceed its available space.

	HStack(
		Color("blue").
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

# CSS

The companion stylesheet is exposed as [CSS].
Serve it once per page.
A [Renderer] emits CSS for dynamic style values with each rendered page.

All CSS rules are declared in the "xui" cascade layer.
*/
package ui
