package app

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func (a *App) rosterPage(c *ui.Context) {
	rows := []row{}
	for i := 0; i < 60; i++ {
		p := fmt.Sprintf("Characters[%d].data.", i)
		rows = append(rows, row{i, a.DB.Name("characters", int(a.number(p+"Id"))), fmt.Sprintf("Lv.%d", a.number(p+"Level"))})
	}
	a.editorLayout(c, "roster", "Character List", rows, nil, func(slot int) {
		p := fmt.Sprintf("Characters[%d].data.", slot)
		ui.Row(c).Gap(editorAppearance.ControlGap).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.T("Character Identity:")).SingleLine()
			a.combo(c, a.DB.Catalog["characters"], int(a.number(p+"Id")), func(id int) { a.Set(p+"Id", int64(id)) }).Label(a.T("Character Identity:")).Width(180).Shrink(0)
			if ui.Button(c, a.T("Export")).Clicked() {
				a.exportCharacter(slot)
			}
			if ui.Button(c, a.T("Import")).Clicked() {
				a.importCharacter(slot)
			}
		})
		tab := a.tab("roster")
		ui.ScrollHorizontal(c).Children(func() {
			if ui.Tabs(c, tab, a.T("Main"), a.T("Skills"), a.T("Flags"), a.T("Class Experience"), a.T("Class Flags"), a.T("Abilities"), a.T("Combat Arts")).Changed() {
				a.flushInput()
			}
		})
		if *tab == 3 {
			a.characterClasses(c, slot)
			return
		}
		ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
			ui.Column(c).Gap(editorAppearance.SectionGap).Children(func() {
				switch *tab {
				case 0:
					a.columns(c, 332, func() {
						a.group(c, "Stats", func() {
							ui.Text(c, a.T("Class:")+" "+a.DB.Name("classes", int(a.number(p+"Class"))))
							editorForm(c, func() {
								for _, f := range []struct{ label, path string }{{"RNG Value", "RNG_VALUE"}, {"Level", "Level"}, {"Experience", "Exp"}, {"HP", "HP"}, {"Strength", "Strength"}, {"Magic", "Magic"}, {"Dexterity", "Dexterity"}, {"Speed", "Speed"}, {"Luck", "Luck"}, {"Defense", "Defense"}, {"Resistance", "Resistance"}, {"Movement", "Movement"}, {"Charm", "Charm"}, {"Motivation", "Motivation"}} {
									a.input(c, f.label, p+f.path, 0, 4294967295)
								}
							})
							typ := int(a.number(p + "EquippedBattalion.Type"))
							ui.Text(c, a.T("Equipped Battalion:")+" "+a.DB.Name("battalions", typ))
							if typ < 200 {
								ui.Row(c).Gap(editorAppearance.ControlGap).Children(func() {
									ui.Textf(c, "%s: %d", a.T("Endurance"), a.number(p+"EquippedBattalion.Stamina"))
									a.action(c, "Replenish", save.Operation{"op": "replenishCharacterBattalion", "slot": slot})
								})
							}
						})
					}, func() {
						a.group(c, "Equipped Abilities", func() {
							choices := make([]catalog.Entry, 0, 256)
							for _, entry := range a.DB.Catalog["abilities"] {
								if entry.ID >= 0 && entry.ID < 255 {
									entry.Name = a.DB.Name("abilities", entry.ID)
									choices = append(choices, entry)
								}
							}
							choices = append(choices, catalog.Entry{ID: 255, Name: "----"})
							editorForm(c, func() {
								for i := 0; i < 5; i++ {
									path := fmt.Sprintf("%sEquippedAbilities[%d]", p, i)
									a.choice(c, fmt.Sprint(i+1), choices, int(a.number(path)), func(id int) { a.Set(path, int64(id)) })
								}
							})
						})
						a.group(c, "Equipped Combat Arts", func() {
							editorForm(c, func() {
								for i := 0; i < 3; i++ {
									path := fmt.Sprintf("%sEquippedCombatArts[%d]", p, i)
									a.choice(c, fmt.Sprint(i+1), a.DB.Catalog["arts"], int(a.number(path)), func(id int) { a.Set(path, int64(id)) })
								}
							})
						})
						a.group(c, "Item List", func() {
							a.action(c, "Restore Item Durability", save.Operation{"op": "characterItemDurability", "slot": slot})
							for i := 0; i < 6; i++ {
								a.characterItem(c.Key(i), slot, i)
							}
						})
					})
				case 1:
					a.twoColumns(c, func() {
						a.group(c, "Skill Experience", func() {
							editorForm(c, func() {
								for i := 0; i < 11; i++ {
									rank := int(a.number(fmt.Sprintf("%sSkillLevel[%d]", p, i)))
									rankName := fmt.Sprint(rank)
									if rank < len(skillRanks) {
										rankName = skillRanks[rank]
									}
									label := fmt.Sprintf("%s (%s)", a.gameText(7214+i, fmt.Sprint(i)), rankName)
									a.input(c, label, fmt.Sprintf("%sSkillExp[%d]", p, i), 0, 65534)
								}
							})
							a.action(c, "Max Skill Experience", save.Operation{"op": "maxSkillExp", "slot": slot})
						})
					}, func() {
						a.group(c, "Learned Magic", func() {
							for i := 0; i < 12; i++ {
								id := int(a.number(fmt.Sprintf("%sLearnedMagic[%d]", p, i)))
								ui.Textf(c, "%02d · %s · %d", i, a.DB.Name("magic", id), a.number(fmt.Sprintf("%sMagicDurability[%d]", p, i)))
							}
						})
					})
				case 2:
					a.twoColumns(c, func() {
						a.group(c, "Character Flags", func() {
							a.flagRows(c, p+"Flags", 32, func(i int) string { return a.T(a.DB.Name("enmCharacterFlags", i)) })
						})
					}, func() {
						a.group(c, "Class Flags", func() {
							a.flagRows(c, p+"ClassFlags", 8, func(i int) string { return a.T(a.DB.Name("enmCharacterClassFlags", i)) })
						})
						a.group(c, "Adjutant", func() {
							a.combo(c, a.DB.Catalog["characters"], int(a.number(p+"AdjutantId")), func(id int) {
								a.Set(p+"AdjutantId", int64(id))
							}).Label(a.T("Adjutant"))
						})
					})
				case 4:
					a.group(c, "Class Unlock Flags", func() {
						a.flagGrid(c, p+"ClassUnlockFlags", 64, func(i int) string { return a.DB.Name("unit-classes", i) })
					})
				case 5:
					a.groupBox(c, "Abilities").Children(func() {
						ui.Row(c).AlignItems(ui.Center).Children(func() {
							ui.Text(c, a.T("Abilities")).Bold().Grow(1)
							a.action(c, "Unlock All", save.Operation{"op": "unlockAll", "slot": slot, "kind": "abilities"})
						})
						a.flagGrid(c, p+"Abilities", 240, func(i int) string { return a.DB.Name("abilities", i) })
					})
				case 6:
					a.groupBox(c, "Combat Arts").Children(func() {
						ui.Row(c).AlignItems(ui.Center).Children(func() {
							ui.Text(c, a.T("Combat Arts")).Bold().Grow(1)
							a.action(c, "Unlock All", save.Operation{"op": "unlockAll", "slot": slot, "kind": "combat-arts"})
						})
						ui.Column(c).Gap(4).Children(func() {
							a.flagRows(c, p+"CombatArts", 80, func(i int) string { return a.DB.Name("arts", i) })
						})
					})
				}
			})
		})
	})
}

