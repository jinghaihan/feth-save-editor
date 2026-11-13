package app

import "github.com/egoist/mygo/ui"

// The C# application explicitly requested Light. These opaque layers match
// its SukiUI window/card/control appearance without requiring glass effects.
// Keep the palette and container metrics here so every page uses one style.
var editorAppearance = struct {
	Window, Card            ui.Color
	CardRadius, CardPadding float32
	ControlGap, SectionGap  float32
}{
	Window:      ui.Hex("#e7e7e7"),
	Card:        ui.Hex("#f2f2f2"),
	CardRadius:  20,
	CardPadding: 20,
	ControlGap:  8,
	SectionGap:  12,
}

var editorTheme = func() *ui.Theme {
	t := ui.LightTheme()
	t.Background = ui.Hex("#fefefe")
	t.Surface = ui.Hex("#f9f9f9")
	t.SurfaceHover = ui.Hex("#ececec")
	t.SurfacePressed = ui.Hex("#dfdfdf")
	t.Border = ui.Hex("#e2e2e2")
	t.Text = ui.Hex("#222222")
	t.TextMuted = ui.Hex("#7f7f7f")
	t.Accent = ui.Hex("#003dff")
	t.AccentHover = ui.Hex("#0037e5")
	t.AccentPressed = ui.Hex("#0030cc")
	t.Selection = ui.RGBA(0, 61, 255, 0.25)
	t.Focus = ui.RGBA(0, 61, 255, 0.55)
	t.Radius = 8
	t.FontSize = 13
	return t
}()

var listTheme = func() *ui.Theme {
	t := *editorTheme
	t.Accent = ui.Hex("#abc5f7")
	t.AccentText = editorTheme.Accent
	return &t
}()

func themedList(c *ui.Context, state *ui.ListState, count int, row func(int)) ui.Element {
	var list ui.Element
	ui.Themed(c, listTheme).Grow(1).Basis(0).MinHeight(0).Children(func() {
		list = ui.List(c, state, count, row).Grow(1).MinHeight(0).
			Background(editorAppearance.Card).Gap(2)
	})
	return list
}

func editorForm(c *ui.Context, fields func()) ui.Element {
	return ui.Form(c, fields).Gap(editorAppearance.ControlGap)
}

func editorField(c *ui.Context, label string, control func()) ui.Element {
	return ui.Field(c, label, control).Gap(editorAppearance.ControlGap)
}
