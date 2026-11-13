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
	c.SetTheme(a.currentTheme())
	c = c.Key(a.revision)
	root := ui.Column(c).Fill().Background(c.Theme().Background)
	root.Children(func() {
		selected := 0
		labels := make([]string, len(pages))
		for i, page := range pages {
			labels[i] = a.T(page)
			if page == a.Page {
				selected = i
			}
		}
		// MyGo draws the tab underline Space(1.5) inside its container.
		ui.ScrollHorizontal(c).PaddingX(16 - c.Theme().Space(1.5)).Children(func() {
			if ui.Tabs(c, &selected, labels...).Changed() {
				a.flushInput()
				a.Page = pages[selected]
			}
		})
		ui.Row(c).Grow(1).Basis(0).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			ui.Column(c.Key(a.Page)).Grow(1).MinWidth(0).MinHeight(0).Padding(16).Gap(editorAppearance.SectionGap).Disabled(a.Save == nil && a.Page != "Database" || a.Page == "System" && a.System == nil).Children(func() {
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
		if a.Status != "" {
			ui.Text(c, a.Status).Padding(8, 12)
		}
	})
	if files := root.DroppedFiles(); len(files) > 0 {
		a.flushInput()
		if err := a.Load(files[0]); err != nil {
			a.Status = err.Error()
		}
	}
	if ui.AlertDialog(c, &a.confirm, a.T(a.confirmLabel), "", a.T("Cancel"), a.T("Confirm")) == 1 && a.confirmAction != nil {
		a.confirmAction()
	}
}
func (a *App) input(c *ui.Context, label, path string, lo, hi int64) {
	a.inputWith(c, label, path, lo, hi, func(value int64) { a.Set(path, value) })
}
func (a *App) inputWith(c *ui.Context, label, path string, lo, hi int64, update func(int64)) {
	if a.Save != nil {
		if minValue, maxValue, err := a.Save.NumberRange(path); err == nil {
			lo = max(lo, minValue)
			hi = min(hi, maxValue)
		}
	}
	a.editNumber(c, label, path, func() string { return a.numberText(path) }, lo, hi, update)
}
func (a *App) editNumber(c *ui.Context, label, key string, value func() string, lo, hi int64, update func(int64)) {
	opened := a.Save
	field := ui.Column(c.Key(key))
	draft := ui.Local(field, "number", func() numberDraft { return numberDraft{Text: value()} })
	commit := func() {
		if opened != nil && a.Save == opened {
			if n, ok := a.parseNumber(draft.Text, label, lo, hi); ok {
				update(n)
			}
			draft.Text = value()
		}
		draft.Focused = false
	}
	field.OnUnmount(func() {
		if draft.Focused {
			commit()
		}
	})
	field.Children(func() {
		input := a.numberField(c, label, &draft.Text)
		focused := input.Focused()
		if draft.Focused && !focused || input.Submitted() {
			commit()
		} else if !focused {
			draft.Text = value()
		}
		draft.Focused = focused
		if focused {
			a.pendingInput = commit
		}
	})
}
func (a *App) choice(c *ui.Context, label string, entries []catalog.Entry, id int, changed func(int)) {
	editorField(c.Key(label), a.T(label), func() {
		a.combo(c, entries, id, changed).Label(a.T(label))
	})
}

func (a *App) combo(c *ui.Context, entries []catalog.Entry, id int, changed func(int)) ui.Element {
	labels := make([]string, len(entries))
	selected := fmt.Sprintf("[%d]", id)
	counts := map[string]int{}
	for _, v := range entries {
		counts[choiceName(v.Name)]++
	}
	for i, v := range entries {
		name := choiceName(v.Name)
		labels[i] = name
		if counts[name] > 1 {
			labels[i] = fmt.Sprintf("%s [%d]", name, v.ID)
		}
		if v.ID == id {
			selected = labels[i]
		}
	}
	if a.Save == nil {
		selected = ""
	}
	input := ui.Combobox(c, &selected, labels)
	if input.Changed() {
		for i, label := range labels {
			if label == selected {
				a.flushInput()
				changed(entries[i].ID)
				break
			}
		}
	}
	return input
}
func choiceName(name string) string {
	if prefix, rest, ok := strings.Cut(name, " - "); ok {
		if _, err := strconv.Atoi(strings.Trim(prefix, "[] ")); err == nil {
			return strings.TrimSuffix(rest, "]")
		}
	}
	return name
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
	if a.Save == nil {
		selected = ""
	}
	editorField(c.Key(label), a.T(label), func() {
		if ui.Select(c, &selected, translated).Changed() {
			for i, v := range translated {
				if v == selected {
					a.flushInput()
					changed(i)
					break
				}
			}
		}
	})
}
func (a *App) action(c *ui.Context, label string, op save.Operation) {
	if ui.Button(c.Key(label), a.T(label)).Clicked() {
		a.flushInput()
		a.apply(op)
	}
}
func (a *App) mainPage(c *ui.Context) {
	ui.Scroll(c).Grow(1).Children(func() {
		editorForm(c, func() {
			a.twoColumns(c, func() {
				a.section(c, "Player", func() {
					ui.Column(c).Gap(editorAppearance.ControlGap).Children(func() {
						name := ""
						if a.Save != nil {
							value, _ := a.Save.Get("playerName")
							name, _ = value.(string)
						}
						box := ui.Column(c.Key("playerName"))
						draft := ui.Local(box, "name", func() numberDraft { return numberDraft{Text: name} })
						opened := a.Save
						commit := func() {
							if opened != nil && a.Save == opened {
								a.apply(save.Operation{"op": "setName", "value": draft.Text})
							}
							draft.Focused = false
						}
						box.OnUnmount(func() {
							if draft.Focused {
								commit()
							}
						})
						box.Children(func() {
							input := a.numberField(c, "Player Name:", &draft.Text)
							focused := input.Focused()
							if draft.Focused && !focused || input.Submitted() {
								commit()
							} else if !focused {
								draft.Text = name
							}
							draft.Focused = focused
							if focused {
								a.pendingInput = commit
							}
						})
						gender := ""
						if a.Save != nil {
							gender, _ = a.Save.ProtagonistGender()
						}
						if a.Save == nil || gender != "" {
							at := 0
							if gender == "female" {
								at = 1
							}
							a.selectIndex(c, "Gender:", at, []string{"Male", "Female"}, func(i int) {
								a.apply(save.Operation{"op": "setProtagonistGender", "gender": []string{"male", "female"}[i]})
							})
						}
						a.input(c, "Playtime:", "Player.Playtime", 0, 4294967295)
						a.input(c, "Money:", "Player.Money", 0, 9999999)
						a.input(c, "Reputation:", "Activities.Reputation", 0, 999999)
						prof := int(a.number("Activities.InstructExp"))
						rank := 0
						for i, v := range catalog.Rules.ProfessorThresholds {
							if prof >= v {
								rank = i
							}
						}
						a.selectIndex(c, "Instruct Level:", rank, []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+"}, func(i int) { a.apply(save.Operation{"op": "setProfessorRank", "rank": i}) })
						a.input(c, "Instruct Exp:", "Activities.InstructExp", 0, 44500)
						inheritedRank := 0
						if a.Save != nil {
							inheritedRank = int(a.Save.Inheritance(a.DB)["professorRank"].(byte))
						}
						a.selectIndex(c, "Inherited rank:", inheritedRank, []string{"E", "E+", "D", "D+", "C", "C+", "B", "B+", "A", "A+"}, func(i int) { a.apply(save.Operation{"op": "setNgPlusProfessorRank", "rank": i}) })
					})
				})
				a.section(c, "Activity Points", func() {
					ui.Column(c).Gap(editorAppearance.ControlGap).Children(func() {
						for i, f := range []string{"ActivityExplore", "ActivityLesson", "ActivityBattle"} {
							a.input(c, a.gameText(1025+i, []string{"Explore:", "Lesson:", "Battle:"}[i]), "Activities."+f, 0, 255)
						}
					})
				})
				a.section(c, "Goddess Statue", func() {
					ui.Column(c).Gap(editorAppearance.ControlGap).Children(func() {
						for i := 1; i <= 4; i++ {
							a.input(c, a.gameText(9763+i, fmt.Sprintf("Statue%d:", i)), fmt.Sprintf("Activities.Statue%d", i), 0, 255)
						}
					})
				})
			}, func() {
				a.section(c, "Game Values", func() {
					ui.Column(c).Gap(editorAppearance.ControlGap).Children(func() {
						a.input(c, "Chapter:", "Player.Chapter", 0, 65535)
						difficulties := make([]string, catalog.Rules.Constants["DIFFICULTY_COUNT"])
						for i := range difficulties {
							difficulties[i] = a.gameText(373+i, fmt.Sprint(i))
						}
						a.selectIndex(c, "Difficulty:", int(a.number("Player.Difficulty")), difficulties, func(i int) { a.Set("Player.Difficulty", int64(i)) })
						styles := make([]string, catalog.Rules.Constants["GAMESTYLE_COUNT"])
						for i := range styles {
							styles[i] = a.gameText(381+i, fmt.Sprint(i))
						}
						a.selectIndex(c, "Gamestyle:", int(a.number("Player.Gamestyle")), styles, func(i int) { a.Set("Player.Gamestyle", int64(i)) })
						a.input(c, "Route:", "Player.Route", 0, 65535)
						a.input(c, "Battle MapId:", "Player.MapID", 0, 65535)
					})
				})
				a.section(c, "Monthly Statistics", func() {
					ui.Column(c).Gap(editorAppearance.ControlGap).Children(func() {
						fields := []struct {
							name string
							id   int
						}{{"Wark", 1025}, {"Lecture", 1026}, {"ToBtl", 1027}, {"Rest", 1212}, {"Trnmnt", 405}, {"Sing", 402}, {"Lunch", 401}, {"Cooking", 555}, {"Drill", 1860}, {"Teaparty", 3696}, {"SCOUT", 1820}}
						for _, f := range fields {
							a.input(c, a.gameText(f.id, "PlayLog_"+f.name+":"), "Activities.PlayLog_"+f.name, 0, 65535)
						}
					})
				})
			})
		})
	})
}
func (a *App) gameText(id int, fallback string) string {
	if text := a.DB.Names["ui"][id]; text != "" {
		return text
	}
	return a.T(fallback)
}
