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
		rows := make([]row, 400)
		for i := range rows {
			p := fmt.Sprintf("Items[%d].", i)
			rows[i] = row{i, a.DB.Name("items", int(a.number(p+"Id"))), fmt.Sprintf("%d / %d", a.number(p+"Durability"), a.number(p+"Amount"))}
		}
		selected := -1
		a.card(c, a.T("Item List")).Width(listWidth).MinHeight(0).Children(func() { selected = a.selection(c, "items-items", "Item List", rows, nil) })
		ui.Column(c).Width(editorWidth).MinHeight(0).Children(func() {
			panel := ui.Column(c.Key(selected)).Fill().Gap(10)
			if selected < 0 {
				return
			}
			p := fmt.Sprintf("Items[%d].", selected)
			draft := a.draft(panel, p+"Id", p+"Durability", p+"Amount")
			panel.Children(func() {
				a.scrollingSection(c, "Item Editor", func() {
					ui.Textf(c, "%s %d", a.T("Slot"), selected)
					draft.choice(a, c, "Item", p+"Id", a.DB.Catalog["items"])
					draft.number(a, c, "Durability", p+"Durability")
					draft.number(a, c, "Amount", p+"Amount")
					if ui.Button(c, a.T("Apply Changes")).Clicked() {
						if draft.values[p+"Id"] == "-1" {
							draft.values[p+"Durability"] = "0"
							draft.values[p+"Amount"] = "0"
						}
						if ops, ok := draft.operations(a, p+"Id", p+"Durability", p+"Amount"); ok {
							ops = append(ops, save.Operation{"op": "sortItems"})
							if a.applyBatch(ops...) {
								a.table("items-items").Selected = 0
							}
						}
					}
				})
				a.scrollingSection(c, "Item Tools", func() {
					if ui.Button(c, a.T("Sort Items")).Clicked() {
						a.apply(save.Operation{"op": "sortItems"})
						a.table("items-items").Selected = 0
					}
					a.action(c, "Add Essential Items", save.Operation{"op": "addEssentialItems"})
					mode := a.tab("durability-mode")
					a.selectIndex(c, "Set Durability", *mode, []string{"Normal maximum", "100 (infinite)", "Weapons 100, other maximum"}, func(i int) { *mode = i })
					if ui.Button(c, a.T("Apply to Inventory")).Clicked() {
						a.apply(save.Operation{"op": "inventoryDurability", "mode": []string{"normal", "unlimited", "weapons-unlimited"}[*mode]})
					}
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

func (a *App) smallInventory(c *ui.Context, kind string) {
	rows := []row{}
	for _, entry := range a.DB.Catalog[kind] {
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
	if id < 0 {
		return
	}
	path := fmt.Sprintf("Player.MiscItems[%d]", id)
	if kind == "gifts" {
		path = fmt.Sprintf("gifts[%d]", id)
	}
	draft := a.draft(panel, path)
	panel.Children(func() {
		draft.number(a, c, "Amount", path)
		if ui.Button(c, a.T("Set Selected Amount")).Clicked() {
			if ops, ok := draft.operations(a, path); ok {
				a.applyBatch(ops...)
			}
		}
	})
}

func (a *App) questsPage(c *ui.Context) {
	rows := []row{}
	for _, q := range a.DB.Catalog["quests"] {
		rows = append(rows, row{q.ID, q.Name, fmt.Sprint(a.number(fmt.Sprintf("Activities.QuestStateList[%d]", q.ID)))})
	}
	a.editorLayout(c, "quests", "Quest List", rows, nil, func(id int) {
		panel := ui.Column(c).Width(340).Gap(editorAppearance.ControlGap)
		path := fmt.Sprintf("Activities.QuestStateList[%d]", id)
		draft := a.draft(panel, path)
		panel.Children(func() {
			ui.Text(c, a.T("Quest Editor"))
			at, _ := strconv.Atoi(draft.values[path])
			a.selectIndex(c, "Quest State", at, []string{"Unavailable", "Available", "Active", "Unk_0x03", "Finished1", "Unk_0x05", "Finished2"}, func(i int) { draft.values[path] = strconv.Itoa(i) })
			if ui.Button(c, a.T("Set State")).Clicked() {
				if ops, ok := draft.operations(a, path); ok {
					a.applyBatch(ops...)
				}
			}
		})
	})
}
