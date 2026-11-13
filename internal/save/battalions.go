package save

import "fmt"

func battalionPath(slot int) string { return fmt.Sprintf("Player.Battalions[%d].", slot) }
func equippedPath(slot int) string {
	return fmt.Sprintf("Characters[%d].data.EquippedBattalion.", slot)
}

// EquippedBattalionSlot returns the active roster slot linked to this entry,
// or -1 when the battalion is not equipped.
func (s *Buffer) EquippedBattalionSlot(slot int) (int, error) {
	e := &editor{s: s}
	linked := e.linked(slot)
	return linked, e.err
}
func (e *editor) active(slot int) bool {
	p := fmt.Sprintf("Characters[%d].data.", slot)
	return e.n(p+"Id") >= 0 && e.n(p+"Level") > 0
}
func (e *editor) linked(slot int) int {
	if !e.index(slot, 200) {
		return -1
	}
	p := battalionPath(slot)
	id, typ := e.n(p+"CharacterId"), e.n(p+"Type")
	if id < 0 || typ == 200 {
		return -1
	}
	characters := []int{}
	for i := 0; i < 60; i++ {
		if e.active(i) && e.n(fmt.Sprintf("Characters[%d].data.Id", i)) == id && e.n(equippedPath(i)+"Type") == typ {
			characters = append(characters, i)
		}
	}
	if len(characters) == 0 {
		return -1
	}
	n := 0
	for i := 0; i < 200; i++ {
		q := battalionPath(i)
		if e.n(q+"CharacterId") == id && e.n(q+"Type") == typ {
			n++
		}
	}
	if len(characters) != 1 || n != 1 {
		e.fail("the equipped battalion cannot be uniquely matched to its barracks entry")
		return -1
	}
	return characters[0]
}
func (e *editor) full(typ int) (int, bool) {
	entry, ok := e.db.Entry("obtainable-battalions", typ)
	return entry.Stamina, ok
}
func (e *editor) endurance(slot int) int64 {
	if linked := e.linked(slot); linked >= 0 {
		return e.n(equippedPath(linked) + "Stamina")
	}
	return e.n(battalionPath(slot) + "Stamina")
}
func (e *editor) setEndurance(slot int, v int64) {
	linked := e.linked(slot)
	e.put(battalionPath(slot)+"Stamina", v)
	if linked >= 0 {
		e.put(equippedPath(linked)+"Stamina", v)
	}
}
func (e *editor) replenish(slot int) {
	p := battalionPath(slot)
	full, ok := e.full(int(e.n(p + "Type")))
	if !ok {
		e.fail("full endurance is unknown for this battalion type")
		return
	}
	e.setEndurance(slot, int64(full))
}
func (e *editor) replenishCharacter(slot int) {
	if !e.index(slot, 60) || !e.active(slot) {
		e.fail("no active character is selected")
		return
	}
	p := equippedPath(slot)
	typ := e.n(p + "Type")
	full, ok := e.full(int(typ))
	if !ok {
		e.fail("full endurance is unknown for this battalion type")
		return
	}
	id := e.n(fmt.Sprintf("Characters[%d].data.Id", slot))
	matches := []int{}
	for i := 0; i < 200; i++ {
		q := battalionPath(i)
		if e.n(q+"CharacterId") == id && e.n(q+"Type") == typ {
			matches = append(matches, i)
		}
	}
	if len(matches) > 0 {
		if len(matches) != 1 || e.linked(matches[0]) != slot {
			e.fail("the equipped battalion cannot be uniquely matched to its barracks entry")
			return
		}
		e.replenish(matches[0])
	} else {
		e.put(p+"Stamina", int64(full))
	}
}
func (e *editor) battalionOperation(op string, o Operation) {
	slot := func() int { return int(e.number(o, "slot")) }
	switch op {
	case "setBattalionEndurance":
		e.setEndurance(slot(), e.number(o, "endurance"))
	case "setBattalionEnduranceValues":
		i := slot()
		if !e.index(i, 200) {
			return
		}
		e.put(battalionPath(i)+"Stamina", e.number(o, "storedEndurance"))
		if _, ok := o["equippedEndurance"]; ok {
			linked := e.linked(i)
			if linked < 0 {
				e.fail("this battalion is not equipped by an active character")
				return
			}
			e.put(equippedPath(linked)+"Stamina", e.number(o, "equippedEndurance"))
		}
	case "replenishBattalion":
		e.replenish(slot())
	case "replenishCharacterBattalion":
		e.replenishCharacter(slot())
	case "replenishBattalions":
		linked := map[int]bool{}
		for i := 0; i < 200; i++ {
			typ := int(e.n(battalionPath(i) + "Type"))
			if typ == 200 {
				continue
			}
			j := e.linked(i)
			if j >= 0 {
				linked[j] = true
			}
			if _, ok := e.full(typ); ok {
				e.replenish(i)
			}
		}
		for i := 0; i < 60; i++ {
			typ := int(e.n(equippedPath(i) + "Type"))
			if !linked[i] && e.active(i) && typ != 200 {
				if full, ok := e.full(typ); ok {
					e.put(equippedPath(i)+"Stamina", int64(full))
				}
			}
		}
	case "maxBattalionLevel":
		i := slot()
		j := e.linked(i)
		if e.n(battalionPath(i)+"Type") >= 200 {
			e.fail("no battalion is selected")
			return
		}
		e.put(battalionPath(i)+"Exp", max(e.n(battalionPath(i)+"Exp"), 400))
		if j >= 0 {
			e.put(equippedPath(j)+"Exp", max(e.n(equippedPath(j)+"Exp"), 400))
		}
	case "maxBattalionLevels":
		for i := 0; i < 200; i++ {
			p := battalionPath(i)
			if e.n(p+"Type") < 200 {
				e.linked(i)
				e.put(p+"Exp", max(e.n(p+"Exp"), 400))
			}
		}
		for i := 0; i < 60; i++ {
			p := equippedPath(i)
			if e.active(i) && e.n(p+"Type") < 200 {
				e.put(p+"Exp", max(e.n(p+"Exp"), 400))
			}
		}
	case "deleteBattalion":
		i := slot()
		j := e.linked(i)
		p := battalionPath(i)
		if e.n(p+"Type") >= 200 {
			e.fail("no battalion is selected")
			return
		}
		prefixes := []string{p}
		if j >= 0 {
			prefixes = append(prefixes, equippedPath(j))
		}
		for _, q := range prefixes {
			for field, v := range map[string]int64{"CharacterId": -1, "Exp": 0, "Stamina": 0, "Type": 200, "Skill": 80} {
				e.put(q+field, v)
			}
		}
	case "fillMissingBattalions":
		owned := map[int]bool{}
		free := []int{}
		for i := 0; i < 200; i++ {
			typ := int(e.n(battalionPath(i) + "Type"))
			if typ < 200 {
				owned[typ] = true
			}
			if typ == 200 {
				free = append(free, i)
			}
		}
		for i := 0; i < 60; i++ {
			if e.active(i) {
				typ := int(e.n(equippedPath(i) + "Type"))
				if typ < 200 {
					owned[typ] = true
				}
			}
		}
		missing := 0
		for _, v := range e.db.Catalog["obtainable-battalions"] {
			if !owned[v.ID] {
				missing++
			}
		}
		if missing > len(free) {
			e.fail("need %d free battalion slots, but only %d remain", missing, len(free))
			return
		}
		n := 0
		for _, v := range e.db.Catalog["obtainable-battalions"] {
			if owned[v.ID] {
				continue
			}
			p := battalionPath(free[n])
			n++
			for f, value := range map[string]int64{"CharacterId": -1, "Exp": 400, "Stamina": int64(v.Stamina), "Type": int64(v.ID), "Skill": int64(v.Skill)} {
				e.set(p+f, value)
			}
		}
		if n > 0 {
			e.operation(Operation{"op": "sortBattalions"})
		}
	}
}
