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
		t.Accent = ui.Hex("#bda4d8")
		t.AccentHover = ui.Hex("#ccb6e2")
		t.AccentPressed = ui.Hex("#ac91cb")
		t.AccentText = ui.Hex("#21182e")
		t.Selection = ui.RGBA(189, 164, 216, 0.30)
		t.Focus = ui.RGBA(189, 164, 216, 0.75)
	} else {
		t.Background = ui.Hex("#e7e7e7")
		t.Surface = ui.Hex("#f9f9f9")
		t.SurfaceHover = ui.Hex("#ececec")
		t.SurfacePressed = ui.Hex("#dfdfdf")
		t.Border = ui.Hex("#e2e2e2")
		t.Text = ui.Hex("#222222")
		t.TextMuted = ui.Hex("#7f7f7f")
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
		return ui.Hex("#222225")
	}
	return ui.Hex("#f2f2f2")
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
