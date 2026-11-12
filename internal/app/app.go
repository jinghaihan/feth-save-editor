// Package app is the editor's MyGo interface, using standard themed controls.
package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

var Version = "development"
var pages = []string{"Main", "Items", "Roster", "Battalions", "Quests", "Support", "Database", "NG+ Roster", "NG+ Support", "System"}
var languageCodes = []string{"en_u", "en_e", "zh_hans", "zh_hant", "jp", "kr", "de", "fr_e", "fr_u", "es_e", "es_u", "it"}
var languageNames = []string{"English (US)", "English (EU)", "简体中文", "繁體中文", "日本語", "한국어", "Deutsch", "Français (EU)", "Français (CA)", "Español (EU)", "Español (US)", "Italiano"}

type tableState struct {
	List     ui.ListState
	Query    string
	Selected int
	Rows     []row
}
type row struct {
	ID          int
	Name, Value string
}
type App struct {
	win                    *mygo.Window
	Save                   *save.Buffer
	System                 *save.System
	DB                     *catalog.Database
	Path, SystemPath       string
	systemBaseline         []byte
	Page, Language, Status string
	tables                 map[string]*tableState
	tabs                   map[string]*int
	zh                     map[string]string
	preferencesPath        string
	split                  float32
	revision               int
	confirm                bool
	confirmLabel           string
	confirmAction          func()
	allowClose             bool
	fillAmount             float64
}

