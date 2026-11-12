package save

import (
	"fmt"
	"strings"

	"github.com/jinghaihan/feth-save-editor/internal/catalog"
)

func (s *Buffer) ProtagonistGender() (string, error) {
	found := ""
	for i := 0; i < 60; i++ {
		p := fmt.Sprintf("Characters[%d].data.", i)
		id, _ := s.Number(p + "Id")
		level, _ := s.Number(p + "Level")
		if (id == 0 || id == 1) && level > 0 {
			if found != "" {
				return "", fmt.Errorf("ambiguous protagonist records")
			}
			found = "male"
			if id == 1 {
				found = "female"
			}
		}
	}
	return found, nil
}
func rankFor(db *catalog.Database, index int, points int64) string {
	entry, _ := db.Entry("supports", index)
	rank := "None"
	for _, r := range entry.Ranks {
		for _, p := range db.Catalog["support-ranks"] {
			if p.Rank == r && int64(p.Points) <= points {
				rank = r
			}
		}
	}
	return rank
}
func (s *Buffer) Inheritance(db *catalog.Database) map[string]any {
	e := &editor{s: s, db: db}
	base := e.player()
	supports := []any{}
	characters := []any{}
	for i := 0; i < 270; i++ {
		entry, _ := db.Entry("supports", i)
		supports = append(supports, map[string]any{"index": i, "name": entry.Name, "maxPoints": e.support(i, true), "maximumRank": entry.MaximumRank})
	}
	for r := 0; r < 45; r++ {
		skills := []int{}
		classes := []int{}
		for i := 0; i < 11; i++ {
			skills = append(skills, int(s.data[base+0x17d8+r*11+i]))
		}
		for i := 0; i < 100; i++ {
			if s.data[base+0x19c7+r*13+i/8]&(1<<uint(i%8)) != 0 {
				classes = append(classes, i)
			}
		}
		characters = append(characters, map[string]any{"recordIndex": r, "name": db.Name("records", r), "skillRanks": skills, "masteredClassIds": classes})
	}
	return map[string]any{"professorRank": s.data[base+0x17ce], "supports": supports, "characters": characters, "note": "NG+ history is separate from current-run progress. Test changes on a copy before using the save in-game."}
}
func (s *Buffer) Inspect(db *catalog.Database, section string) (map[string]any, error) {
	out := map[string]any{"format": "FE3H 1.1.0-1.2.0 / save version 23", "sha256": s.Hash(), "checksumValid": s.ChecksumValid, "note": "Mapped current-run fields and separately reported NG+ journal history; other unknown data is preserved."}
	requested := func(name string) bool { return section == "all" || section == name }
	get := func(path string) int64 { n, _ := s.Number(path); return n }
	item := func(path string, slot int) map[string]any {
		return map[string]any{"slot": slot, "Id": get(path + "Id"), "name": db.Name("items", int(get(path+"Id"))), "Durability": get(path + "Durability"), "Amount": get(path + "Amount")}
	}
	if requested("inheritance") {
		out["inheritance"] = s.Inheritance(db)
	}
	if requested("summary") {
		gender, err := s.ProtagonistGender()
		if err != nil {
			return nil, err
		}
		var g any
		if gender != "" {
			g = gender
		}
		name, _ := s.Get("playerName")
		summary := map[string]any{"playerName": name, "protagonistGender": g, "renown": get("Activities.Reputation"), "professorExp": get("Activities.InstructExp"), "ItemCount": get("ItemCount")}
		for _, field := range []string{"Playtime", "Money", "Chapter", "Difficulty", "Gamestyle", "Route", "MapID"} {
			summary[field] = get("Player." + field)
		}
		out["summary"] = summary
	}
	if requested("characters") {
		rows := []any{}
		for i := 0; i < 60; i++ {
			p := fmt.Sprintf("Characters[%d].data.", i)
			row := map[string]any{"slot": i, "id": get(p + "Id"), "name": db.Name("characters", int(get(p+"Id"))), "className": db.Name("classes", int(get(p+"Class")))}
			for _, field := range []string{"Level", "Exp", "Class", "CurrentClassExp", "CurrentClassLevel", "HP", "Strength", "Magic", "Dexterity", "Speed", "Luck", "Defense", "Resistance", "Movement", "Charm", "Motivation", "RNG_VALUE", "AdjutantId", "SkillExp", "SkillLevel", "ClassExp", "ClassLevel", "CombatArts", "Abilities", "EquippedAbilities", "EquippedCombatArts", "Flags", "ClassUnlockFlags", "ClassFlags", "ItemCount"} {
				v, _ := s.Record(p + field)
				row[field] = v
			}
			items := []any{}
			for j := 0; j < 6; j++ {
				items = append(items, item(fmt.Sprintf("%sItems[%d].", p, j), j))
			}
			row["items"] = items
			rows = append(rows, row)
		}
		out["characters"] = rows
	}
	if requested("inventory") {
		items := []any{}
		for i := 0; i < 400; i++ {
			items = append(items, item(fmt.Sprintf("Items[%d].", i), i))
		}
		misc, gifts := []any{}, []any{}
		for i := 0; i < 223; i++ {
			misc = append(misc, map[string]any{"id": i, "name": db.Name("misc", i), "value": get(fmt.Sprintf("Player.MiscItems[%d]", i))})
		}
		for i := 0; i < 245; i++ {
			gifts = append(gifts, map[string]any{"id": i, "name": db.Name("gifts", i), "value": get(fmt.Sprintf("gifts[%d]", i))})
		}
		out["inventory"] = map[string]any{"items": items, "misc": misc, "gifts": gifts}
	}
	if requested("battalions") {
		e := &editor{s: s, db: db}
		rows := []any{}
		for i := 0; i < 200; i++ {
			p := battalionPath(i)
			typ := int(get(p + "Type"))
			var maximum any
			if v, ok := e.full(typ); ok {
				maximum = v
			}
			row := map[string]any{"slot": i, "Stamina": e.endurance(i), "storedStamina": get(p + "Stamina"), "maximumEndurance": maximum, "name": db.Name("battalions", typ)}
			for _, f := range []string{"CharacterId", "Exp", "Type", "Skill"} {
				row[f] = get(p + f)
			}
			rows = append(rows, row)
		}
		if e.err != nil {
			return nil, e.err
		}
		out["battalions"] = rows
	}
	if requested("activities") {
		activity := map[string]any{}
		for _, f := range []string{"Reputation", "InstructExp", "ActivityExplore", "ActivityLesson", "ActivityBattle", "Statue1", "Statue2", "Statue3", "Statue4"} {
			activity[f] = get("Activities." + f)
		}
		log := map[string]any{}
		for _, f := range []string{"Wark", "Lecture", "ToBtl", "Rest", "Trnmnt", "Sing", "Lunch", "Cooking", "Drill", "Teaparty", "SCOUT"} {
			log["PlayLog_"+f] = get("Activities.PlayLog_" + f)
		}
		activity["playLog"] = log
		quests := []any{}
		for i := 0; i < 150; i++ {
			quests = append(quests, map[string]any{"id": i, "name": db.Name("quests", i), "value": get(fmt.Sprintf("Activities.QuestStateList[%d]", i))})
		}
		activity["quests"] = quests
		out["activities"] = activity
	}
	if requested("supports") {
		rows := []any{}
		for i := 0; i < 270; i++ {
			n := get(fmt.Sprintf("Player.CharacterSupportValues[%d]", i))
			entry, _ := db.Entry("supports", i)
			rows = append(rows, map[string]any{"id": i, "name": entry.Name, "value": n, "rank": rankFor(db, i, n), "maximumRank": entry.MaximumRank})
		}
		out["supports"] = rows
	}
	if len(out) == 4 {
		return nil, fmt.Errorf("unknown section: %s", section)
	}
	return out, nil
}
func Catalog(db *catalog.Database, kind string) (any, error) {
	kind = strings.ToLower(kind)
	entries, ok := db.Catalog[kind]
	if !ok {
		return nil, fmt.Errorf("unknown catalog: %s", kind)
	}
	out := []any{}
	for _, v := range entries {
		row := map[string]any{"id": v.ID, "name": v.Name}
		switch kind {
		case "characters":
			row["mainCharacterId"] = v.MainCharacterID
		case "supports":
			row["ranks"] = v.Ranks
			row["maximumRank"] = v.MaximumRank
		case "support-ranks":
			row = map[string]any{"rank": v.Rank, "points": v.Points}
		case "obtainable-battalions":
			row["experience"] = v.Experience
			row["stamina"] = v.Stamina
			row["skill"] = v.Skill
		}
		out = append(out, row)
	}
	return out, nil
}
