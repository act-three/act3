package ui

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
//     <ui-border-stroke ...>
//         <ui-scroll ...>
//             ...
//         </ui-scroll>
//     </ui-border-stroke>
//
// It must always lower to that structure, regardless of the
// values of px and c. In particular, it might be tempting to
// omit the ui-border-stroke element if px is 0 or if c is
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
// A subtler example is button labels:
//
//     Button(msg, Group(Text("x"), Text("y")))
//
// Suppose this lowers to the following HTML:
//
//     <button ...>
//         <ui-hstack ...>
//             <ui-text ...>x</ui-text>
//             <ui-text ...>y</ui-text>
//         </ui-hstack>
//     </button>
//
// Changing the number of views in the button label should not
// change the level of the HTML tree where the label is emitted.
// It might be tempting to omit the ui-hstack element if the
// label contains only a single view node, since in that case,
// the HStack has no effect on the layout. This is also
// prohibited. It would be surprising for the entire button
// label to be replaced as a result of merely deleting "y".
//
// Note that this rule is not formalized. It requires judgement
// to apply correctly. The guiding principle is to avoid
// surprising the app author with unexpected structural changes.
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