var skillRanks = []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+", "S", "S+"}

func (a *App) characterClasses(c *ui.Context, slot int) {
	p := fmt.Sprintf("Characters[%d].data.", slot)
	key := fmt.Sprintf("character-classes-%d", slot)
	if a.tables[key] == nil {
		a.table(key).Selected = int(a.number(p + "Class"))
	}
	rows := make([]row, 100)
	for i := range rows {
		rows[i] = row{i, a.DB.Name("classes", i), fmt.Sprintf("%d / %d", a.number(fmt.Sprintf("%sClassExp[%d]", p, i)), a.number(fmt.Sprintf("%sClassLevel[%d]", p, i)))}
	}
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(editorAppearance.SectionGap).AlignItems(ui.Stretch).Children(func() {
		id := -1
		a.groupBox(c, "Class Experience").Grow(1).MinHeight(0).Children(func() { id = a.selection(c, key, "Class Experience", rows, nil) })
		panel := ui.Column(c.Key(id)).Width(240).Gap(editorAppearance.ControlGap)
		if id < 0 {
			return
		}
		expPath := fmt.Sprintf("%sClassExp[%d]", p, id)
		levelPath := fmt.Sprintf("%sClassLevel[%d]", p, id)
		draft := a.draft(panel, expPath, levelPath)
		panel.Children(func() {
			a.group(c, "Class Editor", func() {
				ui.Text(c, a.T("Current class:")+" "+a.DB.Name("classes", int(a.number(p+"Class"))))
				draft.number(a, c, "Experience", expPath)
				draft.number(a, c, "Mastery Level (0 or 1)", levelPath)
				if ui.Button(c, a.T("Set Class Values")).Clicked() {
					exp, ok := a.parseNumber(draft.values[expPath], "Experience", 0, int64(catalog.ClassExp(id)))
					level, levelOK := a.parseNumber(draft.values[levelPath], "Mastery", 0, 1)
					if ok && levelOK {
						a.applyBatch(save.Operation{"op": "set", "path": expPath, "value": exp}, save.Operation{"op": "set", "path": levelPath, "value": level})
					}
				}
				a.action(c, "Max All Class Experience", save.Operation{"op": "maxClassExp", "slot": slot})
			})
		})
	})
}
func (a *App) flagGrid(c *ui.Context, path string, count int, name func(int) string) {
	ui.Grid(c).Columns(3).Gap(editorAppearance.ControlGap).Children(func() { a.flagRows(c, path, count, name) })
}
func (a *App) flagRows(c *ui.Context, path string, count int, name func(int) string) {
	value, err := a.Save.Record(path)
	if err != nil {
		ui.Text(c, err.Error())
		return
	}
	for i := 0; i < count; i++ {
		var on bool
		switch v := value.(type) {
		case int64:
			on = v&(1<<uint(i)) != 0
		case []any:
			on = v[i/8].(int64)&(1<<uint(i%8)) != 0
		}
		if ui.Checkbox(c.Key(i), &on, name(i)).Changed() {
			a.apply(save.Operation{"op": "setBit", "path": path, "index": i, "value": on})
		}
	}
}
func (a *App) characterItem(c *ui.Context, slot, itemSlot int) {
	p := fmt.Sprintf("Characters[%d].data.Items[%d].", slot, itemSlot)
	count := int(a.number(fmt.Sprintf("Characters[%d].data.ItemCount", slot)))
	ui.Row(c).Gap(editorAppearance.ControlGap).AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprint(itemSlot+1)).Width(18).SingleLine()
		a.combo(c, a.DB.Catalog["items"], int(a.number(p+"Id")), func(id int) {
			a.apply(save.Operation{"op": "setCharacterItem", "slot": slot, "itemSlot": itemSlot, "id": id})
		}).Label(fmt.Sprintf("%s %d", a.T("Item"), itemSlot+1)).Grow(1).MinWidth(0).Disabled(itemSlot > count)
		value := float64(a.number(p + "Durability"))
		maximum := max(value, float64(catalog.Rules.Durability[int(a.number(p+"Id"))]))
		// Toolbar gives the stock stepper its standard borderless buttons.
		ui.Toolbar(c, func() {
			if ui.NumberInput(c, &value, 0, maximum, 1).Width(104).Label(fmt.Sprintf("%s %d", a.T("Durability"), itemSlot+1)).Changed() {
				a.apply(save.Operation{"op": "setCharacterItem", "slot": slot, "itemSlot": itemSlot, "id": a.number(p + "Id"), "durability": int(value)})
			}
		}).Width(104).Padding(0).Disabled(itemSlot >= count)
	})
}
func (a *App) exportCharacter(slot int) {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, DefaultPath: fmt.Sprintf("character-%03d.character", slot)})
		if err != nil {
			a.post(func() { a.Status = err.Error() })
			return
		}
		if path == "" {
			return
		}
		a.post(func() {
			b, err := a.Save.ExportCharacter(slot)
			if err == nil {
				_, err = save.WriteVerified(path, b, true, trueIfExists(path), func([]byte) error { return nil })
			}
			if err != nil {
				a.Status = err.Error()
			}
		})
	}()
}
func (a *App) importCharacter(slot int) {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win})
		if err != nil {
			a.post(func() { a.Status = err.Error() })
			return
		}
		if len(paths) == 0 {
			return
		}
		a.post(func() {
			a.apply(save.Operation{"op": "importCharacter", "slot": slot, "file": paths[0]})
		})
	}()
}
