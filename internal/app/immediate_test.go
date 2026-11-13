package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func TestOrdinaryEditorsHaveNoConfirmationButton(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1120, 780)
	for _, view := range layoutViews() {
		a.Page = view.page
		if view.key != "" {
			*a.tab(view.key) = view.tab
		}
		a.revision++
		tt.Frame()
		for _, label := range []string{"Apply Changes", "Set Selected Amount", "Set Class Values", "Set State", "Set"} {
			if _, ok := tt.AccessByRole(ui.RoleButton, label); ok {
				t.Errorf("%s: redundant confirmation %s", view.page, label)
			}
		}
	}
}

func TestPendingInputDoesNotOverrideActions(t *testing.T) {
	for _, page := range []string{"Roster", "Battalions", "Support"} {
		t.Run(page, func(t *testing.T) {
			a := fixture(t)
			a.Page = page
			field, group, button, path := "Experience", "Class Editor", "Max All Class Experience", "Characters[0].data.ClassExp[0]"
			switch page {
			case "Roster":
				*a.tab("roster") = 3
				for _, class := range catalog.Rules.Eligible[catalog.Record(int(a.number("Characters[0].data.Id")))] {
					if catalog.ClassExp(class) > 7 {
						a.table("character-classes-0").Selected = class
						path = fmt.Sprintf("Characters[0].data.ClassExp[%d]", class)
						break
					}
				}
			case "Battalions":
				a.Set("Player.Battalions[0].Type", 20)
				a.Set("Player.Battalions[0].CharacterId", -1)
				group, button, path = "", "Max Level", "Player.Battalions[0].Exp"
			case "Support":
				field, group, button, path = "Points", "", "Reach maximum rank", "Player.CharacterSupportValues[0]"
			}
			tt := ui.NewTester(a.View, 1120, 780)
			sampleType(t, tt, field, group, "7")
			sampleClick(t, tt, ui.RoleButton, button, "")
			want := a.number(path)
			if want <= 7 {
				t.Fatalf("%s action did not apply: %d %s", button, want, a.Status)
			}
			if err := a.SaveCopy(t.TempDir()); err != nil {
				t.Fatal(err)
			}
			if got := a.number(path); got != want {
				t.Fatalf("saving restored stale input: %d, want %d", got, want)
			}
		})
	}
}

func TestLinkedBattalionEnduranceEdits(t *testing.T) {
	a := fixture(t)
	source, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	// Give the fixture's active unit the same type as its barracks entry.
	source[12+0x644+0x18+6] = 20
	binary.LittleEndian.PutUint32(source, save.Checksum(source))
	if err := os.WriteFile(a.Path, source, 0600); err != nil {
		t.Fatal(err)
	}
	if err := a.Load(a.Path); err != nil {
		t.Fatal(err)
	}
	a.Set("Player.Battalions[0].Type", 20)
	a.Set("Player.Battalions[0].CharacterId", 0)
	a.Page = "Battalions"
	tt := ui.NewTester(a.View, 1120, 780)
	sampleType(t, tt, "Stored Endurance", "", "17")
	tt.Key(0, ui.KeyTab)
	if got := a.number("Player.Battalions[0].Stamina"); got != 17 {
		t.Fatalf("stored endurance=%d", got)
	}
	sampleType(t, tt, "Equipped Endurance", "", "13")
	tt.Key(0, ui.KeyEnter)
	if got := a.number("Characters[0].data.EquippedBattalion.Stamina"); got != 13 {
		t.Fatalf("equipped endurance=%d: %s", got, a.Status)
	}
	if got := a.number("Player.Battalions[0].Stamina"); got != 17 {
		t.Fatalf("equipped edit changed stored endurance: %d", got)
	}
	out := t.TempDir()
	if err := a.SaveCopy(out); err != nil {
		t.Fatal(err)
	}
	loaded := New()
	if err := loaded.Load(filepath.Join(out, "slot00")); err != nil {
		t.Fatal(err)
	}
	if loaded.number("Player.Battalions[0].Stamina") != 17 || loaded.number("Characters[0].data.EquippedBattalion.Stamina") != 13 {
		t.Fatal("linked endurance values did not round trip")
	}
	original, err := os.ReadFile(a.Path)
	if err != nil || !bytes.Equal(source, original) {
		t.Fatal("editing changed the source file")
	}
}

