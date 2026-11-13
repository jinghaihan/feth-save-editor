package app

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

// editDraft belongs to an existing Apply/Set editor. Changing its controls
// never writes the save until that editor's original action is pressed.
type editDraft struct {
	source string
	values map[string]string
}

func (a *App) draft(panel ui.Element, paths ...string) *editDraft {
	var source strings.Builder
	for _, path := range paths {
		fmt.Fprintf(&source, "%s=%d;", path, a.number(path))
	}
	d := ui.Local(panel, "draft", func() editDraft { return editDraft{} })
	if d.source != source.String() {
		d.source = source.String()
		d.values = make(map[string]string, len(paths))
		for _, path := range paths {
			d.values[path] = a.numberText(path)
		}
	}
	return d
}

func (d *editDraft) number(a *App, c *ui.Context, label, path string) {
	text := d.values[path]
	a.numberField(c, label, &text)
	d.values[path] = text
}

func (d *editDraft) choice(a *App, c *ui.Context, label, path string, entries []catalog.Entry) {
	id, _ := strconv.Atoi(d.values[path])
	a.choice(c, label, entries, id, func(id int) { d.values[path] = strconv.Itoa(id) })
}

func (d *editDraft) operations(a *App, paths ...string) ([]save.Operation, bool) {
	ops := make([]save.Operation, 0, len(paths))
	for _, path := range paths {
		lo, hi, err := a.Save.NumberRange(path)
		if err != nil {
			a.Status = err.Error()
			return nil, false
		}
		value, ok := a.parseNumber(d.values[path], path, lo, hi)
		if !ok {
			return nil, false
		}
		ops = append(ops, save.Operation{"op": "set", "path": path, "value": value})
	}
	return ops, true
}
