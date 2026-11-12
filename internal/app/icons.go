package app

import (
	"embed"
	"github.com/egoist/mygo/ui"
)

// Lucide icons, distributed through Iconify (ISC license).
//
//go:embed icons/*.svg
var iconFiles embed.FS
var icons = map[string]*ui.SVG{}
var pageIcons = map[string]string{"Main": "house", "Items": "package", "Roster": "users", "Battalions": "swords", "Quests": "flag", "Support": "heart", "Database": "database", "NG+ Roster": "graduation-cap", "NG+ Support": "heart", "System": "settings"}

func icon(name string) *ui.SVG {
	if value := icons[name]; value != nil {
		return value
	}
	data, err := iconFiles.ReadFile("icons/" + name + ".svg")
	if err != nil {
		return nil
	}
	value, err := ui.ParseSVG(data)
	if err != nil {
		return nil
	}
	icons[name] = value
	return value
}