func New() *App {
	db, _ := catalog.Load("en_u")
	a := &App{DB: db, Page: "Main", Language: "en_u", tables: map[string]*tableState{}, tabs: map[string]*int{}, split: 310, fillAmount: 99}
	catalog.Decode("ui-zh_hans", &a.zh)
	return a
}
func (a *App) T(text string) string {
	if a.Language == "zh_hans" {
		if value := a.zh[text]; value != "" {
			return value
		}
	}
	if a.Language != "en_u" && a.Language != "en_e" {
		ids := map[string]string{"Items": "1814", "Roster": "1807", "Battalions": "385", "Quests": "1811", "Support": "1060", "Money": "652", "Reputation": "651", "Skills": "569", "Abilities": "260", "Combat Arts": "264", "Level": "2326", "Experience": "2324"}
		if id := ids[text]; id != "" {
			if value := a.DB.Strings[id]; value != "" {
				return value
			}
		}
	}
	return text
}
func (a *App) SetLanguage(code string) error {
	db, err := catalog.Load(code)
	if err != nil {
		return err
	}
	a.Language, a.DB = code, db
	tags := map[string]string{"en_u": "en-US", "en_e": "en-GB", "zh_hans": "zh-Hans", "zh_hant": "zh-Hant", "jp": "ja", "kr": "ko", "de": "de", "fr_e": "fr", "fr_u": "fr-CA", "es_e": "es", "es_u": "es-MX", "it": "it"}
	if a.win != nil {
		mygo.App.SetLocale(tags[code])
		a.menu()
	}
	a.storePreferences()
	return nil
}
func (a *App) table(key string) *tableState {
	if t := a.tables[key]; t != nil {
		return t
	}
	t := &tableState{}
	a.tables[key] = t
	return t
}
func (a *App) tab(key string) *int {
	if t := a.tabs[key]; t != nil {
		return t
	}
	n := 0
	a.tabs[key] = &n
	return &n
}
func (a *App) number(path string) int64 {
	if a.Save == nil {
		return 0
	}
	v, _ := a.Save.Number(path)
	return v
}
func (a *App) apply(op save.Operation) {
	if a.Save == nil {
		return
	}
	if err := a.Save.Apply(a.DB, []save.Operation{op}); err != nil {
		a.Status = err.Error()
	} else {
		a.Status = ""
	}
}
func (a *App) Set(path string, value int64) {
	a.apply(save.Operation{"op": "set", "path": path, "value": value})
}
func (a *App) dirty() bool {
	return a.Save != nil && a.Save.ChangedBytes() > 0 || a.System != nil && !bytes.Equal(a.systemBaseline, a.System.FinishedBytes())
}
func (a *App) Load(path string) error {
	s, err := save.Open(path)
	if err != nil {
		return err
	}
	systemPath := filepath.Join(filepath.Dir(path), "system")
	var system *save.System
	if _, err := os.Stat(systemPath); err == nil {
		system, err = save.OpenSystem(systemPath)
		if err != nil {
			return err
		}
	}
	a.Save, a.Path, a.System, a.SystemPath = s, path, system, systemPath
	a.systemBaseline = nil
	if system != nil {
		a.systemBaseline = system.FinishedBytes()
	}
	a.tables = map[string]*tableState{}
	a.revision++
	a.Status = ""
	if !s.ChecksumValid {
		a.Status = a.T("Warning: save checksum does not match. Saving an edited copy will repair it.")
	}
	return nil
}
func (a *App) post(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	} else {
		fn()
	}
}
func (a *App) open() {
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: a.T("Open Save...")})
		if err != nil {
			a.post(func() { a.Status = err.Error() })
			return
		}
		if len(paths) == 0 {
			return
		}
		a.post(func() {
			load := func() {
				if err := a.Load(paths[0]); err != nil {
					a.Status = err.Error()
				}
			}
			if a.dirty() {
				a.ask("Discard unsaved changes?", load)
			} else {
				load()
			}
		})
	}()
}
func (a *App) saveCopy() {
	if a.Save == nil {
		return
	}
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: a.T("Save Copy..."), Directory: true})
		if err != nil {
			a.post(func() { a.Status = err.Error() })
			return
		}
		if len(paths) == 0 {
			return
		}
		a.post(func() {
			dir := paths[0]
			write := func() {
				if err := a.SaveCopy(dir); err != nil {
					a.Status = err.Error()
				} else {
					a.Status = a.T("Saved")
				}
			}
			_, err := os.Stat(filepath.Join(dir, filepath.Base(a.Path)))
			_, systemErr := os.Stat(filepath.Join(dir, "system"))
			if err == nil || a.System != nil && systemErr == nil {
				a.ask("Replace existing files?", write)
			} else {
				write()
			}
		})
	}()
}
func (a *App) SaveCopy(dir string) error {
	if a.Save == nil {
		return fmt.Errorf("no save is open")
	}
	path := filepath.Join(dir, filepath.Base(a.Path))
	source, _ := filepath.Abs(a.Path)
	dest, _ := filepath.Abs(path)
	if strings.EqualFold(source, dest) {
		return fmt.Errorf("choose another folder to keep the original save")
	}
	b := a.Save.FinishedBytes()
	_, err := save.WriteVerified(path, b, true, trueIfExists(path), func(b []byte) error { _, err := save.Parse(b); return err })
	if err != nil {
		return err
	}
	if a.System != nil && !bytes.Equal(a.systemBaseline, a.System.FinishedBytes()) {
		systemPath := filepath.Join(dir, "system")
		systemSource, _ := filepath.Abs(a.SystemPath)
		systemDest, _ := filepath.Abs(systemPath)
		if strings.EqualFold(systemSource, systemDest) {
			return fmt.Errorf("choose another folder to keep the original system save")
		}
		_, err = save.WriteVerified(systemPath, a.System.FinishedBytes(), true, trueIfExists(systemPath), func(b []byte) error { _, err := save.ParseSystem(b); return err })
		if err != nil {
			return err
		}
	}
	return nil
}
func trueIfExists(path string) bool { _, err := os.Stat(path); return err == nil }
func (a *App) ask(label string, fn func()) {
	a.confirm, a.confirmLabel, a.confirmAction = true, label, fn
}
func (a *App) loadPreferences() {
	dir, err := os.UserConfigDir()
	if err != nil {
		return
	}
	a.preferencesPath = filepath.Join(dir, "feth-save-editor", "preferences.json")
	b, err := os.ReadFile(a.preferencesPath)
	if err != nil {
		return
	}
	var p struct{ Language string }
	if json.Unmarshal(b, &p) == nil && p.Language != "" {
		a.SetLanguage(p.Language)
	}
}
func (a *App) storePreferences() {
	if a.preferencesPath == "" {
		return
	}
	b, _ := json.Marshal(map[string]string{"language": a.Language})
	if err := os.MkdirAll(filepath.Dir(a.preferencesPath), 0700); err == nil {
		if err = os.WriteFile(a.preferencesPath, b, 0600); err != nil {
			a.Status = err.Error()
		}
	}
}
func (a *App) menu() {
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: a.T("File"), Submenu: []*mygo.MenuItem{
			{Label: a.T("Open Save..."), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.open() }},
			{Label: a.T("Save Copy..."), Accelerator: "CmdOrCtrl+Shift+S", Click: func(*mygo.MenuItem, *mygo.Window) { a.saveCopy() }},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: a.T("Help"), Submenu: []*mygo.MenuItem{{Label: a.T("About"), Click: func(*mygo.MenuItem, *mygo.Window) {
			go mygo.Dialog.Message(mygo.MessageOptions{Parent: a.win, Message: "FETH Save Editor " + Version, Detail: "Based on fe3h-editor by imouto1994. Game data and v1.2.0 support by Falo. UI built with MyGo. Icons: Lucide via Iconify.", Buttons: []string{"OK"}})
		}}}},
	}))
}
func Run(path string) error {
	a := New()
	a.loadPreferences()
	if path != "" {
		if err := a.Load(path); err != nil {
			a.Status = err.Error()
		}
	}
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "FETH Save Editor", Width: 1200, Height: 800, MinWidth: 800, MinHeight: 560, StateKey: "main", Content: ui.View(a.View)})
		a.SetLanguage(a.Language)
		a.menu()
	})
	return mygo.App.Run()
}
