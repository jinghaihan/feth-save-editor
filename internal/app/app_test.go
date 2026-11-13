package app

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

func fixture(t *testing.T) *App {
	t.Helper()
	b := make([]byte, save.Size)
	binary.LittleEndian.PutUint32(b[4:], 23)
	binary.LittleEndian.PutUint32(b[8:], save.Size)
	for i := 0; i < 400; i++ {
		binary.LittleEndian.PutUint16(b[12+i*4:], 65535)
	}
	for slot := 0; slot < 60; slot++ {
		p := 12 + 0x644 + slot*588
		for i := 0; i < 6; i++ {
			binary.LittleEndian.PutUint16(b[p+i*4:], 65535)
		}
		binary.LittleEndian.PutUint16(b[p+0x24:], 65535)
		b[p+0x18+6] = 200
	}
	p := 12 + 0x644
	binary.LittleEndian.PutUint16(b[p+0x24:], 0)
	b[p+0x4a] = 20
	binary.LittleEndian.PutUint32(b, save.Checksum(b))
	path := filepath.Join(t.TempDir(), "slot00")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	system := make([]byte, 0x1204)
	binary.LittleEndian.PutUint32(system[4:], 7)
	binary.LittleEndian.PutUint32(system[8:], uint32(len(system)))
	binary.LittleEndian.PutUint32(system, save.Checksum(system))
	if err := os.WriteFile(filepath.Join(filepath.Dir(path), "system"), system, 0600); err != nil {
		t.Fatal(err)
	}
	a := New()
	if err := a.Load(path); err != nil {
		t.Fatal(err)
	}
	return a
}
func TestAllPagesAndDefaultControls(t *testing.T) {
	a := fixture(t)
	tester := ui.NewTester(a.View, 1200, 800)
	for _, language := range []string{"en_u", "zh_hans", "jp"} {
		if err := a.SetLanguage(language); err != nil {
			t.Fatal(err)
		}
		for _, page := range pages {
			a.Page = page
			tester.Frame()
			if page != "System" && page != "Main" && page != "Database" {
				if len(a.tables) == 0 {
					t.Fatal("no list")
				}
			}
			if page == "Roster" {
				for tab := 0; tab < 7; tab++ {
					*a.tab("roster") = tab
					tester.Frame()
				}
			}
		}
		tester.SetSize(860, 600)
		for _, page := range pages {
			a.Page = page
			tester.Frame()
		}
		tester.SetSize(1200, 800)
	}
	if dir := os.Getenv("FETH_SCREENSHOT_DIR"); dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		a.SetLanguage("zh_hans")
		for _, page := range []string{"Main", "Items", "Roster", "Battalions", "Support", "NG+ Roster"} {
			a.Page = page
			*a.tab("roster") = 0
			tester.Frame()
			f, err := os.Create(filepath.Join(dir, page+".png"))
			if err != nil {
				t.Fatal(err)
			}
			err = png.Encode(f, tester.Image())
			f.Close()
			if err != nil {
				t.Fatal(err)
			}
		}
	}
}
func TestNumberEntryCommitsOnBlur(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1200, 800)
	if err := tt.Click("Money:"); err != nil {
		t.Fatal(err)
	}
	tt.Key(selectAllModifier(), ui.KeyA)
	tt.Type("64000")
	if got := a.number("Player.Money"); got != 0 {
		t.Fatalf("input committed while typing: %d", got)
	}
	tt.Key(0, ui.KeyTab)
	if got := a.number("Player.Money"); got != 64000 {
		t.Fatalf("Money=%d status=%q", got, a.Status)
	}
}

func TestNavigationAndLegacyColumns(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1200, 800)
	tab, ok := tt.AccessByRole(ui.RoleTab, "Roster")
	if !ok || tab.Bounds.Y > 40 {
		t.Fatalf("navigation is not at the top: %+v", tab)
	}
	left, _ := tt.Access("Player")
	right, _ := tt.Access("Game Values")
	if left.Bounds.X >= right.Bounds.X || left.Bounds.Y != right.Bounds.Y {
		t.Fatal("overview no longer has its original two columns")
	}
	for _, node := range tt.AccessAll() {
		if node.Name == "Increase" || node.Name == "Decrease" {
			t.Fatal("plain overview fields were replaced by steppers")
		}
	}
	if err := tt.Click("Roster"); err != nil {
		t.Fatal(err)
	}
	stats, _ := tt.Access("Stats")
	equipped, _ := tt.Access("Equipped Abilities")
	if equipped.Bounds.X <= stats.Bounds.X || stats.Bounds.Y != equipped.Bounds.Y {
		t.Fatal("equipment was moved out of the character's main page")
	}
	a.Page = "Items"
	tt.Frame()
	previous := float32(-1)
	for _, name := range []string{"Item List", "Item Editor", "Misc Items", "Gift Items"} {
		node, ok := tt.Access(name)
		if !ok || node.Bounds.X <= previous {
			t.Fatalf("inventory columns changed: %s %+v", name, node)
		}
		previous = node.Bounds.X
	}
}

func TestEditingSurvivesSwitchingPage(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1200, 800)
	if err := tt.Click("Money:"); err != nil {
		t.Fatal(err)
	}
	tt.Key(selectAllModifier(), ui.KeyA)
	tt.Type("73100")
	if err := tt.Click("Roster"); err != nil {
		t.Fatal(err)
	}
	if got := a.number("Player.Money"); got != 73100 {
		t.Fatalf("switching pages lost pending input: %d", got)
	}
}

