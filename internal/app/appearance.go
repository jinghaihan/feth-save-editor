package app

import "github.com/egoist/mygo/ui"

// Keep the original container geometry; widgets use MyGo's standard themes.
var editorAppearance = struct {
	CardRadius, CardPadding float32
	ControlGap, SectionGap  float32
}{20, 20, 8, 12}

func makeEditorTheme(dark bool) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
		t.Background = ui.Hex("#1d1923")
		t.Surface = ui.Hex("#332c3d")
		t.SurfaceHover = ui.Hex("#40364c")
		t.SurfacePressed = ui.Hex("#4a3e57")
		t.Border = ui.Hex("#51455f")
		t.Text = ui.Hex("#f0eade")
		t.TextMuted = ui.Hex("#b7adbf")
		t.Accent = ui.Hex("#bda4d8")
		t.AccentHover = ui.Hex("#ccb6e2")
		t.AccentPressed = ui.Hex("#ac91cb")
		t.AccentText = ui.Hex("#21182e")
		t.Selection = ui.RGBA(189, 164, 216, 0.30)
		t.Focus = ui.RGBA(189, 164, 216, 0.75)
	} else {
		t.Background = ui.Hex("#e8e4de")
		t.Surface = ui.Hex("#fcf9f2")
		t.SurfaceHover = ui.Hex("#eee7ed")
		t.SurfacePressed = ui.Hex("#e0d5e6")
		t.Border = ui.Hex("#d3cad8")
		t.Text = ui.Hex("#302a37")
		t.TextMuted = ui.Hex("#746a7e")
		t.Accent = ui.Hex("#70538e")
		t.AccentHover = ui.Hex("#62457f")
		t.AccentPressed = ui.Hex("#52376e")
		t.AccentText = ui.Hex("#fffaf1")
		t.Selection = ui.RGBA(112, 83, 142, 0.22)
		t.Focus = ui.RGBA(112, 83, 142, 0.65)
	}
	t.Radius = 8
	t.FontSize = 13
	return t
}

var lightEditorTheme = makeEditorTheme(false)
var darkEditorTheme = makeEditorTheme(true)

func (a *App) currentTheme() *ui.Theme {
	if a.dark {
		return darkEditorTheme
	}
	return lightEditorTheme
}
func cardColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#292331")
	}
	return ui.Hex("#f4f0e8")
}
func themedList(c *ui.Context, state *ui.ListState, count int, row func(int)) ui.Element {
	t := *c.Theme()
	if t.Dark {
		t.Accent = ui.Hex("#554165")
		t.AccentText = ui.Hex("#f4eafd")
	} else {
		t.Accent = ui.Hex("#e0d2ec")
		t.AccentText = ui.Hex("#49315f")
	}
	var list ui.Element
	ui.Themed(c, &t).Grow(1).Basis(0).MinHeight(0).Children(func() { list = ui.List(c, state, count, row).Grow(1).MinHeight(0).Background(cardColor(c)).Gap(2) })
	return list
}
func editorForm(c *ui.Context, fields func()) ui.Element {
	return ui.Form(c, fields).Gap(editorAppearance.ControlGap)
}
func editorField(c *ui.Context, label string, control func()) ui.Element {
	return ui.Field(c, label, control).Gap(editorAppearance.ControlGap)
}
