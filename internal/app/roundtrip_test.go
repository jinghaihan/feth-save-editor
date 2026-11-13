package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

// Drive the same controls and callbacks as the window, then reopen the file
// they save. Fixture setup is the only part that bypasses the UI.
func sampleControl(t *testing.T, tt *ui.Tester, role ui.Role, name, group string) ui.AccessInfo {
	t.Helper()
	var bounds ui.Rect
	if group != "" {
		g, ok := tt.AccessByRole(ui.RoleGroup, group)
		if !ok {
			t.Fatalf("missing group %q", group)
		}
		bounds = g.Bounds
	}
	for _, n := range tt.AccessAll() {
		if n.Role == role && n.Name == name && !n.Offscreen && (group == "" || containsRect(bounds, n.Bounds)) {
			return n
		}
	}
	t.Fatalf("missing visible control %q in %q", name, group)
	return ui.AccessInfo{}
}
func sampleClick(t *testing.T, tt *ui.Tester, role ui.Role, name, group string) {
	t.Helper()
	n := sampleControl(t, tt, role, name, group)
	tt.ClickAt(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
}
func sampleType(t *testing.T, tt *ui.Tester, name, group, value string) {
	t.Helper()
	sampleClick(t, tt, ui.RoleTextField, name, group)
	tt.Key(selectAllModifier(), ui.KeyA)
	tt.Type(value)
}

func TestGUIEditingRoundTrips(t *testing.T) {
	cases := []struct {
		name, page string
		tab        int
		setup      func(*App)
		edit       func(*testing.T, *App, *ui.Tester)
		want       map[string]int64
		check      func(*testing.T, *App)
	}{
		{name: "money with pending input", page: "Main", edit: func(t *testing.T, a *App, tt *ui.Tester) { sampleType(t, tt, "Money:", "", "81234") }, want: map[string]int64{"Player.Money": 81234}},
		{name: "player name", page: "Main", edit: func(t *testing.T, a *App, tt *ui.Tester) { sampleType(t, tt, "Player Name:", "", "Test Byleth") }, check: func(t *testing.T, a *App) {
			v, _ := a.Save.Get("playerName")
			if v != "Test Byleth" {
				t.Fatalf("name=%v", v)
			}
		}},
		{name: "inventory staged apply", page: "Items", setup: func(a *App) { a.Set("Items[0].Id", 22); a.Set("Items[0].Amount", 1) }, edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Durability", "Item Editor", "17")
			sampleType(t, tt, "Amount", "Item Editor", "4")
			sampleClick(t, tt, ui.RoleButton, "Apply Changes", "Item Editor")
		}, want: map[string]int64{"Items[0].Id": 22, "Items[0].Durability": 17, "Items[0].Amount": 4}},
		{name: "misc quantity", page: "Items", edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Amount", "Misc Items", "57")
			sampleClick(t, tt, ui.RoleButton, "Set Selected Amount", "Misc Items")
		}, want: map[string]int64{"Player.MiscItems[0]": 57}},
		{name: "gift quantity", page: "Items", edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Amount", "Gift Items", "47")
			sampleClick(t, tt, ui.RoleButton, "Set Selected Amount", "Gift Items")
		}, want: map[string]int64{"gifts[0]": 47}},
		{name: "character stat", page: "Roster", edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "HP", "Stats", "52")
			tt.Key(0, ui.KeyTab)
		}, want: map[string]int64{"Characters[0].data.HP": 52}},
		{name: "character flag", page: "Roster", tab: 2, edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleClick(t, tt, ui.RoleCheckBox, "Bit2", "Character Flags")
		}, want: map[string]int64{"Characters[0].data.Flags": 4}},
		{name: "adjutant combobox", page: "Roster", tab: 2, edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleClick(t, tt, ui.RoleComboBox, "Adjutant", "")
			tt.Key(selectAllModifier(), ui.KeyA)
			tt.Type(choiceName(a.DB.Name("characters", 1)))
			tt.Key(0, ui.KeyEnter)
		}, want: map[string]int64{"Characters[0].data.AdjutantId": 1}},
		{name: "class staged apply", page: "Roster", tab: 3, edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Experience", "Class Editor", "7")
			sampleType(t, tt, "Mastery Level (0 or 1)", "Class Editor", "1")
			sampleClick(t, tt, ui.RoleButton, "Set Class Values", "Class Editor")
		}, want: map[string]int64{"Characters[0].data.ClassExp[0]": 7, "Characters[0].data.ClassLevel[0]": 1}},
		{name: "unlock abilities", page: "Roster", tab: 5, edit: func(t *testing.T, a *App, tt *ui.Tester) { sampleClick(t, tt, ui.RoleButton, "Unlock All", "") }, check: func(t *testing.T, a *App) {
			v, _ := a.Save.Record("Characters[0].data.Abilities")
			for _, n := range v.([]any) {
				if n.(int64) != 255 {
					t.Fatal("abilities not unlocked")
				}
			}
		}},
		{name: "unlock combat arts", page: "Roster", tab: 6, edit: func(t *testing.T, a *App, tt *ui.Tester) { sampleClick(t, tt, ui.RoleButton, "Unlock All", "") }, check: func(t *testing.T, a *App) {
			v, _ := a.Save.Record("Characters[0].data.CombatArts")
			for _, n := range v.([]any) {
				if n.(int64) != 255 {
					t.Fatal("arts not unlocked")
				}
			}
		}},
		{name: "battalion staged apply", page: "Battalions", setup: func(a *App) { a.Set("Player.Battalions[0].Type", 20); a.Set("Player.Battalions[0].CharacterId", -1) }, edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Experience", "", "45")
			sampleType(t, tt, "Stored Endurance", "", "17")
			sampleClick(t, tt, ui.RoleButton, "Apply Changes", "")
		}, want: map[string]int64{"Player.Battalions[0].Exp": 45, "Player.Battalions[0].Stamina": 17}},
		{name: "support points", page: "Support", edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Points", "", "250")
			sampleClick(t, tt, ui.RoleButton, "Set", "")
		}, want: map[string]int64{"Player.CharacterSupportValues[0]": 250}},
		{name: "quest select and apply", page: "Quests", edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleClick(t, tt, ui.RolePopUpButton, "Quest State", "")
			if err := tt.Click("Available"); err != nil {
				t.Fatal(err)
			}
			sampleClick(t, tt, ui.RoleButton, "Set State", "")
		}, want: map[string]int64{"Activities.QuestStateList[0]": 1}},
		{name: "inherited skills", page: "NG+ Roster", edit: func(t *testing.T, a *App, tt *ui.Tester) { sampleClick(t, tt, ui.RoleButton, "Unlock All", "") }, check: func(t *testing.T, a *App) {
			m := a.Save.Inheritance(a.DB)["characters"].([]any)[0].(map[string]any)
			for _, rank := range m["skillRanks"].([]int) {
				if rank != 11 {
					t.Fatal("inherited skill not saved")
				}
			}
		}},
		{name: "inherited classes", page: "NG+ Roster", tab: 1, edit: func(t *testing.T, a *App, tt *ui.Tester) { sampleClick(t, tt, ui.RoleButton, "Unlock All", "") }, check: func(t *testing.T, a *App) {
			m := a.Save.Inheritance(a.DB)["characters"].([]any)[0].(map[string]any)
			if len(m["masteredClassIds"].([]int)) == 0 {
				t.Fatal("inherited classes not saved")
			}
		}},
		{name: "inherited support", page: "NG+ Support", edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleType(t, tt, "Points", "", "461")
			sampleClick(t, tt, ui.RoleButton, "Set", "")
		}, check: func(t *testing.T, a *App) {
			m := a.Save.Inheritance(a.DB)["supports"].([]any)[0].(map[string]any)
			if m["maxPoints"].(int64) != 461 {
				t.Fatalf("inherited support=%v", m)
			}
		}},
		{name: "system checkbox", page: "System", setup: func(a *App) { a.table("system-flags").Query = "[0008]" }, edit: func(t *testing.T, a *App, tt *ui.Tester) {
			sampleClick(t, tt, ui.RoleCheckBox, "[0008] "+a.DB.Name("system-flags", 8), "")
		}, check: func(t *testing.T, a *App) {
			on, _ := a.System.Flag(8)
			if !on || !a.System.ChecksumValid {
				t.Fatal("system flag not saved")
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture(t)
			if tc.setup != nil {
				tc.setup(a)
			}
			a.Page = tc.page
			if tc.page == "Roster" {
				*a.tab("roster") = tc.tab
			}
			if tc.page == "NG+ Roster" {
				*a.tab("inheritance") = tc.tab
			}
			source := a.Path
			systemSource := a.SystemPath
			original, err := os.ReadFile(source)
			if err != nil {
				t.Fatal(err)
			}
			originalSystem, err := os.ReadFile(systemSource)
			if err != nil {
				t.Fatal(err)
			}
			tt := ui.NewTester(a.View, 1120, 780)
			tc.edit(t, a, tt)
			out := t.TempDir()
			if err := a.SaveCopy(out); err != nil {
				t.Fatal(err)
			}
			if a.Status != "" {
				t.Fatal(a.Status)
			}
			loaded := New()
			if err := loaded.Load(filepath.Join(out, "slot00")); err != nil {
				t.Fatal(err)
			}
			if !loaded.Save.ChecksumValid {
				t.Fatal("invalid output checksum")
			}
			for path, want := range tc.want {
				if got := loaded.number(path); got != want {
					t.Errorf("%s=%d, want %d", path, got, want)
				}
			}
			if tc.check != nil {
				tc.check(t, loaded)
			}
			after, _ := os.ReadFile(source)
			systemAfter, _ := os.ReadFile(systemSource)
			if !bytes.Equal(original, after) || !bytes.Equal(originalSystem, systemAfter) {
				t.Fatal("original files changed")
			}
			if !bytes.Equal(a.Save.FinishedBytes(), loaded.Save.FinishedBytes()) {
				t.Fatal("reload changed saved bytes")
			}
			t.Logf("%s: UI edit -> save copy -> reload verified", tc.page)
		})
	}
}

func TestMainFormsShareLabelColumn(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1120, 900)
	for _, column := range [][]string{{"Money:", "Explore", "Saint Cethleann Statue"}, {"Chapter:", "Choir Practice"}} {
		var x float32
		for i, name := range column {
			n := sampleControl(t, tt, ui.RoleTextField, name, "")
			if i == 0 {
				x = n.Bounds.X
			} else if n.Bounds.X != x {
				t.Fatal(fmt.Sprintf("%s starts at %v, expected %v", name, n.Bounds.X, x))
			}
		}
	}
}
