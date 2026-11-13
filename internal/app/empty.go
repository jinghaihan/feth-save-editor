package app

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// The unloaded editor keeps its form layout. No synthetic save is created.
func (a *App) emptyFields(c *ui.Context, labels ...string) {
	editorForm(c, func() {
		for _, label := range labels {
			value := ""
			a.numberField(c, label, &value)
		}
	})
}
func (a *App) emptyChoice(c *ui.Context, label string) {
	value := ""
	editorField(c, a.T(label), func() { ui.Select(c, &value, nil).Label(a.T(label)) })
}
func (a *App) emptyButtons(c *ui.Context, labels ...string) {
	for _, label := range labels {
		ui.Button(c, a.T(label))
	}
}
func (a *App) emptySplit(c *ui.Context, key, title string, detail func()) {
	ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(editorAppearance.SectionGap).Children(func() {
		a.card(c, a.T(title)).Width(320).MinHeight(0).Children(func() { a.selection(c, "empty-"+key, title, nil, nil) })
		a.card(c, a.T(title+" Editor")).Grow(1).MinHeight(0).Children(detail)
	})
}
func (a *App) emptyPage(c *ui.Context) {
	switch a.Page {
	case "Main":
		ui.Scroll(c).Grow(1).Children(func() {
			a.twoColumns(c, func() {
				a.section(c, "Player", func() {
					a.emptyFields(c, "Player Name:")
					a.emptyChoice(c, "Gender:")
					a.emptyFields(c, "Playtime:", "Money:", "Reputation:")
					a.emptyChoice(c, "Instruct Level:")
					a.emptyFields(c, "Instruct Exp:")
					a.emptyChoice(c, "Inherited rank:")
				})
				a.section(c, "Activity Points", func() { a.emptyFields(c, "Explore:", "Lesson:", "Battle:") })
				a.section(c, "Goddess Statue", func() {
					for i := 1; i <= 4; i++ {
						a.emptyFields(c, a.gameText(9763+i, fmt.Sprintf("Statue%d:", i)))
					}
				})
			}, func() {
				a.section(c, "Game Values", func() {
					a.emptyFields(c, "Chapter:")
					a.emptyChoice(c, "Difficulty:")
					a.emptyChoice(c, "Gamestyle:")
					a.emptyFields(c, "Route:", "Battle MapId:")
				})
				a.section(c, "Monthly Statistics", func() {
					for i, id := range []int{1025, 1026, 1027, 1212, 405, 402, 401, 555, 1860, 3696, 1820} {
						a.emptyFields(c, a.gameText(id, []string{"Explore", "Lesson", "Battle", "Rest", "Tournament", "Choir Practice", "Share a Meal", "Cooking Together", "Advanced Drills", "Tea Party", "Recruit"}[i]))
					}
				})
			})
		})
	case "Items":
		ui.Row(c).Grow(1).Basis(0).MinHeight(0).Gap(10).Children(func() {
			a.card(c, a.T("Item List")).Width(250).MinHeight(0).Children(func() { a.selection(c, "empty-items", "Item List", nil, nil) })
			ui.Column(c).Width(220).Gap(10).Children(func() {
				a.scrollingSection(c, "Item Editor", func() {
					a.emptyChoice(c, "Item")
					a.emptyFields(c, "Durability", "Amount")
					a.emptyButtons(c, "Apply Changes")
				})
				a.scrollingSection(c, "Item Tools", func() {
					a.emptyButtons(c, "Sort Items", "Add Essential Items")
					a.emptyChoice(c, "Set Durability")
					a.emptyButtons(c, "Apply to Inventory")
				})
			})
			for _, title := range []string{"Misc Items", "Gift Items"} {
				a.card(c, a.T(title)).Grow(1).MinHeight(0).Children(func() {
					a.selection(c, "empty-"+title, title, nil, func() { a.emptyFields(c, "Amount"); a.emptyButtons(c, "Set Selected Amount") })
				})
			}
		})
	case "Roster":
		a.emptySplit(c, "roster", "Character List", func() {
			a.emptyChoice(c, "Character Identity:")
			ui.Tabs(c, a.tab("empty-roster"), a.T("Main"), a.T("Skills"), a.T("Flags"), a.T("Class Experience"), a.T("Class Flags"), a.T("Abilities"), a.T("Combat Arts"))
			ui.Scroll(c).Grow(1).Children(func() {
				a.twoColumns(c, func() {
					a.group(c, "Stats", func() {
						a.emptyFields(c, "RNG Value", "Level", "Experience", "HP", "Strength", "Magic", "Dexterity", "Speed", "Luck", "Defense", "Resistance", "Movement", "Charm", "Motivation")
					})
				}, func() {
					a.group(c, "Equipped Abilities", func() {
						for i := 0; i < 5; i++ {
							a.emptyChoice(c, []string{"1", "2", "3", "4", "5"}[i])
						}
					})
					a.group(c, "Equipped Combat Arts", func() { a.emptyChoice(c, "1"); a.emptyChoice(c, "2"); a.emptyChoice(c, "3") })
				})
			})
		})
	case "Database":
		a.databasePage(c)
	case "System":
		ui.Row(c).Grow(1).Gap(editorAppearance.SectionGap).Children(func() {
			a.card(c, a.T("Save Slots")).Width(320).Children(func() { a.selection(c, "empty-slots", "Save Slots", nil, nil) })
			a.card(c, a.T("System Flags")).Grow(1).Children(func() { a.selection(c, "empty-flags", "System Flags", nil, nil) })
		})
	case "Support", "NG+ Support":
		a.emptySplit(c, "support", "Support List", func() {
			ui.Text(c, a.T("Support Editor"))
			a.emptyChoice(c, "Rank")
			a.emptyFields(c, "Points")
			a.emptyButtons(c, "Reach maximum rank", "Set")
		})
	case "Battalions":
		a.emptySplit(c, "battalions", "Battalions", func() {
			a.emptyChoice(c, "Character")
			a.emptyChoice(c, "Battalion")
			a.emptyFields(c, "Experience", "Stored Endurance", "Equipped Endurance")
			a.emptyChoice(c, "Skill")
			a.emptyButtons(c, "Apply Changes", "Set Max Level", "Replenish Endurance", "Delete Battalion")
		})
	case "Quests":
		a.emptySplit(c, "quests", "Quests", func() { a.emptyChoice(c, "State"); a.emptyButtons(c, "Set") })
	case "NG+ Roster":
		a.emptySplit(c, "ng-roster", "Character List", func() {
			ui.Tabs(c, a.tab("empty-inheritance"), a.T("Skills"), a.T("Class Mastery"))
			a.emptyChoice(c, "Skill")
			a.emptyChoice(c, "Rank")
			a.emptyButtons(c, "Unlock All")
		})
	}
}
