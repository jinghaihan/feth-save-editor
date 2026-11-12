// Package catalog contains the original editor's version-1.2.0 game data.
package catalog

import (
	"compress/gzip"
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed data/*.json.gz
var data embed.FS

type Entry struct {
	ID              int      `json:"id"`
	Name            string   `json:"name"`
	MainCharacterID int      `json:"mainCharacterId"`
	Ranks           []string `json:"ranks,omitempty"`
	MaximumRank     string   `json:"maximumRank,omitempty"`
	Rank            string   `json:"rank,omitempty"`
	Points          int      `json:"points,omitempty"`
	Experience      int      `json:"experience,omitempty"`
	Stamina         int      `json:"stamina,omitempty"`
	Skill           int      `json:"skill,omitempty"`
}
type Detail struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Details string `json:"details"`
}
type Database struct {
	Names   map[string]map[int]string `json:"names"`
	Catalog map[string][]Entry        `json:"catalog"`
	Details map[string][]Detail       `json:"details"`
	Strings map[string]string         `json:"strings"`
}
type UnitRule struct {
	ID, Record, MaximumHP int
	MaximumStats          map[string]int
}
type ClassRule struct {
	ID, BaseHP, MaxExp int
	BaseStats          map[string]int
}
type RuleData struct {
	Constants                                            map[string]int
	SkillThresholds, ProfessorThresholds, EssentialItems []int
	Units                                                []UnitRule
	Classes                                              []ClassRule
	Durability                                           map[int]int
	Eligible                                             [][]int
	Playable                                             []int
}

var Rules RuleData
var cache sync.Map

func Decode(name string, value any) error {
	f, err := data.Open("data/" + name + ".json.gz")
	if err != nil {
		return err
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer z.Close()
	return json.NewDecoder(z).Decode(value)
}
func init() {
	if err := Decode("rules", &Rules); err != nil {
		panic(err)
	}
}
func Load(language string) (*Database, error) {
	if language == "" {
		language = "en_u"
	}
	language = strings.ToLower(language)
	if v, ok := cache.Load(language); ok {
		return v.(*Database), nil
	}
	var db Database
	if err := Decode(language, &db); err != nil {
		return nil, fmt.Errorf("unknown game language %q: %w", language, err)
	}
	value, _ := cache.LoadOrStore(language, &db)
	return value.(*Database), nil
}
func (d *Database) Name(kind string, id int) string {
	if kind == "battalions" && id == 200 {
		return "----"
	}
	if value, ok := d.Names[kind][id]; ok {
		return value
	}
	for _, v := range d.Catalog[kind] {
		if v.ID == id {
			return v.Name
		}
	}
	if id == -1 {
		return "----"
	}
	return "???"
}
func (d *Database) Entry(kind string, id int) (Entry, bool) {
	for _, v := range d.Catalog[kind] {
		if v.ID == id {
			return v, true
		}
	}
	return Entry{}, false
}
func (d *Database) Detail(kind string, id int) (Detail, error) {
	for _, v := range d.Details[kind] {
		if v.ID == id {
			return v, nil
		}
	}
	return Detail{}, fmt.Errorf("no database detail for %s ID %d", kind, id)
}
func Eligible(record, class int) bool {
	if record < 0 || record >= len(Rules.Eligible) {
		return false
	}
	for _, v := range Rules.Eligible[record] {
		if v == class {
			return true
		}
	}
	return false
}
func Playable(record int) bool {
	for _, v := range Rules.Playable {
		if v == record {
			return true
		}
	}
	return false
}
func Record(unit int) int {
	if unit < 0 || unit >= len(Rules.Units) {
		return -1
	}
	return Rules.Units[unit].Record
}
func ClassExp(class int) int {
	if class < 0 || class >= len(Rules.Classes) {
		return 0
	}
	return Rules.Classes[class].MaxExp
}
func Maximum(unit, class int, stat string) int {
	if unit < 0 || unit >= len(Rules.Units) || class < 0 || class >= len(Rules.Classes) {
		return 0
	}
	if strings.EqualFold(stat, "HP") {
		return Rules.Units[unit].MaximumHP + Rules.Classes[class].BaseHP
	}
	for k, v := range Rules.Units[unit].MaximumStats {
		if strings.EqualFold(k, stat) {
			return v
		}
	}
	return 0
}
