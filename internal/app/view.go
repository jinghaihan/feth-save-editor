package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func (a *App) View(c *ui.Context) {
	c = c.Key(a.revision)
	ui.Column(c).Fill().Children(func() {
		ui.Toolbar(c, func() {
			if ui.Button(c, a.T("Open Save...")).Clicked() {
				a.open()
			}
			if ui.Button(c, a.T("Save Copy...")).Disabled(a.Save == nil).Clicked() {
				a.saveCopy()
			}
			ui.Spacer(c)
			language := a.Language
			selected := languageNames[0]
			for i, code := range languageCodes {
				if code == language {
					selected = languageNames[i]
				}
			}
			if ui.Select(c, &selected, languageNames).Label(a.T("Language")).Changed() {
				for i, name := range languageNames {
					if name == selected {
						a.SetLanguage(languageCodes[i])
					}
				}
			}
		})
		ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			ui.Sidebar(c, &a.Page, func() {
				for _, page := range pages {
					ui.SidebarItem(c, page, icon(pageIcons[page]), a.T(page))
				}
			}).Width(180)
			ui.Column(c.Key(a.Page)).Grow(1).MinWidth(0).MinHeight(0).Padding(16).Gap(12).Children(func() {
				if a.Save == nil && a.Page != "Database" && a.Page != "System" {
					ui.Text(c, a.T("Open a save to begin."))
					return
				}
				switch a.Page {
				case "Main":
					a.mainPage(c)
				case "Items":
					a.itemsPage(c)
				case "Roster":
					a.rosterPage(c)
				case "Battalions":
					a.battalionsPage(c)
				case "Quests":
					a.questsPage(c)
				case "Support":
					a.supportPage(c, false)
				case "Database":
					a.databasePage(c)
				case "NG+ Roster":
					a.inheritancePage(c)
				case "NG+ Support":
					a.supportPage(c, true)
				case "System":
					a.systemPage(c)
				}
			})
		})
		ui.Row(c).Padding(8, 12).Gap(12).Children(func() {
			if a.Path != "" {
				ui.Text(c, a.Path).SingleLine()
			}
			ui.Spacer(c)
			if a.Save != nil {
				ui.Textf(c, "%d %s", a.Save.ChangedBytes(), a.T("changed bytes"))
			}
			ui.Text(c, a.Status)
		})
	})
	if ui.AlertDialog(c, &a.confirm, a.T(a.confirmLabel), "", a.T("Cancel"), a.T("Confirm")) == 1 && a.confirmAction != nil {
		a.confirmAction()
	}
}
func (a *App) input(c *ui.Context, label, path string, lo, hi int64) {
	if minValue, maxValue, err := a.Save.NumberRange(path); err == nil {
		lo = max(lo, minValue)
		hi = min(hi, maxValue)
	}
	v := float64(a.number(path))
	ui.Field(c.Key(path), a.T(label), func() {
		if ui.NumberInput(c, &v, float64(lo), float64(hi), 1).Changed() {
			a.Set(path, int64(v))
		}
	})
}
func (a *App) choice(c *ui.Context, label string, entries []catalog.Entry, id int, changed func(int)) {
	labels := make([]string, len(entries))
	selected := fmt.Sprintf("[%d]", id)
	for i, v := range entries {
		name := v.Name
		if prefix, rest, ok := strings.Cut(name, " - "); ok {
			if _, err := strconv.Atoi(strings.Trim(prefix, "[] ")); err == nil {
				name = strings.TrimSuffix(rest, "]")
			}
		}
		labels[i] = fmt.Sprintf("[%d] %s", v.ID, name)
		if v.ID == id {
			selected = labels[i]
		}
	}
	ui.Field(c.Key(label), a.T(label), func() {
		if ui.Combobox(c, &selected, labels).Changed() {
			for i, label := range labels {
				if label == selected {
					changed(entries[i].ID)
					break
				}
			}
		}
	})
}
func (a *App) selectIndex(c *ui.Context, label string, at int, labels []string, changed func(int)) {
	translated := make([]string, len(labels))
	for i, v := range labels {
		translated[i] = a.T(v)
	}
	selected := fmt.Sprint(at)
	if at >= 0 && at < len(labels) {
		selected = translated[at]
	}
	ui.Field(c.Key(label), a.T(label), func() {
		if ui.Select(c, &selected, translated).Changed() {
			for i, v := range translated {
				if v == selected {
					changed(i)
					break
				}
			}
		}
	})
}
func (a *App) action(c *ui.Context, label string, op save.Operation) {
	if ui.Button(c.Key(label), a.T(label)).Clicked() {
		a.apply(op)
	}
}
func (a *App) list(c *ui.Context, key string, rows []row, detail func(int)) {
	s := a.table(key)
	ui.Split(c.Key(key), &a.split, func() {
		ui.Column(c).Fill().Gap(8).Children(func() {
			ui.SearchField(c, &s.Query).Label(a.T("Search"))
			filtered := []row{}
			for _, r := range rows {
				if strings.Contains(strings.ToLower(fmt.Sprintf("%d %s %s", r.ID, r.Name, r.Value)), strings.ToLower(s.Query)) {
					filtered = append(filtered, r)
				}
			}
			s.Rows = filtered
			s.List.Selected = &s.Selected
			s.List.Key = func(i int) any { return filtered[i].ID }
			ui.Table(c, &s.List, []ui.TableColumn{{Title: a.T("Name")}, {Title: a.T("Value"), Width: 100}}, len(filtered), func(i, col int) {
				r := filtered[i]
				text := r.Name
				if col == 1 {
					text = r.Value
				}
				ui.Text(c, text).SingleLine()
			}).Grow(1)
		})
	}, func() {
		ui.Scroll(c).Fill().Children(func() {
			selectedID := -1
			if s.Selected >= 0 && s.Selected < len(s.Rows) {
				selectedID = s.Rows[s.Selected].ID
			}
			ui.Column(c.Key(selectedID)).Padding(12).Gap(12).Children(func() {
				if s.Selected >= 0 && s.Selected < len(s.Rows) {
					detail(s.Rows[s.Selected].ID)
				}
			})
		})
	}).Grow(1)
}
func (a *App) mainPage(c *ui.Context) {
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Form(c, func() {
			name, _ := a.Save.Get("playerName")
			text := name.(string)
			ui.Field(c, a.T("Player Name:"), func() {
				if ui.TextInput(c, &text).Changed() {
					a.apply(save.Operation{"op": "setName", "value": text})
				}
			})
			gender, err := a.Save.ProtagonistGender()
			if err == nil && gender != "" {
				at := 0
				if gender == "female" {
					at = 1
				}
				a.selectIndex(c, "Gender", at, []string{"Male", "Female"}, func(i int) {
					a.apply(save.Operation{"op": "setProtagonistGender", "gender": []string{"male", "female"}[i]})
				})
			}
			a.input(c, "Playtime", "Player.Playtime", 0, 3599999)
			a.input(c, "Money", "Player.Money", 0, 9999999)
			a.input(c, "Reputation", "Activities.Reputation", 0, 999999)
			prof := int(a.number("Activities.InstructExp"))
			rank := 0
			for i, v := range catalog.Rules.ProfessorThresholds {
				if prof >= v {
					rank = i
				}
			}
			a.selectIndex(c, "Professor Rank", rank, []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+"}, func(i int) { a.apply(save.Operation{"op": "setProfessorRank", "rank": i}) })
			a.input(c, "Professor Experience", "Activities.InstructExp", 0, 44500)
			history := a.Save.Inheritance(a.DB)
			a.selectIndex(c, "NG+ Professor Rank", int(history["professorRank"].(byte)), []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+"}, func(i int) { a.apply(save.Operation{"op": "setNgPlusProfessorRank", "rank": i}) })
			a.selectIndex(c, "Difficulty", int(a.number("Player.Difficulty")), []string{"Normal", "Hard", "Maddening", "Infernal"}, func(i int) { a.Set("Player.Difficulty", int64(i)) })
			a.selectIndex(c, "Game Style", int(a.number("Player.Gamestyle")), []string{"Casual", "Classic"}, func(i int) { a.Set("Player.Gamestyle", int64(i)) })
			ui.Fieldset(c, a.T("Activity Points"), func() {
				for _, v := range []struct{ label, path string }{{"Explore", "ActivityExplore"}, {"Lesson", "ActivityLesson"}, {"Battle", "ActivityBattle"}} {
					a.input(c, v.label, "Activities."+v.path, 0, 255)
				}
			})
			ui.Fieldset(c, a.T("Goddess Statue"), func() {
				for i := 1; i <= 4; i++ {
					a.input(c, fmt.Sprintf("Statue %d", i), fmt.Sprintf("Activities.Statue%d", i), 0, 255)
				}
			})
			ui.Fieldset(c, a.T("Game Values"), func() {
				for _, f := range []string{"Chapter", "Route", "MapID"} {
					a.input(c, f, "Player."+f, 0, 65535)
				}
			})
			ui.Fieldset(c, a.T("Monthly Statistics"), func() {
				for _, f := range []string{"Wark", "Lecture", "ToBtl", "Rest", "Trnmnt", "Sing", "Lunch", "Cooking", "Drill", "Teaparty", "SCOUT"} {
					a.input(c, f, "Activities.PlayLog_"+f, 0, 255)
				}
			})
		})
	})
}
func (a *App) itemsPage(c *ui.Context) {
	tab := a.tab("items")
	ui.Tabs(c, tab, a.T("Items"), a.T("Miscellaneous"), a.T("Gifts"))
	kind := []string{"items", "misc", "gifts"}[*tab]
	rows := []row{}
	if kind == "items" {
		ui.Row(c).Gap(8).Wrap().Children(func() {
			a.action(c, "Sort", save.Operation{"op": "sortItems"})
			a.action(c, "Add Essential Items", save.Operation{"op": "addEssentialItems"})
			for _, mode := range []string{"normal", "unlimited", "weapons-unlimited"} {
				a.action(c, "Durability: "+mode, save.Operation{"op": "inventoryDurability", "mode": mode})
			}
		})
		for i := 0; i < 400; i++ {
			p := fmt.Sprintf("Items[%d].", i)
			rows = append(rows, row{i, fmt.Sprintf("[%03d] %s", i, a.DB.Name("items", int(a.number(p+"Id")))), fmt.Sprintf("%d / %d", a.number(p+"Durability"), a.number(p+"Amount"))})
		}
	} else {
		ui.Row(c).Gap(8).Children(func() {
			ui.NumberInput(c.Key(kind+"amount"), &a.fillAmount, 0, 255, 1).Label(a.T("Quantity"))
			if ui.Button(c, a.T("Fill All")).Clicked() {
				a.apply(save.Operation{"op": "fillItems", "kind": kind, "amount": int(a.fillAmount)})
			}
		})
		for _, v := range a.DB.Catalog[kind] {
			path := fmt.Sprintf("Player.MiscItems[%d]", v.ID)
			if kind == "gifts" {
				path = fmt.Sprintf("gifts[%d]", v.ID)
			}
			rows = append(rows, row{v.ID, fmt.Sprintf("[%d] %s", v.ID, v.Name), strconv.FormatInt(a.number(path), 10)})
		}
	}
	a.list(c, "items-"+kind, rows, func(i int) {
		ui.Form(c, func() {
			if kind == "items" {
				p := fmt.Sprintf("Items[%d].", i)
				a.choice(c, "Item", a.DB.Catalog["items"], int(a.number(p+"Id")), func(id int) { a.Set(p+"Id", int64(id)) })
				a.input(c, "Durability", p+"Durability", 0, 255)
				a.input(c, "Quantity", p+"Amount", 0, 255)
			} else {
				p := fmt.Sprintf("Player.MiscItems[%d]", i)
				if kind == "gifts" {
					p = fmt.Sprintf("gifts[%d]", i)
				}
				a.input(c, "Quantity", p, 0, 255)
			}
		})
	})
}
func (a *App) questsPage(c *ui.Context) {
	rows := []row{}
	for _, q := range a.DB.Catalog["quests"] {
		rows = append(rows, row{q.ID, q.Name, fmt.Sprint(a.number(fmt.Sprintf("Activities.QuestStateList[%d]", q.ID)))})
	}
	a.list(c, "quests", rows, func(i int) {
		ui.Form(c, func() {
			a.selectIndex(c, "Quest State", int(a.number(fmt.Sprintf("Activities.QuestStateList[%d]", i))), []string{"Unavailable", "Available", "Active", "Unk_0x03", "Finished1", "Unk_0x05", "Finished2"}, func(v int) { a.Set(fmt.Sprintf("Activities.QuestStateList[%d]", i), int64(v)) })
		})
	})
}
