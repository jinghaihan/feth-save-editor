package save

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/jinghaihan/feth-save-editor/internal/catalog"
)

type Operation map[string]any
type editor struct {
	s   *Buffer
	db  *catalog.Database
	err error
}

func (e *editor) fail(format string, args ...any) {
	if e.err == nil {
		e.err = fmt.Errorf(format, args...)
	}
}
func (e *editor) n(path string) int64 {
	if e.err != nil {
		return 0
	}
	v, err := e.s.Number(path)
	if err != nil {
		e.err = err
	}
	return v
}
func (e *editor) put(path string, v int64) {
	if e.err != nil {
		return
	}
	l, err := e.s.location(path)
	if err == nil {
		err = writeNumber(e.s.data, l, v)
	}
	e.err = err
}
func (e *editor) index(i, n int) bool {
	if i < 0 || i >= n {
		e.fail("index %d outside range 0–%d", i, n-1)
		return false
	}
	return true
}
func (e *editor) bit(path string, i int, on bool) {
	if e.err != nil {
		return
	}
	l, err := e.s.location(path)
	if err != nil {
		e.err = err
		return
	}
	n := l.Size
	if l.Length > 0 {
		n *= l.Length
	}
	if !e.index(i, n*8) {
		return
	}
	p := l.Offset + i/8
	mask := byte(1 << uint(i%8))
	if on {
		e.s.data[p] |= mask
	} else {
		e.s.data[p] &^= mask
	}
}

var writable = regexp.MustCompile(`(?i)^(items\[\d+\]\.(id|durability|amount)|characters\[\d+\]\.data\.(id|rng_value|exp|level|hp|strength|magic|dexterity|speed|luck|defense|resistance|movement|charm|motivation|currentclassexp|currentclasslevel|adjutantid|flags|classflags)|characters\[\d+\]\.data\.(skillexp|classexp|classlevel|equippedabilities|equippedcombatarts|combatarts|abilities|classunlockflags)\[\d+\]|characters\[\d+\]\.data\.items\[\d+\]\.(id|durability|amount)|player\.(playtime|money|chapter|difficulty|gamestyle|route|mapid)|player\.(charactersupportvalues|miscitems|giftitems|giftitems2)\[\d+\]|player\.battalions\[\d+\]\.(characterid|exp|stamina|type|skill)|activities\.(reputation|instructexp|activityexplore|activitylesson|activitybattle|statue[1-4]|playlog_[a-z]+)|activities\.queststatelist\[\d+\])$`)
var writableBits = regexp.MustCompile(`(?i)^characters\[\d+\]\.data\.(combatarts|abilities|classunlockflags|flags|classflags)$`)
var charStat = regexp.MustCompile(`^characters\[(\d+)\]\.data\.(hp|strength|magic|dexterity|speed|luck|defense|resistance|movement|charm)$`)
var skillExp = regexp.MustCompile(`^(characters\[\d+\]\.data\.)skillexp\[(\d+)\]$`)
var classValue = regexp.MustCompile(`^(characters\[\d+\]\.data\.)class(exp|level)\[(\d+)\]$`)
var currentClass = regexp.MustCompile(`^(characters\[\d+\]\.data\.)currentclass(exp|level)$`)

