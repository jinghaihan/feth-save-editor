package save

import (
	"fmt"
	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"strings"
)

// NumberRange is the numeric editor range for a mapped field.
func (s *Buffer) NumberRange(path string) (int64, int64, error) {
	l, err := s.location(path)
	if err != nil {
		return 0, 0, err
	}
	if l.Length != 0 {
		return 0, 0, fmt.Errorf("expected numeric field")
	}
	lo, hi := int64(0), int64(0)
	switch l.Type {
	case "Byte":
		hi = 255
	case "SByte":
		lo, hi = -128, 127
	case "Int16":
		lo, hi = -32768, 32767
	case "UInt16":
		hi = 65535
	case "Int32":
		lo, hi = -2147483648, 2147483647
	case "UInt32":
		hi = 4294967295
	default:
		return 0, 0, fmt.Errorf("expected numeric field")
	}
	p := strings.ToLower(path)
	limits := map[string]int64{"player.playtime": 3599999, "player.money": 9999999, "player.difficulty": 3, "player.gamestyle": 1, "activities.reputation": 999999, "activities.instructexp": 44500}
	if value, ok := limits[p]; ok {
		hi = min(hi, value)
	}
	if m := charStat.FindStringSubmatch(p); m != nil {
		prefix := "Characters[" + m[1] + "].data."
		unit, _ := s.Number(prefix + "Id")
		class, _ := s.Number(prefix + "Class")
		hi = int64(catalog.Maximum(int(unit), int(class), m[2]))
	}
	if m := skillExp.FindStringSubmatch(p); m != nil {
		rank, _ := s.Number(m[1] + "SkillLevel[" + m[2] + "]")
		if rank >= 0 && rank < int64(len(catalog.Rules.SkillThresholds)) {
			hi = int64(catalog.Rules.SkillThresholds[rank] - 1)
		}
	}
	return lo, hi, nil
}
