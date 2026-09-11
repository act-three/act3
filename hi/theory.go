package hi

// Stability Rule
//
// Any given view structure should result in a fixed lowering
// structure regardless of the values given as arguments. View
// structure is the set of views and modifiers and how they're
// arranged. Lowering structure is the set of HTML elements and
// text nodes and how they're arranged.
//
// Consider the following view structure:
//
//     ScrollView(Vertical, detail).
//         BorderStroke(px, c)
//
// And suppose it ordinarily lowers to the following HTML:
//
//     <hi-border-stroke ...>
//         <hi-scroll ...>
//             ...
//         </hi-scroll>
//     </hi-border-stroke>
//
// It must always lower to that structure, regardless of the
// values of px and c. In particular, it might be tempting to
// omit the hi-border-stroke element if px is 0 or if c is
// transparent, since in that case, there is no visible stroke.
// This type of optimization is prohibited.
//
// This is a result of the domi diff algorithm. If an element
// is moved to a different position in the HTML tree, even if
// it has identical contents before and after, the entire
// subtree rooted at that element is replaced. This means focus
// and scroll position within that subtree are lost, and keyed
// children are also replaced.
//
// Authors need to be able to reason about these consequences
// precisely, so the HTML structure should change only when the
// authored view structure changes.
//
// Lowering structure must not depend on argument values that
// leave a view's arity unchanged. But a change in a subview's
// arity may change the containment used to compose the
// subview's nodes.
//
// Consider button labels:
//
//     Button(msg, Group(Text("x"), If(show, Text("y"))))
//
// Suppose this lowers to the following HTML when show is true:
//
//     <button ...>
//         <hi-hstack ...>
//             <hi-text ...>x</hi-text>
//             <hi-text ...>y</hi-text>
//         </hi-hstack>
//     </button>
//
// Changing the number of views in the button label may change
// the level of the HTML tree where the label is emitted. When
// show is false, the lowering may omit the hi-hstack element
// and include the subview directly:
//
//     <button ...>
//         <hi-text ...>x</hi-text>
//     </button>
//
// Consequently, a transition between one and multiple label
// views may replace the surviving label subtree, even if keyed.
//
// App authors should wrap a variable-arity view in an explicit
// container when stable containment is required. The following
// view reliably remains at the same containment level:
//
//     Button(msg, HStack(Text("x"), If(show, Text("y"))))
//
// Note that the stability rule is not formalized. It requires
// judgement to apply correctly. The guiding principle is to avoid
// surprising the app author with unexpected structural changes.
//
// Paint Regions
//
// A paint modifier conceptually owns a layout-preserving wrapper.
// It knows its subview's bounding rectangle and its own environment,
// including the border shape. It does not inspect the subview's paint.
//
//   - Background paints inside the border shape behind the subview.
//   - BorderStroke paints inside the border shape in front.
//   - BorderShadow paints outside the border shape behind.
//   - BorderOutline paints outside the border shape in front.
//
// The shadow paints exclusively outside the border shape, which means
// its interior is empty even when the subview is transparent.
//
// These regions let the lowering collect the different paint families
// independently on one box, regardless of their interleaving. Within
// each family, backgrounds and shadows paint inner modifiers in front.
// Strokes paint outer modifiers in front. Transforms and layout
// changes retain their ordinary boxing boundaries. In particular,
// opacity and clipping outside a shadow affect it. The same modifiers
// inside do not.
//
// Outlines resolve by precedence instead of accumulating paint layers.
// A box has at most one outline. When an outline is painted, its
// conceptual wrapper adds a signal to enclosing outline modifiers that
// the view has an outline, and the enclosing outlines are not painted.
// Thus, the innermost outline wins. Gap, width, and color resolve
// together for each state combination. A state-scoped outline exists
// only while its state applies. Otherwise there is no outline and no
// presence to suppress an outer outline. Transparent and zero-width
// outlines are present and take precedence. Inner boxes have their own
// outlines. Presence is not discovered by inspecting subtree paint.
// Ordinary transform and layout boundaries still separate boxes. The
// winning outline lowers to CSS outline on the foreground carrier
// shared with strokes.
//
// Z-Index Rule
//
// The CSS property z-index must be applied only inside an explicit
// stacking context. There is no global z-index. The stacking context
// can be created by CSS property isolation:isolate or any other
// reliable mechanism. An isolated stacking context creates a local
// scope for z-index values. They are unable to conflict with paint
// order outside the scope.
//
// This avoids so-called "z-index wars" where modifying a z-index
// value in one part of an HTML document requires modifying one or
// more z-index values in unrelated places.
//
// If cross-subtree layering is needed, it should use some sort of
// portal mecanism.
//
// Test Coverage
//
// Tests should cover interaction functionality, layout, and the
// behavior of individual views and modifiers. Tests for components
// should cover their functionality but should not pin appearance.
// Only TestGolden should include component appearance.
//
// The goal is for component styling to evolve without updating
// assertions about chosen colors, borders, typography, or
// disabled opacity.
//
// Tests should retain layout, action semantics, environment
// propagation, and modifier behavior for modifiers used by the
// app author.
