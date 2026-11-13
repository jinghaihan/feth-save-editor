package app

import (
	"bytes"
	"github.com/egoist/mygo/ui"
	"os"
	"path/filepath"
	"testing"
)

func TestSystemLanguagePreference(t *testing.T) {
	for _, test := range []struct {
		preferred []string
		want      string
	}{{[]string{"zh-Hans-CN", "en-US"}, "zh_hans"}, {[]string{"zh-TW"}, "zh_hant"}, {[]string{"ja-JP"}, "jp"}, {[]string{"ko-KR"}, "kr"}, {[]string{"en-GB"}, "en_e"}, {[]string{"fr-CA"}, "fr_u"}, {[]string{"es-MX"}, "es_u"}, {[]string{"pt-BR", "de-DE"}, "de"}, {nil, "en_u"}} {
		if got := systemLanguage(test.preferred); got != test.want {
			t.Fatalf("%v => %s, want %s", test.preferred, got, test.want)
		}
	}
}
func TestAutomaticLanguageIsNotSavedAsManualChoice(t *testing.T) {
	a := New()
	a.preferencesPath = filepath.Join(t.TempDir(), "language.txt")
	if err := a.setLanguage("zh_hans", false); err != nil {
		t.Fatal(err)
	}
	if a.languageChosen {
		t.Fatal("automatic language became a manual override")
	}
	if _, err := os.Stat(a.preferencesPath); !os.IsNotExist(err) {
		t.Fatal("automatic language was written to disk")
	}
	if err := a.SetLanguage("en_e"); err != nil {
		t.Fatal(err)
	}
	if !a.languageChosen {
		t.Fatal("manual choice was not remembered")
	}
	if got, _ := os.ReadFile(a.preferencesPath); string(got) != "en_e" {
		t.Fatalf("regional language was not preserved: %q", got)
	}
}
func TestUnloadedEditorKeepsDisabledForms(t *testing.T) {
	for _, dark := range []bool{false, true} {
		a := New()
		a.dark = dark
		tt := ui.NewTester(a.View, 1120, 780)
		for _, page := range pages {
			a.Page = page
			tt.Frame()
			if a.Save != nil || a.System != nil {
				t.Fatal("unloaded view created save data")
			}
			for _, node := range tt.AccessAll() {
				if node.Name == "Open a save to begin." || node.Name == "No system save is loaded." {
					t.Fatalf("%s still replaces its form with a message", page)
				}
				if page != "Database" && (node.Role == ui.RoleTextField || node.Role == ui.RoleComboBox || node.Role == ui.RolePopUpButton || node.Role == ui.RoleButton || node.Role == ui.RoleCheckBox) && !node.Disabled {
					t.Fatalf("%s has an enabled unloaded control: %s", page, node.Name)
				}
			}
			if page == "Main" {
				for _, label := range []string{"Player", "Game Values", "Money:", "Player Name:"} {
					if _, ok := tt.Access(label); !ok {
						t.Fatalf("unloaded main has no %s", label)
					}
				}
			}
		}
	}
}
func TestDatabaseDetailsAreReadOnly(t *testing.T) {
	a := New()
	a.Page = "Database"
	tt := ui.NewTester(a.View, 1120, 780)
	node, ok := tt.AccessByRole(ui.RoleTextField, "Details")
	if !ok {
		t.Fatal("database does not have a detail text area")
	}
	before := node.Value
	tt.ClickAt(node.Bounds.X+10, node.Bounds.Y+10)
	tt.Key(selectAllModifier(), ui.KeyA)
	tt.Type("must not replace database text")
	after, ok := tt.AccessByRole(ui.RoleTextField, "Details")
	if !ok || after.Value != before {
		t.Fatal("database details can be edited")
	}
}
func TestSystemAppearanceRecolorsEveryPage(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1120, 780)
	for _, page := range pages {
		a.Page = page
		a.dark = false
		tt.Frame()
		light := tt.Image()
		a.dark = true
		tt.Frame()
		dark := tt.Image()
		if bytes.Equal(light.Pix, dark.Pix) {
			t.Fatalf("%s does not change with system appearance", page)
		}
	}
}
func TestLoadingFilesDoesNotReplaceStateOnReadFailure(t *testing.T) {
	a := fixture(t)
	original := a.Save
	path := a.Path
	if err := a.Load(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("missing save accepted")
	}
	if a.Save != original || a.Path != path {
		t.Fatal("failed read replaced the loaded save")
	}
	files, err := loadFiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if a.Save != original {
		t.Fatal("reading files changed window state")
	}
	if err = a.installFiles(path, files); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Save.FinishedBytes(), original.FinishedBytes()) {
		t.Fatal("file installation changed save data")
	}
}
func TestSaveMenuFollowsLoadedState(t *testing.T) {
	a := New()
	if !a.menuItems()[1].Submenu[1].Disabled {
		t.Fatal("Save Copy is enabled without a save")
	}
	a = fixture(t)
	if a.menuItems()[1].Submenu[1].Disabled {
		t.Fatal("Save Copy stays disabled after loading")
	}
}
