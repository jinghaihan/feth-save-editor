package app

import (
	"fmt"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func (a *App) itemsPage(c *ui.Context) {
	width, _ := c.Size()
	scale := min(float32(1), width/1120)
	listWidth, editorWidth := max(float32(190), 250*scale), max(float32(168), 220*scale)
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(10).AlignItems(ui.Stretch).Children(func() {
		var rows []row
		if a.Save != nil {
			rows = make([]row, 400)
		}
		for i := range rows {
			p := fmt.Sprintf("Items[%d].", i)
			rows[i] = row{i, a.DB.Name("items", int(a.number(p+"Id"))), fmt.Sprintf("%d / %d", a.number(p+"Durability"), a.number(p+"Amount"))}
		}
		a.card(c, a.T("Item List")).Width(listWidth + editorWidth + 10).MinHeight(0).Children(func() {
			ui.Row(c).Fill().Gap(editorAppearance.CardPadding).AlignItems(ui.Stretch).Children(func() {
				selected := -1
				ui.Column(c).Width(listWidth - 2*editorAppearance.CardPadding).MinHeight(0).Children(func() {
					selected = a.selection(c, "items-items", "Item List", rows, nil)
				})
				ui.Scroll(c).Grow(1).Basis(0).MinWidth(0).MinHeight(0).Children(func() {
					ui.Column(c).Gap(editorAppearance.CardPadding).Children(func() {
						if a.Save == nil {
							selected = 0
						}
						if selected >= 0 {
							a.itemEditor(c, selected)
						}
						a.itemTools(c)
					})
				})
			})
		})
		for _, kind := range []string{"misc", "gifts"} {
			title := "Misc Items"
			if kind == "gifts" {
				title = "Gift Items"
			}
			a.card(c.Key(kind), a.T(title)).Grow(1).MinHeight(0).Children(func() { a.smallInventory(c, kind) })
		}
	})
}

func (a *App) itemEditor(c *ui.Context, selected int) {
	p := fmt.Sprintf("Items[%d].", selected)
	ui.Column(c.Key(selected)).Gap(editorAppearance.ControlGap).Role(ui.RoleGroup).Label(a.T("Item Editor")).Children(func() {
		ui.Text(c, a.T("Item Editor")).Bold()
		if a.Save != nil {
			ui.Textf(c, "%s %d", a.T("Slot"), selected)
		} else {
			ui.Text(c, a.T("Slot"))
		}
		a.choice(c, "Item", a.DB.Catalog["items"], int(a.number(p+"Id")), func(id int) {
			ops := []save.Operation{{"op": "set", "path": p + "Id", "value": id}}
			if id == -1 {
				ops = append(ops, save.Operation{"op": "set", "path": p + "Durability", "value": 0}, save.Operation{"op": "set", "path": p + "Amount", "value": 0})
			}
			a.applyBatch(ops...)
		})
		a.input(c, "Durability", p+"Durability", 0, 4294967295)
		a.input(c, "Amount", p+"Amount", 0, 4294967295)
	})
}

func (a *App) itemTools(c *ui.Context) {
	ui.Column(c).Gap(editorAppearance.ControlGap).Role(ui.RoleGroup).Label(a.T("Item Tools")).Children(func() {
		ui.Text(c, a.T("Item Tools")).Bold()
		if ui.Button(c, a.T("Sort Items")).Clicked() {
			a.flushInput()
			a.apply(save.Operation{"op": "sortItems"})
			a.table("items-items").Selected = 0
		}
		a.action(c, "Add Essential Items", save.Operation{"op": "addEssentialItems"})
		mode := a.tab("durability-mode")
		a.selectIndex(c, "Set Durability", *mode, []string{"Normal maximum", "100 (infinite)", "Weapons 100, other maximum"}, func(i int) { *mode = i })
		if ui.Button(c, a.T("Apply to Inventory")).Clicked() {
			a.flushInput()
			a.apply(save.Operation{"op": "inventoryDurability", "mode": []string{"normal", "unlimited", "weapons-unlimited"}[*mode]})
		}
	})
}

func (a *App) smallInventory(c *ui.Context, kind string) {
	rows := []row{}
	entries := a.DB.Catalog[kind]
	if a.Save == nil {
		entries = nil
	}
	for _, entry := range entries {
		path := fmt.Sprintf("Player.MiscItems[%d]", entry.ID)
		if kind == "gifts" {
			path = fmt.Sprintf("gifts[%d]", entry.ID)
		}
		rows = append(rows, row{entry.ID, entry.Name, strconv.FormatInt(a.number(path), 10)})
	}
	title := "Misc Items"
	if kind == "gifts" {
		title = "Gift Items"
	}
	ui.Text(c, a.T(title))
	a.action(c, "Get All (99)", save.Operation{"op": "fillItems", "kind": kind, "amount": 99})
	id := -1
	ui.Column(c).Grow(1).MinHeight(0).Children(func() { id = a.selection(c, "items-"+kind, "", rows, nil) })
	panel := ui.Column(c.Key(id)).Gap(editorAppearance.ControlGap)
	if a.Save == nil {
		id = 0
	}
	if id < 0 {
		return
	}
	path := fmt.Sprintf("Player.MiscItems[%d]", id)
	if kind == "gifts" {
		path = fmt.Sprintf("gifts[%d]", id)
	}
	panel.Children(func() { a.input(c, "Amount", path, 0, 4294967295) })
}

func (a *App) questsPage(c *ui.Context) {
	rows := []row{}
	quests := a.DB.Catalog["quests"]
	if a.Save == nil {
		quests = nil
	}
	for _, q := range quests {
		rows = append(rows, row{q.ID, q.Name, fmt.Sprint(a.number(fmt.Sprintf("Activities.QuestStateList[%d]", q.ID)))})
	}
	a.editorLayout(c, "quests", "Quest List", rows, nil, func(id int) {
		panel := ui.Column(c).Width(340).Gap(editorAppearance.ControlGap)
		path := fmt.Sprintf("Activities.QuestStateList[%d]", id)
		panel.Children(func() {
			ui.Text(c, a.T("Quest Editor"))
			a.selectIndex(c, "Quest State", int(a.number(path)), []string{"Unavailable", "Available", "Active", "Unk_0x03", "Finished1", "Unk_0x05", "Finished2"}, func(i int) { a.Set(path, int64(i)) })
		})
	})
}
