package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

type numberDraft struct {
	Text    string
	Focused bool
}

// numberField uses MyGo's standard text input for numeric entry.
func (a *App) numberField(c *ui.Context, label string, value *string) ui.Element {
	var input ui.Element
	editorField(c, a.T(label), func() { input = ui.TextInput(c, value) })
	return input
}

func (a *App) parseNumber(text, label string, lo, hi int64) (int64, bool) {
	v, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil || v < lo || v > hi {
		a.Status = fmt.Sprintf("%s must be a whole number from %d to %d.", a.T(label), lo, hi)
		return 0, false
	}
	return v, true
}

func (a *App) applyBatch(ops ...save.Operation) bool {
	if err := a.Save.Apply(a.DB, ops); err != nil {
		a.Status = err.Error()
		return false
	}
	a.Status = ""
	return true
}

func (a *App) twoColumns(c *ui.Context, left, right func()) {
	a.columns(c, 0, left, right)
}

func (a *App) columns(c *ui.Context, rightMinimum float32, left, right func()) {
	ui.Row(c).Gap(editorAppearance.SectionGap).AlignItems(ui.Start).Children(func() {
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(editorAppearance.SectionGap).Children(left)
		ui.Column(c).Grow(1).Basis(0).MinWidth(rightMinimum).Gap(editorAppearance.SectionGap).Children(right)
	})
}

func (a *App) section(c *ui.Context, title string, contents func()) {
	a.card(c, a.T(title)).Children(func() {
		ui.Text(c, a.T(title)).Bold()
		ui.Column(c).Gap(editorAppearance.ControlGap).Children(contents)
	})
}

func (a *App) scrollingSection(c *ui.Context, title string, contents func()) {
	a.card(c, a.T(title)).Grow(1).Basis(0).MinHeight(0).Children(func() {
		ui.Text(c, a.T(title)).Bold()
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(editorAppearance.ControlGap).Children(contents)
		})
	})
}

func (a *App) groupBox(c *ui.Context, title string) ui.Element {
	return ui.Column(c).Padding(editorAppearance.CardPadding).Gap(editorAppearance.SectionGap).MinWidth(0).Role(ui.RoleGroup).Label(a.T(title))
}

func (a *App) group(c *ui.Context, title string, contents func()) {
	a.groupBox(c, title).Children(func() {
		ui.Text(c, a.T(title)).Bold()
		ui.Column(c).Gap(editorAppearance.ControlGap).Children(contents)
	})
}

// Use MyGo's gallery container composition with the shared editor appearance.
func (a *App) card(c *ui.Context, label string) ui.Element {
	t := c.Theme()
	return ui.Column(c).Padding(editorAppearance.CardPadding).Gap(editorAppearance.SectionGap).Radius(editorAppearance.CardRadius).
		Background(cardColor(c)).Border(1, t.Border).MinWidth(0).Role(ui.RoleGroup).Label(label)
}

// selection keeps each old list's own search, selection and stable source ID.
func (a *App) selection(c *ui.Context, key, title string, rows []row, footer func()) int {
	s := a.table(key)
	previousID := -1
	if s.Selected >= 0 && s.Selected < len(s.Rows) {
		previousID = s.Rows[s.Selected].ID
	}
	ui.Column(c.Key(key)).Fill().Gap(editorAppearance.ControlGap).Children(func() {
		if title != "" {
			ui.Text(c, a.T(title))
		}
		ui.SearchField(c, &s.Query).Label(a.T("Search...")).Placeholder(a.T("Search..."))
		filtered := make([]row, 0, len(rows))
		for _, r := range rows {
			if strings.Contains(strings.ToLower(fmt.Sprintf("%d %s %s", r.ID, r.Name, r.Value)), strings.ToLower(s.Query)) {
				filtered = append(filtered, r)
			}
		}
		if previousID >= 0 {
			s.Selected = 0
			for i, r := range filtered {
				if r.ID == previousID {
					s.Selected = i
					break
				}
			}
		}
		s.Rows = filtered
		if s.Selected >= len(filtered) {
			s.Selected = max(0, len(filtered)-1)
		}
		s.List.Selected = &s.Selected
		s.List.Key = func(i int) any { return filtered[i].ID }
		if themedList(c, &s.List, len(filtered), func(i int) {
			r := filtered[i]
			text := r.Name
			if r.Value != "" {
				text += " · " + r.Value
			}
			ui.Text(c, text).SingleLine().Padding(7, 10)
		}).Grow(1).MinHeight(0).Changed() {
			a.flushInput()
		}
		if footer != nil {
			footer()
		}
	})
	if s.Selected < 0 || s.Selected >= len(s.Rows) {
		return -1
	}
	return s.Rows[s.Selected].ID
}

func (a *App) editorLayout(c *ui.Context, key, title string, rows []row, footer func(), detail func(int)) {
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(editorAppearance.SectionGap).AlignItems(ui.Stretch).Children(func() {
		width := float32(320)
		if key == "roster" || key == "inheritance" {
			width = 264
		}
		if key == "roster" {
			windowWidth, _ := c.Size()
			width = max(220, width*min(float32(1), windowWidth/1120))
		}
		id := -1
		a.card(c, a.T(title)).Width(width).MinHeight(0).Children(func() { id = a.selection(c, key, title, rows, footer) })
		editorTitle := map[string]string{"roster": "Character Editor", "inheritance": "NG+ Roster", "supports": "Support Editor", "ng-supports": "Inherited support rank", "battalions": "Battalion Editor", "quests": "Quest Editor"}[key]
		a.card(c, a.T(editorTitle)).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c.Key(id)).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Gap(editorAppearance.SectionGap).Children(func() {
				if id >= 0 || a.Save == nil {
					detail(max(0, id))
				}
			})
		})
	})
}
