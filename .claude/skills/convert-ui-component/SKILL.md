---
name: convert-ui-component
description: Convert a ui/ component from inline Tailwind utilities to Radix-style component CSS with `a$` prefix. Use when asked to convert, migrate, or restyle a ui/ component.
---

# Convert UI Component to Component CSS

Convert a `ui/` component from inline Tailwind utility classes to
semantic `a$`-prefixed CSS classes in a per-component stylesheet.

## Naming conventions

- Base class: `a$<component>` (e.g. `a$button`, `a$badge`)
- Variant classes use `+` separator: `a$<component>+<variant>`
  (e.g. `a$button+solid`, `a$button+size-2`)
- In CSS selectors, escape `$` and `+`: `.a\$button\+solid`
- In Go strings, no escaping needed: `"a$button+solid"`

## Steps

### 1. Read the existing Go file

Read `ui/<component>.go`. Identify:

- The base Tailwind class string (usually a multi-line const)
- The variant/size/shape table maps (e.g. `map[xxxVariant]string`)
- The `FuncAttr("class", ...)` calls that resolve them
- The public API vars (`ButtonSolid`, `ButtonSize2`, etc.)

### 2. Rewrite the Go file

Replace the Tailwind class strings with `a$`-prefixed class names.

**Remove:**
- The base const with inline Tailwind utilities
- All `map[...]string` tables that hold Tailwind class lists

**Add:**
- `attr.Class("a$<component>")` as the base class
- New `map[...]string` tables returning `a$` class names, e.g.:

```go
var fooVariantClasses = map[fooVariant]string{
    fooSolid: "a$foo+solid",
    fooSoft:  "a$foo+soft",
}
```

**Keep unchanged:**
- The `FuncAttr("class", ...)` pattern (just update table names)
- All public API vars (`FooSolid`, `FooSize2`, etc.)
- Type definitions and iota consts

### 3. Write the CSS file

Create or replace `ui/<component>.css`. All styles go inside
`@layer components { ... }` so that Tailwind utilities can
override them.

Follow these patterns from Radix Themes:

**Base selector** — layout, typography, transitions:
```css
@layer components {

.a\$foo {
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
.a\$foo:focus-visible {
    outline: 2px solid var(--color-accent-8);
    outline-offset: 2px;
}
```

**Disabled** — cover both `:disabled` and `[aria-disabled]`:
```css
.a\$foo:disabled,
.a\$foo[aria-disabled="true"] {
    pointer-events: none;
    opacity: 0.5;
}
```

**Hover** — wrap in `@media (hover: hover)` so touch devices
don't get sticky hover:
```css
@media (hover: hover) {
    .a\$foo\+solid:hover {
        background-color: var(--color-accent-10);
    }
}
```

**Active/press** — one step deeper than hover, exclude disabled:
```css
.a\$foo\+solid:active:not(:disabled) {
    background-color: var(--color-accent-10);
    filter: brightness(0.92) saturate(1.1);
}
```

**Borders** — use `box-shadow: inset 0 0 0 1px` instead of
`border` to avoid layout shift:
```css
.a\$foo\+outline {
    box-shadow: inset 0 0 0 1px var(--color-accent-8);
}
```

**Colors** — use CSS custom properties from the Tailwind
`@theme`, e.g. `var(--color-accent-9)`, `var(--color-crimson-9)`.

### 4. Import the CSS

Make sure `web/main.css` imports the new stylesheet:
```css
@import "../ui/<component>.css";
```

### 5. Verify

```sh
go build -tags goexperiment.jsonv2 .
go test -tags goexperiment.jsonv2 ./ui/...
```

Run locally and visually check the component renders correctly.

## Reference

See `ui/button.go` and `ui/button.css` as the canonical example.
