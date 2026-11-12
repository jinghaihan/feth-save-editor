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
		rows = append(rows, row{i, fmt.Sprintf("[%02d] %s", i, a.DB.Name("characters", int(a.number(p+"Id")))), fmt.Sprintf("Lv %d", a.number(p+"Level"))})
	}
	a.list(c, "roster", rows, func(slot int) {
		p := fmt.Sprintf("Characters[%d].data.", slot)
		ui.Form(c, func() {
			a.choice(c, "Character", a.DB.Catalog["characters"], int(a.number(p+"Id")), func(id int) { a.Set(p+"Id", int64(id)) })
		})
		ui.Row(c).Gap(8).Wrap().Children(func() {
			if ui.Button(c, a.T("Export Character...")).Clicked() {
				a.exportCharacter(slot)
			}
			if ui.Button(c, a.T("Import Character...")).Clicked() {
				a.importCharacter(slot)
			}
		})
		tab := a.tab("roster")
		ui.ScrollHorizontal(c).Children(func() {
			ui.Tabs(c, tab, a.T("Main"), a.T("Skills"), a.T("Flags"), a.T("Class Experience"), a.T("Class Flags"), a.T("Abilities"), a.T("Combat Arts"))
		})
		switch *tab {
		case 0:
			ui.Text(c, a.T("Class:")+" "+a.DB.Name("classes", int(a.number(p+"Class"))))
			ui.Form(c, func() {
				for _, field := range []string{"Level", "Exp", "HP", "Strength", "Magic", "Dexterity", "Speed", "Luck", "Defense", "Resistance", "Movement", "Charm", "Motivation", "RNG_VALUE"} {
					hi := int64(255)
					if field == "RNG_VALUE" {
						hi = 4294967295
					}
					if field == "Exp" {
						hi = 65535
					}
					a.input(c, field, p+field, 0, hi)
				}
				a.choice(c, "Adjutant", a.DB.Catalog["characters"], int(a.number(p+"AdjutantId")), func(id int) { a.Set(p+"AdjutantId", int64(id)) })
			})
			ui.Text(c, a.T("Equipped Battalion:")+" "+a.DB.Name("battalions", int(a.number(p+"EquippedBattalion.Type"))))
			ui.Textf(c, "%s: %d", a.T("Endurance"), a.number(p+"EquippedBattalion.Stamina"))
			a.action(c, "Replenish", save.Operation{"op": "replenishCharacterBattalion", "slot": slot})
			ui.Fieldset(c, a.T("Held Items"), func() {
				for i := 0; i < 6; i++ {
					a.characterItem(c.Key(i), slot, i)
				}
			})
		case 1:
			a.action(c, "Max Skill Experience", save.Operation{"op": "maxSkillExp", "slot": slot})
			ui.Form(c, func() {
				for i, label := range []string{"Sword", "Lance", "Axe", "Bow", "Brawling", "Reason", "Faith", "Authority", "Heavy Armor", "Riding", "Flying"} {
					rank := int(a.number(fmt.Sprintf("%sSkillLevel[%d]", p, i)))
					xp := int(a.number(fmt.Sprintf("%sSkillExp[%d]", p, i)))
					skill := i
					a.selectIndex(c, label, rank, []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+", "S", "S+"}, func(r int) {
						a.apply(save.Operation{"op": "setSkillRank", "slot": slot, "skill": skill, "rank": r, "experience": min(xp, catalog.Rules.SkillThresholds[r]-1)})
					})
					a.input(c, label+" "+a.T("Experience"), fmt.Sprintf("%sSkillExp[%d]", p, i), 0, 65534)
				}
				for i := 0; i < 5; i++ {
					path := fmt.Sprintf("%sEquippedAbilities[%d]", p, i)
					a.choice(c, fmt.Sprintf("Ability %d", i+1), a.DB.Catalog["abilities"], int(a.number(path)), func(id int) { a.Set(path, int64(id)) })
				}
				for i := 0; i < 3; i++ {
					path := fmt.Sprintf("%sEquippedCombatArts[%d]", p, i)
					a.choice(c, fmt.Sprintf("Combat Art %d", i+1), a.DB.Catalog["arts"], int(a.number(path)), func(id int) { a.Set(path, int64(id)) })
				}
			})
			ui.Fieldset(c, a.T("Learned Magic"), func() {
				for i := 0; i < 12; i++ {
					id := int(a.number(fmt.Sprintf("%sLearnedMagic[%d]", p, i)))
					durability := a.number(fmt.Sprintf("%sMagicDurability[%d]", p, i))
					ui.Textf(c, "%02d · %s · %d", i, a.DB.Name("magic", id), durability)
				}
			})
		case 2:
			a.flagRows(c, p+"Flags", 32, func(i int) string {
				switch i {
				case 0:
					return a.T("Is Available")
				case 1:
					return a.T("Has Joined")
				case 3:
					return a.T("Is Dead")
				case 18:
					return a.T("Is Deployed")
				case 19:
					return a.T("Must take part in Battle")
				case 20:
					return a.T("Is Adjutant")
				}
				return fmt.Sprintf("Bit %d", i)
			})
		case 3:
			a.action(c, "Max Class Experience", save.Operation{"op": "maxClassExp", "slot": slot})
			ui.Form(c, func() {
				for i := 0; i < 100; i++ {
					ui.Fieldset(c.Key(i), a.DB.Name("classes", i), func() {
						a.input(c, "Experience", fmt.Sprintf("%sClassExp[%d]", p, i), 0, int64(catalog.ClassExp(i)))
						a.input(c, "Mastery", fmt.Sprintf("%sClassLevel[%d]", p, i), 0, 1)
					})
				}
			})
		case 4:
			ui.Fieldset(c, a.T("Class Flags"), func() { a.flagRows(c, p+"ClassFlags", 8, func(i int) string { return fmt.Sprintf("Bit %d", i) }) })
			ui.Fieldset(c, a.T("Unlocked Classes"), func() { a.flagRows(c, p+"ClassUnlockFlags", 64, func(i int) string { return a.DB.Name("classes", i) }) })
		case 5:
			a.action(c, "Unlock All Abilities", save.Operation{"op": "unlockAll", "slot": slot, "kind": "abilities"})
			a.flagRows(c, p+"Abilities", 240, func(i int) string { return a.DB.Name("abilities", i) })
		case 6:
			a.action(c, "Unlock All Combat Arts", save.Operation{"op": "unlockAll", "slot": slot, "kind": "combat-arts"})
			a.flagRows(c, p+"CombatArts", 80, func(i int) string { return a.DB.Name("arts", i) })
		}
	})
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
		if ui.Checkbox(c.Key(i), &on, fmt.Sprintf("[%d] %s", i, name(i))).Changed() {
			a.apply(save.Operation{"op": "setBit", "path": path, "index": i, "value": on})
		}
	}
}
func (a *App) characterItem(c *ui.Context, slot, itemSlot int) {
	p := fmt.Sprintf("Characters[%d].data.Items[%d].", slot, itemSlot)
	id := int(a.number(p + "Id"))
	ui.Form(c, func() {
		a.choice(c, fmt.Sprintf("Item %d", itemSlot+1), a.DB.Catalog["items"], id, func(newID int) {
			if newID == -1 {
				return
			}
			a.apply(save.Operation{"op": "setCharacterItem", "slot": slot, "itemSlot": itemSlot, "id": newID})
		})
		a.input(c, "Durability", p+"Durability", 0, 255)
		a.input(c, "Quantity", p+"Amount", 0, 255)
	})
}
func (a *App) exportCharacter(slot int) {
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, DefaultPath: fmt.Sprintf("character-%d.bin", slot)})
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
			a.ask("Replace this character?", func() { a.apply(save.Operation{"op": "importCharacter", "slot": slot, "file": paths[0]}) })
		})
	}()
}
