package app

import (
	"fmt"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

type layoutView struct {
	page, key string
	tab       int
}

func layoutViews() []layoutView {
	var views []layoutView
	for _, page := range pages {
		key, count := "", 1
		switch page {
		case "Roster":
			key, count = "roster", 7
		case "Database":
			key, count = "database", 3
		case "NG+ Roster":
			key, count = "inheritance", 2
		}
		for tab := 0; tab < count; tab++ {
			views = append(views, layoutView{page, key, tab})
		}
	}
	return views
}

func layoutFixture(t *testing.T) *App {
	a := fixture(t)
	for i, id := range []int{22, 17, 31} {
		if err := a.Save.Apply(a.DB, []save.Operation{{"op": "setCharacterItem", "slot": 0, "itemSlot": i, "id": id}}); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func visibleControl(n ui.AccessInfo) bool {
	if n.Offscreen || n.Bounds.W <= 0 || n.Bounds.H <= 0 {
		return false
	}
	switch n.Role {
	case ui.RoleButton, ui.RoleComboBox, ui.RoleTextField, ui.RolePopUpButton, ui.RoleCheckBox:
		return true
	}
	return false
}

func containsRect(outer, inner ui.Rect) bool {
	return inner.X >= outer.X-0.5 && inner.Y >= outer.Y-0.5 && inner.X+inner.W <= outer.X+outer.W+0.5 && inner.Y+inner.H <= outer.Y+outer.H+0.5
}

func assertControlsFit(t *testing.T, tt *ui.Tester, width int) {
	t.Helper()
	var controls []ui.AccessInfo
	all := tt.AccessAll()
	for _, n := range all {
		if !visibleControl(n) && (n.Role != ui.RoleText || n.Offscreen || n.Bounds.W <= 0 || n.Bounds.H <= 0) {
			continue
		}
		if n.Bounds.X < 0 || n.Bounds.X+n.Bounds.W > float32(width)+0.5 {
			t.Errorf("control outside the window: %q %+v", n.Name, n.Bounds)
		}
		// Accessibility retains the complete row rectangle when only a part
		// of a scrolled row is visible. Compare its painted vertical extent.
		for _, area := range all {
			if area.Role != ui.RoleScroll && area.Role != ui.RoleList {
				continue
			}
			r, s := n.Bounds, area.Bounds
			if r.X >= s.X-0.5 && r.X+r.W <= s.X+s.W+0.5 && r.Y < s.Y+s.H && r.Y+r.H > s.Y {
				top, bottom := max(r.Y, s.Y), min(r.Y+r.H, s.Y+s.H)
				n.Bounds.Y, n.Bounds.H = top, bottom-top
			}
		}
		for _, other := range controls {
			// A combobox's text entry belongs inside its own container.
			if containsRect(other.Bounds, n.Bounds) || containsRect(n.Bounds, other.Bounds) {
				continue
			}
			x := min(other.Bounds.X+other.Bounds.W, n.Bounds.X+n.Bounds.W) - max(other.Bounds.X, n.Bounds.X)
			y := min(other.Bounds.Y+other.Bounds.H, n.Bounds.Y+n.Bounds.H) - max(other.Bounds.Y, n.Bounds.Y)
			if x > 1 && y > 1 && (visibleControl(n) || visibleControl(other)) {
				t.Errorf("overlapping controls: %q %+v / %q %+v", other.Name, other.Bounds, n.Name, n.Bounds)
			}
			if visibleControl(n) && visibleControl(other) && n.Role != ui.RoleCheckBox && other.Role != ui.RoleCheckBox && math.Abs(float64(n.Bounds.X-other.Bounds.X)) < 0.5 && math.Abs(float64(n.Bounds.W-other.Bounds.W)) < 0.5 {
				gap := max(n.Bounds.Y, other.Bounds.Y) - min(n.Bounds.Y+n.Bounds.H, other.Bounds.Y+other.Bounds.H)
				if gap >= 0 && gap < editorAppearance.ControlGap-0.5 {
					t.Errorf("controls are stuck together: %q / %q have %.1fpx gap", other.Name, n.Name, gap)
				}
			}
		}
		controls = append(controls, n)
	}
}

func saveLayoutImage(t *testing.T, tt *ui.Tester, parts ...string) {
	t.Helper()
	dir := os.Getenv("FETH_LAYOUT_SCREENSHOTS")
	if dir == "" {
		return
	}
	path := filepath.Join(append([]string{dir}, parts...)...) + ".png"
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = png.Encode(f, tt.Image())
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestEveryPageLayout(t *testing.T) {
	for _, language := range languageCodes {
		for _, dark := range []bool{false, true} {
			// Exercise both system appearances. The editor itself stays light,
			// matching the C# application's explicit theme preference.
			if dark && language != "en_u" && language != "zh_hans" {
				continue
			}
			for _, size := range [][2]int{{1120, 780}, {860, 600}} {
				theme := "system-light"
				if dark {
					theme = "system-dark"
				}
				t.Run(fmt.Sprintf("%s/%s/%dx%d", language, theme, size[0], size[1]), func(t *testing.T) {
					a := layoutFixture(t)
					a.SetLanguage(language)
					tt := ui.NewTesterWith(a.View, size[0], size[1], ui.TesterOptions{Dark: dark})
					for _, view := range layoutViews() {
						t.Run(fmt.Sprintf("%s-%d", view.page, view.tab), func(t *testing.T) {
							a.Page = view.page
							a.tables = map[string]*tableState{}
							a.tabs = map[string]*int{}
							a.revision++
							if view.key != "" {
								*a.tab(view.key) = view.tab
							}
							tt.Frame()
							assertControlsFit(t, tt, size[0])
							parts := []string{language, theme, fmt.Sprintf("%dx%d", size[0], size[1]), fmt.Sprintf("%s-%d", view.page, view.tab)}
							saveLayoutImage(t, tt, append(parts, "top")...)
							areas := tt.AccessAll()
							for _, area := range areas {
								if (area.Role == ui.RoleScroll || area.Role == ui.RoleList) && area.Bounds.H > 100 && area.Bounds.W > 100 && !area.Offscreen {
									r := area.Bounds
									tt.Scroll(r.X+r.W/2, r.Y+r.H/2, 0, 10000)
								}
							}
							tt.Frame()
							assertControlsFit(t, tt, size[0])
							saveLayoutImage(t, tt, append(parts, "bottom")...)
						})
					}
				})
			}
		}
	}
}

func alignedCenters(t *testing.T, nodes ...ui.AccessInfo) {
	t.Helper()
	want := nodes[0].Bounds.Y + nodes[0].Bounds.H/2
	for _, n := range nodes[1:] {
		if math.Abs(float64(n.Bounds.Y+n.Bounds.H/2-want)) > 0.5 {
			t.Errorf("row is not centered: %q %+v / %q %+v", nodes[0].Name, nodes[0].Bounds, n.Name, n.Bounds)
		}
	}
}

func TestCharacterHeaderAndItemRows(t *testing.T) {
	for _, size := range [][2]int{{1120, 780}, {860, 600}} {
		a := layoutFixture(t)
		a.Page = "Roster"
		tt := ui.NewTester(a.View, size[0], size[1])
		label, _ := tt.AccessByRole(ui.RoleText, "Character Identity:")
		combo, _ := tt.AccessByRole(ui.RoleComboBox, "Character Identity:")
		export, _ := tt.AccessByRole(ui.RoleButton, "Export")
		importButton, _ := tt.AccessByRole(ui.RoleButton, "Import")
		alignedCenters(t, label, combo, export, importButton)
		left, ok := tt.AccessByRole(ui.RoleGroup, "Character List")
		if !ok {
			t.Fatal("left list card is missing")
		}
		right, ok := tt.AccessByRole(ui.RoleGroup, "Character Editor")
		if !ok {
			t.Fatal("right editor card is missing")
		}
		if left.Bounds.Y != right.Bounds.Y || left.Bounds.Y+left.Bounds.H != right.Bounds.Y+right.Bounds.H {
			t.Fatal("outer card edges do not align")
		}
		// Sample the long straight edge, away from rounded corners and text.
		for _, card := range []ui.AccessInfo{left, right} {
			x, y := int(card.Bounds.X), int(card.Bounds.Y+card.Bounds.H/2)
			if tt.Image().RGBAAt(x, y) == tt.Image().RGBAAt(x+3, y) {
				t.Fatalf("card edge has no visible border: %s", card.Name)
			}
		}
		tt.Scroll(right.Bounds.X+right.Bounds.W/2, right.Bounds.Y+right.Bounds.H/2, 0, 10000)
		tt.Frame()
		var lastX, lastW float32
		for i := 1; i <= 6; i++ {
			item, ok := tt.AccessByRole(ui.RoleComboBox, fmt.Sprintf("Item %d", i))
			if !ok || item.Offscreen {
				t.Fatalf("item %d is missing", i)
			}
			durability, ok := tt.AccessByRole(ui.RoleGroup, fmt.Sprintf("Durability %d", i))
			if !ok || durability.Offscreen {
				t.Fatalf("durability %d is missing", i)
			}
			alignedCenters(t, item, durability)
			if i > 1 && (item.Bounds.X != lastX || item.Bounds.W != lastW) {
				t.Fatal("held-item column widths differ between rows")
			}
			lastX, lastW = item.Bounds.X, item.Bounds.W
		}
	}
}

func TestLegacyAppearanceLayers(t *testing.T) {
	for _, dark := range []bool{false, true} {
		a := fixture(t)
		tt := ui.NewTesterWith(a.View, 1120, 780, ui.TesterOptions{Dark: dark})
		if got := tt.Image().RGBAAt(0, 779); got.R != 231 || got.G != 231 || got.B != 231 {
			t.Fatalf("window background changed: %v", got)
		}
		card, ok := tt.AccessByRole(ui.RoleGroup, "Player")
		if !ok {
			t.Fatal("player card missing")
		}
		if got := tt.Image().RGBAAt(int(card.Bounds.X+8), int(card.Bounds.Y+card.Bounds.H/2)); got.R != 242 || got.G != 242 || got.B != 242 {
			t.Fatalf("card background changed: %v", got)
		}
		input, ok := tt.AccessByRole(ui.RoleTextField, "Money:")
		if !ok {
			t.Fatal("money input missing")
		}
		if got := tt.Image().RGBAAt(int(input.Bounds.X+input.Bounds.W-12), int(input.Bounds.Y+8)); got.R != 249 || got.G != 249 || got.B != 249 {
			t.Fatalf("control background changed: %v", got)
		}
	}
}
