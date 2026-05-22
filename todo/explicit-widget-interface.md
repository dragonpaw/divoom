# Make the widget interface explicit

`internal/widget/widget.go` (or wherever the type lives — verify)
declares the central widget shape, but the concrete widgets across
`internal/widget/*/` mostly satisfy it by structural happenstance
rather than against a published interface. New widgets can drift —
add a field, miss a method — and you only notice at the call site
when the scene driver type-asserts.

## What to do

1. Check the current shape. Grep for what scene driver and
   formatters actually call on widgets — probably `Fetch(ctx) (string, error)`
   and `Name() string`, possibly `Count() int` (optional, via
   the `counter` interface in serve.go).
2. Declare a single canonical interface:
   ```go
   // Widget is the contract every rotation widget implements.
   type Widget interface {
       Name() string
       Fetch(ctx context.Context) (string, error)
   }
   ```
   in `internal/widget/widget.go`.
3. For each concrete widget, add a compile-time assertion at the
   bottom of its file:
   ```go
   var _ Widget = (*Source)(nil)   // or whatever the concrete type is
   ```
   This way the compiler catches any drift the moment a widget no
   longer matches the contract.
4. Leave optional interfaces (`Count()` etc.) as separate small
   interfaces alongside the main one — that pattern is already used.

## Why

Today's "implicit shape contract" works until someone refactors a
widget and silently breaks it for the driver. Compile-time
assertion costs one line and surfaces the break at the source, not
at the call site weeks later.
