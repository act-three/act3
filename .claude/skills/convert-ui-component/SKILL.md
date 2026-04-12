---
name: convert-ui-component
description: Convert a ui/ component from inline Tailwind utilities to Radix-style component CSS with `u-` prefix. Use when asked to convert, migrate, or restyle a ui/ component.
---

# Convert UI Component to Component CSS

Convert a `ui/` component from inline Tailwind utility classes to
semantic `u-`-prefixed CSS classes in a per-component stylesheet.

## Naming conventions

- Base class: `u-<component>` (e.g. `u-button`, `u-badge`)
- Variant classes use `+` separator: `u-<component>+<variant>`
  (e.g. `u-button+solid`, `u-button+size-2`)
- In CSS selectors, escape `+`: `.u-button\+solid`
- In Go strings, no escaping needed: `"u-button+solid"`

## Steps

### 1. Read the existing Go file

Read `ui/<component>.go`. Identify:

- The base component class name
- The variant/size/shape names & classes (e.g. `u-button+ghost`,
  `u-button+size2`)
- The `FuncAttr("class", ...)` calls, if any, that resolve them
- The public API vars (`ButtonSolid`, `ButtonSize2`, etc.)

### 2. Rewrite the Go file

Replace the Tailwind class strings with `u-`-prefixed class names.

**Remove:**
- The base const with inline Tailwind utilities
- All `map[...]string` tables that hold Tailwind class lists

**Add:**
- `attr.Class("u-<component>")` as the base class
- `attr.Class("u-<component>+<variant>")` for variant class names

**Keep unchanged:**
- The `FuncAttr("class", ...)` pattern (just update table names)
- All public API vars (`FooSolid`, `FooSize2`, etc.)

### 3. Write the CSS file

Create or replace `ui/<component>.css`. All styles go inside
`@layer components { ... }` so that Tailwind utilities can
override them.

Follow these patterns from Radix Themes:

**Base selector** — layout, typography, transitions:
```css
@layer components {

.u-foo {
    position: relative;
    display: inline-flex;
    /* ... */
    outline: none;
    cursor: pointer;
}
```

**Focus ring** — use `outline` not `box-shadow` (survives
`overflow: hidden`):
```css
.u-foo:focus-visible {
    outline: 2px solid var(--color-accent-8);
    outline-offset: 2px;
}
```

**Disabled** — two flavors, `:disabled` for form controls and
`[inert]` for everything else.
Use `disabled` on form controls (input, button, textarea, select)
and `inert` on containers and non-form elements
(divs, anchors, forms, custom components).
Component CSS targeting a form control uses `:disabled`;
component CSS targeting a non-form element uses `[inert]`;
if a rule could apply to either,
include both selectors (e.g. `.u-foo:disabled, .u-foo[inert]`).
Shared cosmetic dimming (opacity, saturation) for both states
lives in the global `[disabled], [inert]` rule in `ui/theme.css`,
so component-level rules only need to add what's specific
(e.g. `cursor: not-allowed`).
Note that `[inert]` blocks hit-testing entirely
(per spec it acts like `pointer-events: none`),
so `:hover` does not fire inside an inert subtree.

**Hover** — wrap in `@media (hover: hover)` so touch devices
don't get sticky hover:
```css
@media (hover: hover) {
    .u-foo\+solid:hover {
        background-color: var(--color-accent-10);
    }
}
```

**Active/press** — one step deeper than hover, exclude disabled:
```css
.u-foo\+solid:active:not(:disabled):not([inert]) {
    background-color: var(--color-accent-10);
    filter: brightness(0.92) saturate(1.1);
}
```

**Borders** — use `box-shadow: inset 0 0 0 1px` instead of
`border` to avoid layout shift:
```css
.u-foo\+outline {
    box-shadow: inset 0 0 0 1px var(--color-accent-8);
}
```

**Colors** — use CSS custom properties from the Tailwind
`@theme`, e.g. `var(--color-accent-9)`, `var(--color-crimson-9)`.

### 4. Import the CSS

Make sure `main.css` imports the new stylesheet:
```css
@import "./ui/<component>.css";
```

### 5. Verify

```sh
go generate ./...
go build -o /dev/null -tags goexperiment.jsonv2 .
go test -tags goexperiment.jsonv2 ./ui/...
```

Run locally and use the Playwright MCP to visually verify the
component renders correctly — navigate to a page that uses the
component, take a snapshot, and confirm the layout and states
(hover, focus, disabled) look right.

## Reference

See `ui/button.go` and `ui/button.css` as the canonical example.
