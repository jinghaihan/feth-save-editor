package app

import (
	"fmt"
	"strconv"
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
		draft := ui.Local(panel, "support", func() struct {
			source int64
			text   string
		} {
			return struct {
				source int64
				text   string
			}{points, fmt.Sprint(points)}
		})
		if draft.source != points {
			draft.source, draft.text = points, fmt.Sprint(points)
		}
		if a.Save == nil {
			draft.text = ""
		}
		panel.Children(func() {
			title := "Support Editor"
			if inherited {
				title = "Inherited support rank"
			}
			ui.Text(c, a.T(title))
			entry, _ := a.DB.Entry("supports", index)
			ui.Text(c, a.T("Maximum rank:")+" "+entry.MaximumRank)
			n, _ := strconv.ParseInt(draft.text, 10, 64)
			rank := a.supportRank(index, n)
			if a.Save == nil {
				rank = ""
			}
			editorField(c, a.T("Rank"), func() {
				if ui.Select(c, &rank, entry.Ranks).Changed() {
					for _, v := range a.DB.Catalog["support-ranks"] {
						if v.Rank == rank {
							draft.text = fmt.Sprint(v.Points)
							break
						}
					}
				}
			})
			a.numberField(c, "Points", &draft.text)
			maxOp := "maxSupportRank"
			if inherited {
				maxOp = "maxNgPlusSupportRank"
			}
			if ui.Button(c, a.T("Reach maximum rank")).Disabled(entry.MaximumRank == "None").Clicked() {
				a.apply(save.Operation{"op": maxOp, "index": index})
			}
			if ui.Button(c, a.T("Set")).Clicked() {
				if n, ok := a.parseNumber(draft.text, "Points", 0, 65535); ok {
					if inherited {
						a.apply(save.Operation{"op": "setNgPlusSupport", "index": index, "points": n})
					} else {
						a.Set(fmt.Sprintf("Player.CharacterSupportValues[%d]", index), n)
					}
				}
			}
		})
	})
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
			paths := []string{p + "CharacterId", p + "Type", p + "Exp", p + "Stamina", p + "Skill"}
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
			if equipped >= 0 {
				paths = append(paths, currentPath)
			}
			draft := a.draft(panel, paths...)
			panel.Children(func() {
				ui.Text(c, a.T("Battalion Editor"))
				draft.choice(a, c, "Character", p+"CharacterId", a.DB.Catalog["characters"])
				types := append([]catalog.Entry(nil), a.DB.Catalog["battalions"]...)
				for i := range types {
					if types[i].ID == -1 {
						types[i].ID = 200
					}
				}
				draft.choice(a, c, "Type", p+"Type", types)
				editorField(c, a.T("Experience"), func() {
					ui.Row(c).Gap(editorAppearance.ControlGap).AlignItems(ui.Center).Children(func() {
						text := draft.values[p+"Exp"]
						ui.TextInput(c, &text).Grow(1).MinWidth(0).Label(a.T("Experience"))
						draft.values[p+"Exp"] = text
						a.action(c, "Max Level", save.Operation{"op": "maxBattalionLevel", "slot": slot})
					})
				})
				draft.number(a, c, "Stored Endurance", p+"Stamina")
				if equipped >= 0 {
					draft.number(a, c, "Equipped Endurance", currentPath)
				}
				a.action(c, "Replenish", save.Operation{"op": "replenishBattalion", "slot": slot})
				draft.choice(a, c, "Skill", p+"Skill", a.DB.Catalog["battalion-skills"])
				if ui.Button(c, a.T("Apply Changes")).Clicked() {
					if ops, ok := draft.operations(a, paths...); ok {
						// Use the existing linked-endurance operation before changing ownership.
						endurance := save.Operation{"op": "setBattalionEnduranceValues", "slot": slot, "storedEndurance": ops[3]["value"]}
						if equipped >= 0 {
							endurance["equippedEndurance"] = ops[len(ops)-1]["value"]
						}
						batch := append([]save.Operation{endurance}, ops[:3]...)
						a.applyBatch(append(batch, ops[4])...)
					}
				}
				if ui.Button(c, a.T("Delete")).Disabled(a.number(p+"Type") >= 200).Clicked() {
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
