// Package app is the editor's MyGo interface, using standard themed controls.
package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/assets"
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
	languageChosen         bool
	dark                   bool
	revision               int
	confirm                bool
	confirmLabel           string
	confirmAction          func()
	pendingInput           func()
}

func New() *App {
	db, _ := catalog.Load("en_u")
	a := &App{DB: db, Page: "Main", Language: "en_u", tables: map[string]*tableState{}, tabs: map[string]*int{}}
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

var localeTags = map[string]string{"en_u": "en-US", "en_e": "en-GB", "zh_hans": "zh-Hans", "zh_hant": "zh-Hant", "jp": "ja", "kr": "ko", "de": "de", "fr_e": "fr", "fr_u": "fr-CA", "es_e": "es", "es_u": "es-MX", "it": "it"}

func (a *App) SetLanguage(code string) error { return a.setLanguage(code, true) }
func (a *App) setLanguage(code string, remember bool) error {
	db, err := catalog.Load(code)
	if err != nil {
		return err
	}
	a.Language, a.DB = code, db
	if remember {
		a.languageChosen = true
		a.storePreferences()
	}
	if a.win != nil {
		a.refreshLocale()
	}
	return nil
}
func (a *App) refreshLocale() {
	mygo.App.SetLocale(localeTags[a.Language])
	a.menu()
	a.win.Invalidate()
}
func systemLanguage(preferred []string) string {
	tags := make([]string, len(languageCodes))
	for i, code := range languageCodes {
		tags[i] = localeTags[code]
	}
	matched := i18n.Match(preferred, tags)
	for code, tag := range localeTags {
		if strings.EqualFold(tag, matched) {
			return code
		}
	}
	return "en_u"
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

type loadedFiles struct {
	save       *save.Buffer
	system     *save.System
	systemPath string
}

func loadFiles(path string) (loadedFiles, error) {
	s, err := save.Open(path)
	if err != nil {
		return loadedFiles{}, err
	}
	files := loadedFiles{save: s, systemPath: filepath.Join(filepath.Dir(path), "system")}
	if _, err := os.Stat(files.systemPath); err == nil {
		files.system, err = save.OpenSystem(files.systemPath)
		if err != nil {
			return loadedFiles{}, err
		}
	}
	return files, nil
}
func (a *App) Load(path string) error {
	files, err := loadFiles(path)
	if err != nil {
		return err
	}
	return a.installFiles(path, files)
}
func (a *App) installFiles(path string, files loadedFiles) error {
	if a.System != nil && !bytes.Equal(a.systemBaseline, a.System.FinishedBytes()) && !strings.EqualFold(a.SystemPath, files.systemPath) {
		return fmt.Errorf("save the edited system copy before loading another system save")
	}
	a.Save, a.Path, a.System, a.SystemPath = files.save, path, files.system, files.systemPath
	a.pendingInput = nil
	a.systemBaseline = nil
	if files.system != nil {
		a.systemBaseline = files.system.FinishedBytes()
	}
	a.tables = map[string]*tableState{}
	a.revision++
	a.Status = ""
	if !files.save.ChecksumValid {
		a.Status = a.T("Warning: save checksum does not match. Saving an edited copy will repair it.")
	}
	if a.win != nil {
		a.menu()
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
	a.flushInput()
	title := a.T("Open Save...")
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: title})
		if err != nil {
			a.post(func() { a.Status = err.Error() })
			return
		}
		if len(paths) == 0 {
			return
		}
		files, err := loadFiles(paths[0])
		a.post(func() {
			if err != nil {
				a.Status = err.Error()
				return
			}
			if err := a.installFiles(paths[0], files); err != nil {
				a.Status = err.Error()
			}
		})
	}()
}
func (a *App) saveCopy() {
	if a.Save == nil {
		return
	}
	a.flushInput()
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
			write()
		})
	}()
}
func (a *App) SaveCopy(dir string) error {
	a.flushInput()
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
func (a *App) flushInput() {
	if pending := a.pendingInput; pending != nil {
		a.pendingInput = nil
		pending()
	}
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
	a.preferencesPath = filepath.Join(dir, "FethEditor", "language.txt")
	if forced := strings.TrimSpace(os.Getenv("FETH_EDITOR_LANGUAGE")); forced != "" {
		if a.setLanguage(canonicalLanguage(forced), false) == nil {
			a.languageChosen = true
			return
		}
	}
	if b, err := os.ReadFile(a.preferencesPath); err == nil {
		if a.setLanguage(canonicalLanguage(strings.TrimSpace(string(b))), false) == nil {
			a.languageChosen = true
		}
	}
}
func (a *App) storePreferences() {
	_, forced := os.LookupEnv("FETH_EDITOR_LANGUAGE")
	if a.preferencesPath == "" || forced {
		return
	}
	if err := os.MkdirAll(filepath.Dir(a.preferencesPath), 0700); err == nil {
		if err = os.WriteFile(a.preferencesPath, []byte(a.Language), 0600); err != nil {
			a.Status = err.Error()
		}
	}
}
func canonicalLanguage(code string) string { return strings.ToLower(strings.TrimSpace(code)) }
func (a *App) menu()                       { mygo.App.SetMenu(mygo.NewMenu(a.menuItems())) }
func (a *App) menuItems() []*mygo.MenuItem {
	return []*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: a.T("File"), Submenu: []*mygo.MenuItem{
			{Label: a.T("Open Save..."), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.open() }},
			{Label: a.T("Save Copy..."), Disabled: a.Save == nil, Accelerator: "CmdOrCtrl+Shift+S", Click: func(*mygo.MenuItem, *mygo.Window) { a.saveCopy() }},
			{Label: a.T("Exit"), Accelerator: "CmdOrCtrl+Q", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
		}},
		{Label: a.T("Language"), Submenu: a.languageMenu()},
		{Role: mygo.RoleEditMenu},
		{Label: a.T("Help"), Submenu: []*mygo.MenuItem{{Label: a.T("About"), Click: func(*mygo.MenuItem, *mygo.Window) {
			go mygo.Dialog.Message(mygo.MessageOptions{Parent: a.win, Message: "FETH Save Editor " + Version, Detail: "Based on fe3h-editor by imouto1994. Game data and v1.2.0 support by Falo. UI built with MyGo.", Buttons: []string{"OK"}})
		}}}},
	}
}
func (a *App) languageMenu() []*mygo.MenuItem {
	items := make([]*mygo.MenuItem, len(languageCodes))
	for i, code := range languageCodes {
		items[i] = &mygo.MenuItem{Label: languageNames[i], Click: func(*mygo.MenuItem, *mygo.Window) {
			a.post(func() {
				if err := a.SetLanguage(code); err != nil {
					a.Status = err.Error()
				}
			})
		}}
	}
	return items
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
		if !a.languageChosen {
			a.setLanguage(systemLanguage(mygo.App.PreferredLanguages()), false)
		}
		mygo.Theme.SetSource(mygo.ThemeSystem)
		a.dark = mygo.Theme.IsDark()
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "FETH Save Editor", Width: 1120, Height: 780, MinWidth: 860, MinHeight: 600, StateKey: "main", Content: ui.View(a.View)})
		if err := a.win.SetIcon(assets.WindowIcon); err != nil {
			a.Status = err.Error()
		}
		a.refreshLocale()
		mygo.Theme.OnUpdated(func() { a.post(func() { a.dark = mygo.Theme.IsDark() }) })
	})
	return mygo.App.Run()
}
