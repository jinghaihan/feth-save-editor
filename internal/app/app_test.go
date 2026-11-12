package app

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"os"
	"path/filepath"
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
		tester.SetSize(800, 560)
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
		a.SetLanguage("en_u")
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
func TestNativeNumberInputEditsSave(t *testing.T) {
	a := fixture(t)
	tt := ui.NewTester(a.View, 1200, 800)
	if err := tt.Click("Money"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("64000")
	tt.Key(0, ui.KeyTab)
	if got := a.number("Player.Money"); got != 64000 {
		t.Fatalf("Money=%d status=%q", got, a.Status)
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
