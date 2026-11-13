package app

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
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

func TestUnloadedPageGeometry(t *testing.T) {
	for _, language := range []string{"en_u", "zh_hans"} {
		for _, dark := range []bool{false, true} {
			for _, size := range [][2]int{{1120, 780}, {860, 600}} {
				t.Run(fmt.Sprintf("%s/dark=%v/%dx%d", language, dark, size[0], size[1]), func(t *testing.T) {
					a := New()
					a.dark = dark
					a.setLanguage(language, false)
					tt := ui.NewTesterWith(a.View, size[0], size[1], ui.TesterOptions{Dark: dark})
					for _, view := range layoutViews() {
						t.Run(fmt.Sprintf("%s-%d", view.page, view.tab), func(t *testing.T) {
							a.Page = view.page
							a.revision++
							if view.key != "" {
								*a.tab(view.key) = view.tab
							}
							tt.Frame()
							assertControlsFit(t, tt, size[0])
							if view.page == "Main" {
								var fieldX float32 = -1
								for _, node := range tt.AccessAll() {
									if !visibleControl(node) || node.Bounds.X > float32(size[0])/2 {
										continue
									}
									if fieldX < 0 {
										fieldX = node.Bounds.X
									}
									if math.Abs(float64(node.Bounds.X-fieldX)) > 0.5 {
										t.Errorf("overview fields have different left edges: %s", node.Name)
									}
								}
							}
							if view.page == "Items" {
								list, _ := tt.AccessByRole(ui.RoleGroup, a.T("Item List"))
								if list.Bounds.H < float32(size[1])-100 {
									t.Fatalf("unloaded inventory collapsed: %+v", list.Bounds)
								}
								for _, title := range []string{"Misc Items", "Gift Items"} {
									card, _ := tt.AccessByRole(ui.RoleGroup, a.T(title))
									if card.Bounds.Y != list.Bounds.Y || card.Bounds.H != list.Bounds.H {
										t.Errorf("unloaded inventory cards have different heights: %s", title)
									}
								}
								editor, _ := tt.AccessByRole(ui.RoleGroup, a.T("Item Editor"))
								tools, _ := tt.AccessByRole(ui.RoleGroup, a.T("Item Tools"))
								if editor.Bounds.Y != list.Bounds.Y || tools.Bounds.Y+tools.Bounds.H != list.Bounds.Y+list.Bounds.H || math.Abs(float64(editor.Bounds.H-tools.Bounds.H)) > 0.5 {
									t.Error("unloaded inventory editor column does not fill the page equally")
								}
							}
							for _, area := range tt.AccessAll() {
								if area.Role == ui.RoleScroll && !area.Offscreen && area.Bounds.H > 100 {
									r := area.Bounds
									tt.Scroll(r.X+r.W/2, r.Y+r.H/2, 0, 10000)
								}
							}
							tt.Frame()
							assertControlsFit(t, tt, size[0])
						})
					}
				})
			}
		}
	}
}

func TestProvidedSaveLayouts(t *testing.T) {
	path := os.Getenv("FETH_TEST_SAVE")
	if path == "" {
		t.Skip("set FETH_TEST_SAVE to inspect a local save without writing it")
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	a := New()
	if err := a.Load(path); err != nil {
		t.Fatal(err)
	}
	for _, language := range []string{"en_u", "zh_hans"} {
		a.setLanguage(language, false)
		for _, dark := range []bool{false, true} {
			a.dark = dark
			for _, size := range [][2]int{{1120, 780}, {860, 600}} {
				tt := ui.NewTesterWith(a.View, size[0], size[1], ui.TesterOptions{Dark: dark})
				for _, view := range layoutViews() {
					t.Run(fmt.Sprintf("%s/dark=%v/%dx%d/%s-%d", language, dark, size[0], size[1], view.page, view.tab), func(t *testing.T) {
						a.Page = view.page
						a.revision++
						if view.key != "" {
							*a.tab(view.key) = view.tab
						}
						tt.Frame()
						assertControlsFit(t, tt, size[0])
					})
				}
			}
		}
	}
	if !bytes.Equal(before, a.Save.FinishedBytes()) {
		t.Fatal("viewing the forms changed the loaded save")
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("viewing the forms changed the source file")
	}
}

func TestLoadingEnablesExistingForms(t *testing.T) {
	loaded := fixture(t)
	a := New()
	a.Page = "Items"
	tt := ui.NewTester(a.View, 1120, 780)
	if err := a.Load(loaded.Path); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	apply, ok := tt.AccessByRole(ui.RoleButton, "Apply Changes")
	if !ok || apply.Disabled {
		t.Fatal("loading a save did not enable the existing inventory editor")
	}
	a.Page = "Main"
	tt.Frame()
	money, ok := tt.AccessByRole(ui.RoleTextField, "Money:")
	if !ok || money.Disabled || money.Value != "0" {
		t.Fatal("loading a save did not populate and enable the overview")
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