func TestSupportRankSelectionUpdatesMemory(t *testing.T) {
	for _, inherited := range []bool{false, true} {
		a := fixture(t)
		a.Page = "Support"
		if inherited {
			a.Page = "NG+ Support"
		}
		entry, ok := a.DB.Entry("supports", 0)
		if !ok || len(entry.Ranks) == 0 {
			t.Fatal("missing support fixture")
		}
		rank := entry.Ranks[len(entry.Ranks)-1]
		want := int64(-1)
		for _, r := range a.DB.Catalog["support-ranks"] {
			if r.Rank == rank {
				want = int64(r.Points)
			}
		}
		tt := ui.NewTester(a.View, 1120, 780)
		sampleType(t, tt, "Points", "", "7")
		sampleClick(t, tt, ui.RolePopUpButton, "Rank", "")
		if err := tt.Click(rank); err != nil {
			t.Fatal(err)
		}
		got := a.number("Player.CharacterSupportValues[0]")
		if inherited {
			got = a.Save.Inheritance(a.DB)["supports"].([]any)[0].(map[string]any)["maxPoints"].(int64)
		}
		if got != want {
			t.Fatalf("rank selection did not update points: %d, want %d", got, want)
		}
		if err := a.SaveCopy(t.TempDir()); err != nil {
			t.Fatal(err)
		}
		after := a.number("Player.CharacterSupportValues[0]")
		if inherited {
			after = a.Save.Inheritance(a.DB)["supports"].([]any)[0].(map[string]any)["maxPoints"].(int64)
		}
		if after != want {
			t.Fatalf("saving replaced the selected rank: %d, want %d", after, want)
		}
	}
}

func TestPointerBlurCommitsOrdinaryFields(t *testing.T) {
	for _, tc := range []struct {
		page, group, field, value, next, path string
		tab                                   int
	}{
		{"Items", "Item Editor", "Amount", "7", "Durability", "Items[0].Amount", 0},
		{"Items", "Misc Items", "Amount", "57", "Search...", "Player.MiscItems[0]", 0},
		{"Items", "Gift Items", "Amount", "47", "Search...", "gifts[0]", 0},
		{"Roster", "Class Editor", "Experience", "7", "Mastery Level (0 or 1)", "Characters[0].data.ClassExp[0]", 3},
		{"Battalions", "", "Stored Endurance", "17", "Experience", "Player.Battalions[0].Stamina", 0},
		{"Support", "", "Points", "250", "Rank", "Player.CharacterSupportValues[0]", 0},
		{"NG+ Support", "", "Points", "461", "Rank", "", 0},
	} {
		t.Run(tc.page+"/"+tc.group, func(t *testing.T) {
			a := fixture(t)
			a.Page = tc.page
			if tc.page == "Roster" {
				*a.tab("roster") = tc.tab
			}
			if tc.page == "Items" {
				a.Set("Items[0].Id", 22)
				a.Set("Items[0].Amount", 1)
			}
			source, err := os.ReadFile(a.Path)
			if err != nil {
				t.Fatal(err)
			}
			tt := ui.NewTester(a.View, 1120, 780)
			sampleType(t, tt, tc.field, tc.group, tc.value)
			role := ui.RoleTextField
			if tc.next == "Rank" {
				role = ui.RolePopUpButton
			}
			sampleClick(t, tt, role, tc.next, tc.group)
			got := a.number(tc.path)
			if tc.page == "NG+ Support" {
				got = a.Save.Inheritance(a.DB)["supports"].([]any)[0].(map[string]any)["maxPoints"].(int64)
			}
			var want int64
			fmt.Sscan(tc.value, &want)
			if got != want {
				t.Fatalf("mouse blur did not commit the input: %d, want %d (%s)", got, want, a.Status)
			}
			after, err := os.ReadFile(a.Path)
			if err != nil || !bytes.Equal(source, after) {
				t.Fatal("blur wrote the source file")
			}
		})
	}
}
