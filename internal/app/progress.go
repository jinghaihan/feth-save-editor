package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func (a *App) supportRank(index int, points int64) string {
	entry, _ := a.DB.Entry("supports", index)
	rank := "None"
	for _, v := range a.DB.Catalog["support-ranks"] {
		for _, available := range entry.Ranks {
			if v.Rank == available && points >= int64(v.Points) {
				rank = v.Rank
			}
		}
	}
	return rank
}
func (a *App) supportPage(c *ui.Context, inherited bool) {
	key, section, op := "supports", "supports", "maxSupports"
	if inherited {
		key, section, op = "ng-supports", "inheritance", "maxNgPlusSupports"
	}
	var values []any
	if a.Save != nil {
		snapshot, err := a.Save.Inspect(a.DB, section)
		if err != nil {
			ui.Text(c, err.Error())
			return
		}
		if inherited {
			values = snapshot["inheritance"].(map[string]any)["supports"].([]any)
		} else {
			values = snapshot["supports"].([]any)
		}
	}
	rows := []row{}
	for i, v := range values {
		m := v.(map[string]any)
		value := m["value"]
		if inherited {
			value = m["maxPoints"]
		}
		rows = append(rows, row{i, strings.ReplaceAll(a.DB.Name("supports", i), "🔗", "—"), a.supportRank(i, value.(int64))})
	}
	a.editorLayout(c, key, "Support List", rows, func() { a.action(c, "Reach All Maximum Ranks", save.Operation{"op": op}) }, func(index int) {
		var points int64
		if a.Save != nil {
			m := values[index].(map[string]any)
			if inherited {
				points = m["maxPoints"].(int64)
			} else {
				points = m["value"].(int64)
			}
		}
		panel := ui.Column(c).Width(340).Gap(editorAppearance.ControlGap)

		panel.Children(func() {
			title := "Support Editor"
			if inherited {
				title = "Inherited support rank"
			}
			ui.Text(c, a.T(title))
			entry, _ := a.DB.Entry("supports", index)
			ui.Text(c, a.T("Maximum rank:")+" "+entry.MaximumRank)
			rank := a.supportRank(index, points)
			if a.Save == nil {
				rank = ""
			}
			editorField(c, a.T("Rank"), func() {
				if ui.Select(c, &rank, entry.Ranks).Changed() {
					for _, v := range a.DB.Catalog["support-ranks"] {
						if v.Rank == rank {
							a.flushInput()
							a.setSupportPoints(index, inherited, int64(v.Points))
							break
						}
					}
				}
			})
			a.editNumber(c, "Points", fmt.Sprintf("%s/%d/points", key, index), func() string {
				if a.Save == nil {
					return ""
				}
				if !inherited {
					return a.numberText(fmt.Sprintf("Player.CharacterSupportValues[%d]", index))
				}
				supports := a.Save.Inheritance(a.DB)["supports"].([]any)
				return fmt.Sprint(supports[index].(map[string]any)["maxPoints"])
			}, 0, 65535, func(n int64) { a.setSupportPoints(index, inherited, n) })
			maxOp := "maxSupportRank"
			if inherited {
				maxOp = "maxNgPlusSupportRank"
			}
			if ui.Button(c, a.T("Reach maximum rank")).Disabled(entry.MaximumRank == "None").Clicked() {
				a.flushInput()
				a.apply(save.Operation{"op": maxOp, "index": index})
			}
		})
	})
}
func (a *App) setSupportPoints(index int, inherited bool, n int64) {
	if inherited {
		a.apply(save.Operation{"op": "setNgPlusSupport", "index": index, "points": n})
	} else {
		a.Set(fmt.Sprintf("Player.CharacterSupportValues[%d]", index), n)
	}
}
func (a *App) inheritancePage(c *ui.Context) {
	var characters []any
	if a.Save != nil {
		characters = a.Save.Inheritance(a.DB)["characters"].([]any)
	}
	rows := []row{}
	for i, v := range characters {
		m := v.(map[string]any)
		rows = append(rows, row{i, m["name"].(string), ""})
	}
	a.editorLayout(c, "inheritance", "Character List", rows, func() { a.action(c, "Unlock All Skills & Classes", save.Operation{"op": "unlockNgPlusRoster"}) }, func(index int) {
		var m map[string]any
		if a.Save != nil {
			m = characters[index].(map[string]any)
		}
		tab := a.tab("inheritance")
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Tabs(c, tab, a.T("Skills"), a.T("Class Mastery"))
			ui.Spacer(c)
			if ui.Button(c, a.T("Unlock All")).Clicked() {
				a.flushInput()
				if *tab == 0 {
					ops := make([]save.Operation, 11)
					for i := range ops {
						ops[i] = save.Operation{"op": "setNgPlusSkillRank", "recordIndex": index, "skill": i, "rank": 11}
					}
					a.applyBatch(ops...)
				} else {
					a.apply(save.Operation{"op": "unlockNgPlusClasses", "recordIndex": index})
				}
			}
		})
		if *tab == 0 {
			ranks := make([]int, 11)
			if m != nil {
				ranks = m["skillRanks"].([]int)
			}
			editorForm(c, func() {
				ui.Grid(c).Columns(2).Gap(editorAppearance.SectionGap).Children(func() {
					for i := 0; i < 11; i++ {
						a.selectIndex(c.Key(i), a.gameText(7214+i, fmt.Sprint(i)), ranks[i], skillRanks, func(rank int) {
							a.apply(save.Operation{"op": "setNgPlusSkillRank", "recordIndex": index, "skill": i, "rank": rank})
						})
					}
				})
			})
		} else {
			search := a.table("inheritance-classes")
			ui.SearchField(c, &search.Query).Label(a.T("Search...")).Placeholder(a.T("Search..."))
			mastered := map[int]bool{}
			if m != nil {
				for _, id := range m["masteredClassIds"].([]int) {
					mastered[id] = true
				}
			}
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
				ui.Grid(c).Columns(3).Gap(editorAppearance.ControlGap).Children(func() {
					for i := 0; i < 100; i++ {
						name := a.DB.Name("classes", i)
						if !strings.Contains(strings.ToLower(fmt.Sprintf("%d %s", i, name)), strings.ToLower(search.Query)) {
							continue
						}
						on := mastered[i]
						if ui.Checkbox(c.Key(i), &on, name).Changed() {
							a.apply(save.Operation{"op": "setNgPlusClassMastery", "recordIndex": index, "classId": i, "mastered": on})
						}
					}
				})
			})
		}
	})
}
func (a *App) battalionsPage(c *ui.Context) {
	var values []any
	if a.Save != nil {
		snapshot, err := a.Save.Inspect(a.DB, "battalions")
		if err != nil {
			ui.Text(c, err.Error())
			return
		}
		values = snapshot["battalions"].([]any)
	}
	rows := []row{}
	for i, v := range values {
		m := v.(map[string]any)
		rows = append(rows, row{i, m["name"].(string), fmt.Sprint(m["Stamina"])})
	}
	a.editorLayout(c, "battalions", "Battalion List", rows, func() {
		for _, v := range []struct{ label, op string }{{"Sort Battalion", "sortBattalions"}, {"Fill Missing Battalions", "fillMissingBattalions"}, {"Replenish All", "replenishBattalions"}, {"Max All Levels", "maxBattalionLevels"}} {
			a.action(c, v.label, save.Operation{"op": v.op})
		}
	}, func(slot int) {
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			panel := ui.Column(c).Width(340).Gap(editorAppearance.ControlGap)
			p := fmt.Sprintf("Player.Battalions[%d].", slot)
			equipped := -1
			if a.Save != nil {
				var err error
				equipped, err = a.Save.EquippedBattalionSlot(slot)
				if err != nil {
					ui.Text(c, err.Error())
					return
				}
			}
			currentPath := fmt.Sprintf("Characters[%d].data.EquippedBattalion.Stamina", equipped)

			panel.Children(func() {
				ui.Text(c, a.T("Battalion Editor"))
				a.choice(c, "Character", a.DB.Catalog["characters"], int(a.number(p+"CharacterId")), func(id int) { a.Set(p+"CharacterId", int64(id)) })
				types := append([]catalog.Entry(nil), a.DB.Catalog["battalions"]...)
				for i := range types {
					if types[i].ID == -1 {
						types[i].ID = 200
					}
				}
				a.choice(c, "Type", types, int(a.number(p+"Type")), func(id int) { a.Set(p+"Type", int64(id)) })
				ui.Row(c).Gap(editorAppearance.ControlGap).AlignItems(ui.End).Children(func() {
					ui.Column(c).Grow(1).MinWidth(0).Children(func() { a.input(c, "Experience", p+"Exp", 0, 4294967295) })
					a.action(c, "Max Level", save.Operation{"op": "maxBattalionLevel", "slot": slot})
				})
				a.input(c, "Stored Endurance", p+"Stamina", 0, 4294967295)
				if equipped >= 0 {
					a.inputWith(c, "Equipped Endurance", currentPath, 0, 4294967295, func(n int64) {
						a.apply(save.Operation{"op": "setBattalionEnduranceValues", "slot": slot, "storedEndurance": a.number(p + "Stamina"), "equippedEndurance": n})
					})
				}
				a.action(c, "Replenish", save.Operation{"op": "replenishBattalion", "slot": slot})
				a.choice(c, "Skill", a.DB.Catalog["battalion-skills"], int(a.number(p+"Skill")), func(id int) { a.Set(p+"Skill", int64(id)) })
				if ui.Button(c, a.T("Delete")).Disabled(a.number(p+"Type") >= 200).Clicked() {
					a.flushInput()
					a.ask("Delete this battalion?", func() { a.apply(save.Operation{"op": "deleteBattalion", "slot": slot}) })
				}
			})
		})
	})
}
func (a *App) databasePage(c *ui.Context) {
	a.card(c, a.T("Database")).Grow(1).MinHeight(0).Children(func() {
		tab := a.tab("database")
		ui.Tabs(c, tab, a.T("Characters"), a.T("Classes"), a.T("Items"))
		kind := []string{"characters", "classes", "items"}[*tab]
		rows := []row{}
		for _, v := range a.DB.Details[kind] {
			rows = append(rows, row{v.ID, v.Name, ""})
		}
		ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(editorAppearance.SectionGap).AlignItems(ui.Stretch).Children(func() {
			id := -1
			ui.Column(c).Width(320).MinHeight(0).Children(func() {
				id = a.selection(c, "database-"+kind, []string{"Character Database", "Class Database", "Item Database"}[*tab], rows, nil)
			})
			if id >= 0 {
				detail, err := a.DB.Detail(kind, id)
				if err != nil {
					ui.Text(c, err.Error())
					return
				}
				text := strings.ReplaceAll(detail.Details, "\r\n", "\n")
				ui.TextArea(c.Key(kind), &text).ReadOnly(true).Grow(1).MinWidth(0).MinHeight(0).Label(a.T("Details"))
			}

		})
	})
}
func (a *App) systemPage(c *ui.Context) {
	var slots []any
	if a.System != nil {
		snapshot, err := a.System.Inspect(a.DB, "all")
		if err != nil {
			ui.Text(c, err.Error())
			return
		}
		slots = snapshot["slots"].([]any)
	}
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(editorAppearance.SectionGap).AlignItems(ui.Stretch).Children(func() {
		a.card(c, a.T("Save Slots")).Width(320).MinHeight(0).Children(func() {
			ui.Text(c, a.T("Save Slots"))
			state := a.table("system-slots")
			themedList(c, &state.List, len(slots), func(i int) {
				m := slots[i].(map[string]any)
				ui.Textf(c, "[%02d] %s · %v", i, m["playerName"], m["Playtime"]).Padding(7, 10)
			}).Grow(1).MinHeight(0)
		})
		a.card(c, a.T("System Flags")).Grow(1).MinHeight(0).Children(func() {
			ui.Text(c, a.T("System Flags"))
			state := a.table("system-flags")
			ui.SearchField(c, &state.Query).Label(a.T("Search...")).Placeholder(a.T("Search..."))
			ids := []int{}
			for i := 0; i < save.SystemFlags; i++ {
				name := fmt.Sprintf("[%04d] %s", i, a.DB.Name("system-flags", i))
				if strings.Contains(strings.ToLower(name), strings.ToLower(state.Query)) {
					ids = append(ids, i)
				}
			}
			state.List.Key = func(i int) any { return ids[i] }
			ui.List(c, &state.List, len(ids), func(row int) {
				i := ids[row]
				on := false
				if a.System != nil {
					on, _ = a.System.Flag(i)
				}
				name := fmt.Sprintf("[%04d] %s", i, a.DB.Name("system-flags", i))
				if ui.Checkbox(c.Key(i), &on, name).Changed() {
					_, err := a.System.Apply([]save.Operation{{"op": "setSystemFlag", "index": i, "value": on}})
					if err != nil {
						a.Status = err.Error()
					}
				}
			}).Grow(1).MinHeight(0).Background(cardColor(c)).Gap(editorAppearance.ControlGap)
		})
	})
}