func (e *editor) set(path string, v int64) {
	path, err := alias(path)
	if err != nil {
		e.err = err
		return
	}
	p := strings.ToLower(path)
	if !writable.MatchString(p) {
		e.fail("field is not writable: %s", path)
		return
	}
	limits := map[string]int64{"player.playtime": 3599999, "player.money": 9999999, "player.difficulty": 3, "player.gamestyle": 1, "activities.reputation": 999999, "activities.instructexp": 44500}
	if m, ok := limits[p]; ok && (v < 0 || v > m) {
		e.fail("value exceeds %s limit", path)
	}
	if (strings.Contains(p, ".classlevel[") || strings.HasSuffix(p, ".currentclasslevel")) && (v < 0 || v > 1) {
		e.fail("class mastery must be 0 or 1")
	}
	if strings.HasPrefix(p, "activities.queststatelist[") && (v < 0 || v > 6) {
		e.fail("quest state must be 0 through 6")
	}
	if m := charStat.FindStringSubmatch(p); m != nil {
		prefix := "Characters[" + m[1] + "].data."
		maximum := catalog.Maximum(int(e.n(prefix+"Id")), int(e.n(prefix+"Class")), m[2])
		if v < 0 || v > int64(maximum) {
			e.fail("%s exceeds this character's maximum of %d", m[2], maximum)
		}
	}
	if m := skillExp.FindStringSubmatch(p); m != nil {
		rank := int(e.n(m[1] + "SkillLevel[" + m[2] + "]"))
		if !e.index(rank, len(catalog.Rules.SkillThresholds)) {
			return
		}
		if v < 0 || v >= int64(catalog.Rules.SkillThresholds[rank]) {
			e.fail("skill experience exceeds the current rank limit")
		}
	}
	if m := classValue.FindStringSubmatch(p); m != nil && m[2] == "exp" {
		var class int
		fmt.Sscan(m[3], &class)
		if v < 0 || v > int64(catalog.ClassExp(class)) {
			e.fail("class experience exceeds the GUI limit")
		}
	}
	if m := currentClass.FindStringSubmatch(p); m != nil && m[2] == "exp" {
		if v < 0 || v > int64(catalog.ClassExp(int(e.n(m[1]+"Class")))) {
			e.fail("current class experience exceeds the GUI limit")
		}
	}
	e.put(p, v)
	if m := skillExp.FindStringSubmatch(p); m != nil {
		e.put(m[1]+"SkillExp2["+m[2]+"]", v)
	}
	if m := currentClass.FindStringSubmatch(p); m != nil {
		e.put(fmt.Sprintf("%sClass%s[%d]", m[1], m[2], e.n(m[1]+"Class")), v)
	}
	if m := classValue.FindStringSubmatch(p); m != nil {
		if fmt.Sprint(e.n(m[1]+"Class")) == m[3] {
			e.put(m[1]+"CurrentClass"+m[2], v)
		}
	}
	if strings.HasPrefix(p, "items[") && strings.HasSuffix(p, ".id") {
		e.recount()
	}
}
func (e *editor) recount() {
	var n int64
	for i := 0; i < 400; i++ {
		if e.n(fmt.Sprintf("Items[%d].Id", i)) != -1 {
			n++
		}
	}
	e.put("ItemCount", n)
}
func (e *editor) str(o Operation, k string) string {
	v, ok := o[k].(string)
	if !ok {
		e.fail("%s must be a string", k)
	}
	return v
}
func (e *editor) number(o Operation, k string) int64 {
	v, ok := o[k]
	if !ok {
		e.fail("missing %s", k)
		return 0
	}
	var n int64
	switch v := v.(type) {
	case int:
		n = int64(v)
	case int64:
		n = v
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || v != math.Trunc(v) || v >= math.Exp2(63) || v < -math.Exp2(63) {
			e.fail("%s must be an integer", k)
		} else {
			n = int64(v)
		}
	case json.Number:
		var err error
		n, err = v.Int64()
		if err != nil {
			e.err = err
		}
	default:
		e.fail("%s must be a number", k)
	}
	return n
}
func (e *editor) boolean(o Operation, k string) bool {
	v, ok := o[k].(bool)
	if !ok {
		e.fail("%s must be a boolean", k)
	}
	return v
}
func (e *editor) rank(index int, rank string) int {
	entry, ok := e.db.Entry("supports", index)
	if !ok {
		e.fail("invalid support index")
		return 0
	}
	for _, r := range entry.Ranks {
		if strings.EqualFold(r, rank) {
			for _, v := range e.db.Catalog["support-ranks"] {
				if v.Rank == r {
					return v.Points
				}
			}
		}
	}
	e.fail("support entry %d does not have rank %s", index, rank)
	return 0
}
func (e *editor) support(index int, inherited bool) int64 {
	if !e.index(index, 270) {
		return 0
	}
	if inherited {
		return int64(binaryUint16(e.s.data, e.player()+0x1576+index*2))
	}
	return e.n(fmt.Sprintf("Player.CharacterSupportValues[%d]", index))
}
func (e *editor) setSupport(index int, v int64, inherited bool) {
	if !e.index(index, 270) {
		return
	}
	if inherited {
		if v < 0 || v > 65535 {
			e.fail("support points out of range")
			return
		}
		e.raw16(e.player()+0x1576+index*2, v)
	} else {
		e.set(fmt.Sprintf("Player.CharacterSupportValues[%d]", index), v)
	}
}
func (e *editor) maxSupport(index int, inherited bool) {
	entry, ok := e.db.Entry("supports", index)
	if !ok || entry.MaximumRank == "None" {
		e.fail("this support entry has no conversations")
		return
	}
	target := int64(e.rank(index, entry.MaximumRank))
	e.setSupport(index, max(e.support(index, inherited), target), inherited)
}
func (e *editor) player() int          { l, _ := e.s.location("Player"); return l.Offset }
func binaryUint16(b []byte, i int) int { return int(b[i]) | int(b[i+1])<<8 }
func (e *editor) raw16(i int, v int64) {
	if e.err == nil {
		e.s.data[i] = byte(v)
		e.s.data[i+1] = byte(v >> 8)
	}
}
func (e *editor) journalByte(i int, v int64, max int64) {
	if v < 0 || v > max {
		e.fail("journal value out of range")
		return
	}
	if e.err == nil {
		e.s.data[i] = byte(v)
	}
}
func (e *editor) journalClass(record, class int, on bool) {
	if !e.index(record, 45) || !e.index(class, 100) {
		return
	}
	i := e.player() + 0x19c7 + record*13 + class/8
	mask := byte(1 << uint(class%8))
	if e.err == nil {
		if on {
			e.s.data[i] |= mask
		} else {
			e.s.data[i] &^= mask
		}
	}
}
func (e *editor) unlockClasses(record int) {
	if !catalog.Playable(record) {
		e.fail("record %d is not a playable character", record)
		return
	}
	for _, class := range catalog.Rules.Eligible[record] {
		e.journalClass(record, class, true)
	}
}
func (s *Buffer) Apply(db *catalog.Database, ops []Operation) error {
	e := &editor{s: s.Clone(), db: db}
	for _, op := range ops {
		e.operation(op)
		if e.err != nil {
			return e.err
		}
	}
	s.data = e.s.data
	return nil
}
func (e *editor) operation(o Operation) {
	op := e.str(o, "op")
	num := func(k string) int { return int(e.number(o, k)) }
	slot := func() int { return num("slot") }
	prefix := func() string { return fmt.Sprintf("Characters[%d].data.", slot()) }
	switch op {
	case "set":
		e.set(e.str(o, "path"), e.number(o, "value"))
	case "setName":
		l, _ := e.s.location("PlayerName")
		copy(e.s.data[l.Offset:l.Offset+40], EncodeName(e.str(o, "value")))
	case "setBit":
		p := e.str(o, "path")
		if !writableBits.MatchString(p) {
			e.fail("flag group is not writable: %s", p)
			return
		}
		e.bit(p, num("index"), e.boolean(o, "value"))
	case "setProtagonistGender":
		gender := strings.ToLower(e.str(o, "gender"))
		if gender != "male" && gender != "female" {
			e.fail("gender must be male or female")
			return
		}
		found := -1
		for i := 0; i < 60; i++ {
			p := fmt.Sprintf("Characters[%d].data.", i)
			id := e.n(p + "Id")
			if (id == 0 || id == 1) && e.n(p+"Level") > 0 {
				if found >= 0 {
					e.fail("ambiguous protagonist records")
					return
				}
				found = i
			}
		}
		if found < 0 {
			e.fail("no protagonist record")
			return
		}
		v := int64(0)
		if gender == "female" {
			v = 1
		}
		e.set(fmt.Sprintf("Characters[%d].data.Id", found), v)
	case "sortItems":
		e.sortRecords("Items", 400, 4, func(a, b []byte) int {
			ai, bi := int(int16(binaryUint16(a, 0))), int(int16(binaryUint16(b, 0)))
			if ai == -1 && bi != -1 {
				return 1
			}
			if bi == -1 && ai != -1 {
				return -1
			}
			if ai != bi {
				return ai - bi
			}
			return int(a[2]) - int(b[2])
		})
		e.recount()
	case "sortBattalions":
		e.sortRecords("Player.Battalions", 200, 8, func(a, b []byte) int {
			if a[6] == 200 && b[6] != 200 {
				return 1
			}
			if b[6] == 200 && a[6] != 200 {
				return -1
			}
			if a[6] != b[6] {
				return int(a[6]) - int(b[6])
			}
			return int(int16(binaryUint16(a, 0))) - int(int16(binaryUint16(b, 0)))
		})
	case "inventoryDurability":
		mode := e.str(o, "mode")
		if mode != "normal" && mode != "unlimited" && mode != "weapons-unlimited" {
			e.fail("invalid durability mode")
			return
		}
		for i := 0; i < 400; i++ {
			p := fmt.Sprintf("Items[%d].", i)
			id := int(e.n(p + "Id"))
			if id == -1 {
				continue
			}
			v := catalog.Rules.Durability[id]
			if mode == "unlimited" || mode == "weapons-unlimited" && id >= 10 && id < 510 {
				v = 100
			}
			e.set(p+"Durability", int64(v))
		}
	case "characterItemDurability":
		p := prefix()
		count := int(e.n(p + "ItemCount"))
		if count > 6 {
			e.fail("character item count is invalid")
			return
		}
		for i := 0; i < count; i++ {
			path := fmt.Sprintf("%sItems[%d].", p, i)
			id := int(e.n(path + "Id"))
			if id == -1 {
				continue
			}
			if _, ok := e.db.Entry("items", id); !ok {
				e.fail("unknown character item ID %d", id)
				return
			}
			e.set(path+"Durability", int64(catalog.Rules.Durability[id]))
		}
	case "setCharacterItem":
		p := prefix()
		i, id := num("itemSlot"), num("id")
		v := catalog.Rules.Durability[id]
		if _, ok := o["durability"]; ok {
			v = num("durability")
		}
		if !e.index(i, 6) {
			return
		}
		if _, ok := e.db.Entry("items", id); id == -1 || !ok {
			e.fail("choose an item from the catalog")
			return
		}
		if v < 0 || v > catalog.Rules.Durability[id] {
			e.fail("item durability exceeds normal maximum")
			return
		}
		count := int(e.n(p + "ItemCount"))
		if count > 6 {
			e.fail("character item count is invalid")
			return
		}
		for j := 0; j < 6; j++ {
			if (e.n(fmt.Sprintf("%sItems[%d].Id", p, j)) != -1) != (j < count) {
				e.fail("character item slots do not match their item count")
				return
			}
		}
		if i > count {
			e.fail("add items to the first empty slot")
			return
		}
		path := fmt.Sprintf("%sItems[%d].", p, i)
		e.set(path+"Id", int64(id))
		e.set(path+"Durability", int64(v))
		if i == count {
			e.put(p+"ItemCount", int64(count+1))
		}
	case "maxSkillExp":
		p := prefix()
		for i := 0; i < 11; i++ {
			rank := int(e.n(fmt.Sprintf("%sSkillLevel[%d]", p, i)))
			if !e.index(rank, 12) {
				return
			}
			e.set(fmt.Sprintf("%sSkillExp[%d]", p, i), int64(catalog.Rules.SkillThresholds[rank]-1))
		}
	case "setSkillRank":
		p := prefix()
		skill, rank, xp := num("skill"), num("rank"), num("experience")
		if !e.index(skill, 11) || !e.index(rank, 12) {
			return
		}
		if xp < 0 || xp >= catalog.Rules.SkillThresholds[rank] {
			e.fail("skill experience exceeds rank limit")
			return
		}
		e.put(fmt.Sprintf("%sSkillLevel[%d]", p, skill), int64(rank))
		e.put(fmt.Sprintf("%sSkillLevel2[%d]", p, skill), int64(rank))
		e.set(fmt.Sprintf("%sSkillExp[%d]", p, skill), int64(xp))
	case "setProfessorRank":
		rank := num("rank")
		if e.index(rank, 10) {
			e.set("Activities.InstructExp", int64(catalog.Rules.ProfessorThresholds[rank]))
		}
	case "setSupportRank", "setNgPlusSupportRank":
		i := num("index")
		e.setSupport(i, int64(e.rank(i, e.str(o, "rank"))), op == "setNgPlusSupportRank")
	case "maxSupportRank", "maxNgPlusSupportRank":
		e.maxSupport(num("index"), op == "maxNgPlusSupportRank")
	case "maxSupports", "maxNgPlusSupports":
		for i := 0; i < 270; i++ {
			entry, _ := e.db.Entry("supports", i)
			if entry.MaximumRank != "None" {
				e.maxSupport(i, op == "maxNgPlusSupports")
			}
		}
	case "setNgPlusProfessorRank":
		e.journalByte(e.player()+0x17ce, e.number(o, "rank"), 9)
	case "setNgPlusSupport":
		e.setSupport(num("index"), e.number(o, "points"), true)
	case "setNgPlusSkillRank":
		r, skill := num("recordIndex"), num("skill")
		if e.index(r, 45) && e.index(skill, 11) {
			e.journalByte(e.player()+0x17d8+r*11+skill, e.number(o, "rank"), 11)
		}
	case "setNgPlusClassMastery":
		e.journalClass(num("recordIndex"), num("classId"), e.boolean(o, "mastered"))
	case "unlockNgPlusClasses":
		e.unlockClasses(num("recordIndex"))
	case "unlockNgPlusRoster":
		for _, r := range catalog.Rules.Playable {
			for i := 0; i < 11; i++ {
				p := e.player() + 0x17d8 + r*11 + i
				if e.s.data[p] < 11 {
					e.journalByte(p, 11, 11)
				}
			}
			e.unlockClasses(r)
		}
	case "maxClassExp":
		p := prefix()
		r := catalog.Record(int(e.n(p + "Id")))
		if !catalog.Playable(r) {
			e.fail("no playable character rules are known")
			return
		}
		for _, class := range catalog.Rules.Eligible[r] {
			e.set(fmt.Sprintf("%sClassExp[%d]", p, class), int64(catalog.ClassExp(class)))
		}
	case "unlockAll":
		p := prefix()
		kind := e.str(o, "kind")
		switch kind {
		case "abilities":
			for i := 0; i < 240; i++ {
				e.bit(p+"Abilities", i, true)
			}
		case "combat-arts":
			for i := 0; i < 80; i++ {
				e.bit(p+"CombatArts", i, true)
			}
		default:
			e.fail("unlock kind must be abilities or combat-arts")
		}
	case "fillItems":
		kind := e.str(o, "kind")
		amount := e.number(o, "amount")
		if amount < 0 || amount > 255 {
			e.fail("amount out of range")
			return
		}
		switch kind {
		case "misc":
			for i := 0; i < 223; i++ {
				e.set(fmt.Sprintf("Player.MiscItems[%d]", i), amount)
			}
		case "gifts":
			for i := 0; i < 245; i++ {
				e.set(fmt.Sprintf("gifts[%d]", i), amount)
			}
		default:
			e.fail("fill kind must be misc or gifts")
		}
	case "addEssentialItems":
		for _, id := range catalog.Rules.EssentialItems {
			slot, free := -1, -1
			for i := 0; i < 400; i++ {
				v := e.n(fmt.Sprintf("Items[%d].Id", i))
				if v == int64(id) {
					slot = i
					break
				}
				if v == -1 && free == -1 {
					free = i
				}
			}
			if slot == -1 {
				slot = free
			}
			if slot == -1 {
				e.fail("inventory has no free slot for essential items")
				return
			}
			p := fmt.Sprintf("Items[%d].", slot)
			e.set(p+"Id", int64(id))
			e.set(p+"Durability", int64(catalog.Rules.Durability[id]))
			e.set(p+"Amount", 99)
		}
		e.operation(Operation{"op": "sortItems"})
	case "importCharacter":
		b, err := os.ReadFile(e.str(o, "file"))
		if err != nil {
			e.err = err
			return
		}
		l, err := e.s.location(fmt.Sprintf("Characters[%d]", slot()))
		if err != nil {
			e.err = err
			return
		}
		if len(b) != l.Size {
			e.fail("character record size must be %d", l.Size)
			return
		}
		copy(e.s.data[l.Offset:l.Offset+l.Size], b)
	case "fillMissingBattalions", "setBattalionEndurance", "setBattalionEnduranceValues", "replenishBattalion", "replenishCharacterBattalion", "replenishBattalions", "maxBattalionLevel", "maxBattalionLevels", "deleteBattalion":
		e.battalionOperation(op, o)
	default:
		e.fail("unknown patch operation: %s", op)
	}
}
func (e *editor) sortRecords(path string, count, size int, compare func([]byte, []byte) int) {
	l, err := e.s.location(path + "[0]")
	if err != nil {
		e.err = err
		return
	}
	records := make([][]byte, count)
	for i := range records {
		records[i] = bytes.Clone(e.s.data[l.Offset+i*size : l.Offset+(i+1)*size])
	}
	slices.SortStableFunc(records, compare)
	for i, b := range records {
		copy(e.s.data[l.Offset+i*size:], b)
	}
}
