package app

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func (a *App) supportPage(c *ui.Context, inherited bool) {
	key, op := "supports", "maxSupports"
	if inherited {
		key, op = "ng-supports", "maxNgPlusSupports"
	}
	a.action(c, "Max All Supports", save.Operation{"op": op})
	section := "supports"
	if inherited {
		section = "inheritance"
	}
	snapshot, err := a.Save.Inspect(a.DB, section)
	if err != nil {
		ui.Text(c, err.Error())
		return
	}
	var values []any
	if inherited {
		values = snapshot["inheritance"].(map[string]any)["supports"].([]any)
	} else {
		values = snapshot["supports"].([]any)
	}
	rows := []row{}
	for i, v := range values {
		m := v.(map[string]any)
		value := m["value"]
		if inherited {
			value = m["maxPoints"]
		}
		rows = append(rows, row{i, a.DB.Name("supports", i), fmt.Sprint(value)})
	}
	a.list(c, key, rows, func(index int) {
		entry, _ := a.DB.Entry("supports", index)
		value := values[index].(map[string]any)["value"]
		if inherited {
			value = values[index].(map[string]any)["maxPoints"]
		}
		points := int(value.(int64))
		rank := "None"
		for _, v := range a.DB.Catalog["support-ranks"] {
			for _, available := range entry.Ranks {
				if v.Rank == available && points >= v.Points {
					rank = v.Rank
				}
			}
		}
		rankOp, maxOp := "setSupportRank", "maxSupportRank"
		if inherited {
			rankOp, maxOp = "setNgPlusSupportRank", "maxNgPlusSupportRank"
		}
		ui.Form(c, func() {
			selected := rank
			ui.Field(c, a.T("Rank"), func() {
				if ui.Select(c, &selected, entry.Ranks).Changed() {
					a.apply(save.Operation{"op": rankOp, "index": index, "rank": selected})
				}
			})
			n := float64(points)
			ui.Field(c, a.T("Points"), func() {
				if ui.NumberInput(c, &n, 0, 65535, 1).Changed() {
					if inherited {
						a.apply(save.Operation{"op": "setNgPlusSupport", "index": index, "points": int(n)})
					} else {
						a.Set(fmt.Sprintf("Player.CharacterSupportValues[%d]", index), int64(n))
					}
				}
			})
		})
		a.action(c, "Max Support Rank", save.Operation{"op": maxOp, "index": index})
	})
}
func (a *App) inheritancePage(c *ui.Context) {
	a.action(c, "Unlock All Playable Characters", save.Operation{"op": "unlockNgPlusRoster"})
	history := a.Save.Inheritance(a.DB)
	characters := history["characters"].([]any)
	rows := []row{}
	for i, v := range characters {
		m := v.(map[string]any)
		rows = append(rows, row{i, m["name"].(string), ""})
	}
	a.list(c, "inheritance", rows, func(index int) {
		m := characters[index].(map[string]any)
		tab := a.tab("inheritance")
		ui.Tabs(c, tab, a.T("Skills"), a.T("Class Mastery"))
		if *tab == 0 {
			ranks := m["skillRanks"].([]int)
			ui.Form(c, func() {
				for i, label := range []string{"Sword", "Lance", "Axe", "Bow", "Brawling", "Reason", "Faith", "Authority", "Heavy Armor", "Riding", "Flying"} {
					a.selectIndex(c.Key(i), label, ranks[i], []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+", "S", "S+"}, func(rank int) {
						a.apply(save.Operation{"op": "setNgPlusSkillRank", "recordIndex": index, "skill": i, "rank": rank})
					})
				}
			})
		} else {
			a.action(c, "Unlock Available Classes", save.Operation{"op": "unlockNgPlusClasses", "recordIndex": index})
			mastered := map[int]bool{}
			for _, id := range m["masteredClassIds"].([]int) {
				mastered[id] = true
			}
			for i := 0; i < 100; i++ {
				on := mastered[i]
				if ui.Checkbox(c.Key(i), &on, fmt.Sprintf("[%d] %s", i, a.DB.Name("classes", i))).Changed() {
					a.apply(save.Operation{"op": "setNgPlusClassMastery", "recordIndex": index, "classId": i, "mastered": on})
				}
			}
		}
	})
}
func (a *App) battalionsPage(c *ui.Context) {
	ui.Row(c).Gap(8).Wrap().Children(func() {
		for _, v := range []struct{ label, op string }{{"Sort", "sortBattalions"}, {"Fill Missing Battalions", "fillMissingBattalions"}, {"Max All Levels", "maxBattalionLevels"}, {"Replenish All", "replenishBattalions"}} {
			a.action(c, v.label, save.Operation{"op": v.op})
		}
	})
	snapshot, err := a.Save.Inspect(a.DB, "battalions")
	if err != nil {
		ui.Text(c, err.Error())
		return
	}
	values := snapshot["battalions"].([]any)
	rows := []row{}
	for i, v := range values {
		m := v.(map[string]any)
		rows = append(rows, row{i, fmt.Sprintf("[%03d] %s", i, m["name"]), fmt.Sprint(m["Stamina"])})
	}
	a.list(c, "battalions", rows, func(slot int) {
		p := fmt.Sprintf("Player.Battalions[%d].", slot)
		ui.Form(c, func() {
			a.choice(c, "Type", a.DB.Catalog["battalions"], int(a.number(p+"Type")), func(id int) { a.Set(p+"Type", int64(id)) })
			a.choice(c, "Assigned Character", a.DB.Catalog["characters"], int(a.number(p+"CharacterId")), func(id int) { a.Set(p+"CharacterId", int64(id)) })
			a.input(c, "Experience", p+"Exp", 0, 65535)
			a.choice(c, "Skill", a.DB.Catalog["battalion-skills"], int(a.number(p+"Skill")), func(id int) { a.Set(p+"Skill", int64(id)) })
			stored := float64(a.number(p + "Stamina"))
			ui.Field(c, a.T("Stored Endurance"), func() {
				if ui.NumberInput(c, &stored, 0, 65535, 1).Changed() {
					a.apply(save.Operation{"op": "setBattalionEnduranceValues", "slot": slot, "storedEndurance": int(stored)})
				}
			})
			var equipped int = -1
			character := a.number(p + "CharacterId")
			typ := a.number(p + "Type")
			if character >= 0 && typ < 200 {
				for i := 0; i < 60; i++ {
					q := fmt.Sprintf("Characters[%d].data.", i)
					if a.number(q+"Id") == character && a.number(q+"Level") > 0 && a.number(q+"EquippedBattalion.Type") == typ {
						equipped = i
						break
					}
				}
			}
			if equipped >= 0 {
				v := float64(a.number(fmt.Sprintf("Characters[%d].data.EquippedBattalion.Stamina", equipped)))
				ui.Field(c, a.T("Equipped Endurance"), func() {
					if ui.NumberInput(c, &v, 0, 65535, 1).Changed() {
						a.apply(save.Operation{"op": "setBattalionEnduranceValues", "slot": slot, "storedEndurance": int(stored), "equippedEndurance": int(v)})
					}
				})
			}
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			a.action(c, "Max Level", save.Operation{"op": "maxBattalionLevel", "slot": slot})
			a.action(c, "Replenish", save.Operation{"op": "replenishBattalion", "slot": slot})
			if ui.Button(c, a.T("Delete Battalion")).Clicked() {
				a.ask("Delete this battalion?", func() { a.apply(save.Operation{"op": "deleteBattalion", "slot": slot}) })
			}
		})
	})
}
func (a *App) databasePage(c *ui.Context) {
	tab := a.tab("database")
	ui.Tabs(c, tab, a.T("Characters"), a.T("Classes"), a.T("Items"))
	kind := []string{"characters", "classes", "items"}[*tab]
	rows := []row{}
	for _, v := range a.DB.Details[kind] {
		rows = append(rows, row{v.ID, fmt.Sprintf("[%d] %s", v.ID, v.Name), ""})
	}
	a.list(c, "database-"+kind, rows, func(id int) {
		detail, err := a.DB.Detail(kind, id)
		if err != nil {
			ui.Text(c, err.Error())
		} else {
			ui.Text(c, strings.ReplaceAll(detail.Details, "\r\n", "\n"))
		}
	})
}
func (a *App) systemPage(c *ui.Context) {
	if ui.Button(c, a.T("Open System Save...")).Clicked() {
		go func() {
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: a.T("Open System Save...")})
			if err != nil {
				a.post(func() { a.Status = err.Error() })
				return
			}
			if len(paths) == 0 {
				return
			}
			a.post(func() {
				load := func() {
					system, err := save.OpenSystem(paths[0])
					if err != nil {
						a.Status = err.Error()
						return
					}
					a.System, a.SystemPath = system, paths[0]
					a.systemBaseline = system.FinishedBytes()
				}
				if a.dirty() {
					a.ask("Discard unsaved changes?", load)
				} else {
					load()
				}
			})
		}()
	}
	if a.System == nil {
		ui.Text(c, a.T("No system save is loaded."))
		return
	}
	snapshot, err := a.System.Inspect(a.DB, "all")
	if err != nil {
		ui.Text(c, err.Error())
		return
	}
	tab := a.tab("system")
	ui.Tabs(c, tab, a.T("Save Slots"), a.T("Flags"))
	if *tab == 0 {
		slots := snapshot["slots"].([]any)
		state := a.table("system-slots")
		ui.Table(c, &state.List, []ui.TableColumn{{Title: a.T("Slot"), Width: 70}, {Title: a.T("Player Name:")}, {Title: a.T("Playtime"), Width: 100}, {Title: a.T("Chapter"), Width: 100}, {Title: a.T("Flags"), Width: 100}}, len(slots), func(i, col int) {
			m := slots[i].(map[string]any)
			var text string
			switch col {
			case 0:
				text = fmt.Sprint(i)
			case 1:
				text = m["playerName"].(string)
			case 2:
				text = fmt.Sprint(m["Playtime"])
			case 3:
				text = fmt.Sprint(m["Chapter1"])
			case 4:
				text = fmt.Sprint(m["Flags"])
			}
			ui.Text(c, text)
		}).Grow(1)
	} else {
		s := a.table("system-flags")
		ui.SearchField(c, &s.Query).Label(a.T("Search"))
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Children(func() {
				for i := 0; i < save.SystemFlags; i++ {
					name := fmt.Sprintf("[%04d] %s", i, a.DB.Name("system-flags", i))
					if !strings.Contains(strings.ToLower(name), strings.ToLower(s.Query)) {
						continue
					}
					on, _ := a.System.Flag(i)
					if ui.Checkbox(c.Key(i), &on, name).Changed() {
						_, err := a.System.Apply([]save.Operation{{"op": "setSystemFlag", "index": i, "value": on}})
						if err != nil {
							a.Status = err.Error()
						}
					}
				}
			})
		})
	}
}