func TestStagedInventoryAndSupportEdits(t *testing.T) {
	a := fixture(t)
	a.Set("Items[0].Id", 1)
	a.Set("Items[0].Amount", 1)
	a.Page = "Items"
	tt := ui.NewTester(a.View, 1200, 800)
	if err := tt.Click("Amount"); err != nil {
		t.Fatal(err)
	}
	tt.Key(selectAllModifier(), ui.KeyA)
	tt.Type("7")
	tt.Key(0, ui.KeyTab)
	if got := a.number("Items[0].Amount"); got != 1 {
		t.Fatalf("staged inventory was applied early: %d", got)
	}
	if err := tt.Click("Apply Changes"); err != nil {
		t.Fatal(err)
	}
	if got := a.number("Items[0].Amount"); got != 7 {
		t.Fatalf("inventory apply failed: %d %s", got, a.Status)
	}
	a.Page = "Support"
	tt.Frame()
	if err := tt.Click("Points"); err != nil {
		t.Fatal(err)
	}
	tt.Key(selectAllModifier(), ui.KeyA)
	tt.Type("250")
	tt.Key(0, ui.KeyTab)
	if got := a.number("Player.CharacterSupportValues[0]"); got != 0 {
		t.Fatal("support was applied before Set")
	}
	if err := tt.Click("Set"); err != nil {
		t.Fatal(err)
	}
	if got := a.number("Player.CharacterSupportValues[0]"); got != 250 {
		t.Fatalf("support apply failed: %d %s", got, a.Status)
	}
}

func TestSystemFlagCopy(t *testing.T) {
	a := fixture(t)
	a.Page = "System"
	a.table("system-flags").Query = "[0008]"
	tt := ui.NewTester(a.View, 1200, 800)
	label := "[0008] " + a.DB.Name("system-flags", 8)
	if err := tt.Click(label); err != nil {
		t.Fatal(err)
	}
	if enabled, _ := a.System.Flag(8); !enabled {
		t.Fatal("system flag did not change")
	}
	dir := t.TempDir()
	if err := a.SaveCopy(dir); err != nil {
		t.Fatal(err)
	}
	system, err := save.OpenSystem(filepath.Join(dir, "system"))
	if err != nil {
		t.Fatal(err)
	}
	if enabled, _ := system.Flag(8); !enabled || !system.ChecksumValid {
		t.Fatal("system copy lost flag or checksum")
	}
	original, err := save.OpenSystem(a.SystemPath)
	if err != nil {
		t.Fatal(err)
	}
	if enabled, _ := original.Flag(8); enabled {
		t.Fatal("source system was changed")
	}
}
func TestSearchRetainsSourceSlotAndSaveCopy(t *testing.T) {
	a := fixture(t)
	a.Page = "Roster"
	state := a.table("roster")
	state.Query = "Byleth"
	tt := ui.NewTester(a.View, 1200, 800)
	if len(state.Rows) != 1 || state.Rows[0].ID != 0 {
		t.Fatalf("search mapped to wrong slot: %+v", state.Rows)
	}
	a.Set("Player.Money", 81000)
	a.Path = filepath.Clean(a.Path)
	before, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err = a.SaveCopy(dir); err != nil {
		t.Fatal(err)
	}
	after, _ := os.ReadFile(a.Path)
	if !bytes.Equal(before, after) {
		t.Fatal("source was modified")
	}
	loaded, err := save.Open(filepath.Join(dir, "slot00"))
	if err != nil {
		t.Fatal(err)
	}
	money, _ := loaded.Number("Player.Money")
	if money != 81000 || !loaded.ChecksumValid {
		t.Fatal("save copy lost edit")
	}
	tt.Frame()
}

func TestFilterPreservesSelectedSource(t *testing.T) {
	a := fixture(t)
	a.Page = "Roster"
	a.Set("Characters[5].data.Id", 10)
	a.Set("Characters[5].data.Level", 20)
	s := a.table("roster")
	s.Selected = 5
	tt := ui.NewTester(a.View, 1200, 800)
	s.Query = "Lv.20"
	tt.Frame()
	if len(s.Rows) != 2 || s.Rows[s.Selected].ID != 5 {
		t.Fatalf("filter changed the selected character: %+v selected=%d", s.Rows, s.Selected)
	}
}

func TestLegacyLanguagePreferences(t *testing.T) {
	a := New()
	a.preferencesPath = filepath.Join(t.TempDir(), "FethEditor", "language.txt")
	t.Setenv("FETH_EDITOR_LANGUAGE", "")
	os.Unsetenv("FETH_EDITOR_LANGUAGE")
	if err := a.SetLanguage("zh_hans"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(a.preferencesPath)
	if err != nil || string(b) != "zh_hans" {
		t.Fatalf("legacy language file: %q %v", b, err)
	}
	t.Setenv("FETH_EDITOR_LANGUAGE", "jp")
	if err := a.SetLanguage("jp"); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(a.preferencesPath)
	if string(b) != "zh_hans" {
		t.Fatal("environment override changed persisted language")
	}
}
func TestConfirmationCancelAndCallbacks(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1200, 800)
	called := 0
	a.ask("Delete this battalion?", func() { called++ })
	tt.Frame()
	tt.Key(0, ui.KeyEscape)
	if a.confirm || called != 0 {
		t.Fatal("Escape did not cancel")
	}
	a.ask("Delete this battalion?", func() { called++ })
	tt.Frame()
	if err := tt.Click("Confirm"); err != nil {
		t.Fatal(err)
	}
	if called != 1 || a.confirm {
		t.Fatal("confirmation did not settle once")
	}
}

func selectAllModifier() ui.Modifiers {
	if runtime.GOOS == "darwin" {
		return ui.Cmd
	}
	return ui.Ctrl
}
