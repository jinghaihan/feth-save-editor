package save

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/jinghaihan/feth-save-editor/internal/catalog"
)

const SystemFlags = 0x134 * 8
const systemFlagOffset = 12 + 37*0x74

type System struct {
	data, original []byte
	SourceVersion  int
	ChecksumValid  bool
}

func OpenSystem(path string) (*System, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseSystem(b)
}
func ParseSystem(b []byte) (*System, error) {
	if len(b) < 12 {
		return nil, fmt.Errorf("expected a version-5 or version-7 system save")
	}
	version := int(binary.LittleEndian.Uint32(b[4:]))
	expected := 0x11f8 + 12
	if version == 5 {
		expected = 0x440 + 12
	}
	if (version != 5 && version != 7) || len(b) != expected || int(binary.LittleEndian.Uint32(b[8:])) != len(b) {
		return nil, fmt.Errorf("expected a version-5 or version-7 system save")
	}
	s := &System{bytes.Clone(b), bytes.Clone(b), version, binary.LittleEndian.Uint32(b) == Checksum(b)}
	if version == 5 {
		n := make([]byte, 0x11f8+12)
		copy(n[12:], b[12:12+7*0x74])
		for i := 7; i < 37; i++ {
			binary.LittleEndian.PutUint32(n[12+i*0x74:], 0x11)
		}
		copy(n[systemFlagOffset:], b[12+7*0x74:12+7*0x74+0x114])
		binary.LittleEndian.PutUint32(n[4:], 7)
		binary.LittleEndian.PutUint32(n[8:], uint32(len(n)))
		s.data = n
	}
	return s, nil
}
func (s *System) Hash() string { return Digest(s.original) }
func (s *System) Flag(i int) (bool, error) {
	if i < 0 || i >= SystemFlags {
		return false, fmt.Errorf("flag index out of range")
	}
	return s.data[systemFlagOffset+i/8]&(1<<uint(i%8)) != 0, nil
}
func (s *System) Apply(ops []Operation) ([]int, error) {
	b := bytes.Clone(s.data)
	e := &editor{}
	for _, op := range ops {
		if e.str(op, "op") != "setSystemFlag" {
			return nil, fmt.Errorf("unknown system patch operation")
		}
		i := int(e.number(op, "index"))
		on := e.boolean(op, "value")
		if !e.index(i, SystemFlags) || e.err != nil {
			return nil, e.err
		}
		mask := byte(1 << uint(i%8))
		if on {
			b[systemFlagOffset+i/8] |= mask
		} else {
			b[systemFlagOffset+i/8] &^= mask
		}
	}
	changed := []int{}
	for i := 0; i < SystemFlags; i++ {
		mask := byte(1 << uint(i%8))
		p := systemFlagOffset + i/8
		if s.data[p]&mask != b[p]&mask {
			changed = append(changed, i)
		}
	}
	s.data = b
	return changed, nil
}
func (s *System) FinishedBytes() []byte {
	b := bytes.Clone(s.data)
	binary.LittleEndian.PutUint32(b, Checksum(b))
	return b
}
func (s *System) Inspect(db *catalog.Database, section string) (map[string]any, error) {
	out := map[string]any{"format": "FE3H system save", "sha256": s.Hash(), "sourceVersion": s.SourceVersion, "checksumValid": s.ChecksumValid}
	if section == "all" || section == "slots" {
		slots := []any{}
		for i := 0; i < 37; i++ {
			p := 12 + i*0x74
			row := map[string]any{"index": i, "Flags": binary.LittleEndian.Uint32(s.data[p:]), "Playtime": binary.LittleEndian.Uint32(s.data[p+4:]), "Chapter1": binary.LittleEndian.Uint32(s.data[p+12:]), "Day": binary.LittleEndian.Uint32(s.data[p+16:]), "Chapter2": binary.LittleEndian.Uint32(s.data[p+24:]), "PlaceId": binary.LittleEndian.Uint32(s.data[p+84:]), "playerName": DecodeName(s.data[p+44 : p+84])}
			slots = append(slots, row)
		}
		out["slots"] = slots
	}
	if section == "all" || section == "flags" {
		flags := []any{}
		for i := 0; i < SystemFlags; i++ {
			on, _ := s.Flag(i)
			var name any
			if i >= 8 && i < 108 {
				name = db.Name("system-flags", i)
			}
			flags = append(flags, map[string]any{"index": i, "enabled": on, "name": name})
		}
		out["flags"] = flags
	}
	if len(out) == 4 {
		return nil, fmt.Errorf("unknown system section: %s", section)
	}
	return out, nil
}
