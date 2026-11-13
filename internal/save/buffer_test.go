package save

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf8"

	"github.com/jinghaihan/feth-save-editor/internal/catalog"
)

func blank(t *testing.T) *Buffer {
	t.Helper()
	b := make([]byte, Size)
	binary.LittleEndian.PutUint32(b[4:], 23)
	binary.LittleEndian.PutUint32(b[8:], Size)
	for i := 12; i < 12+1600; i += 4 {
		binary.LittleEndian.PutUint16(b[i:], 65535)
	}
	binary.LittleEndian.PutUint32(b, Checksum(b))
	s, err := Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestPackedLayout(t *testing.T) {
	for path, want := range map[string]int{"Items[1].Id": 16, "Characters[0]": 12 + 0x644, "Player": 12 + 0x231d9, "Activities": 12 + 0x250a1} {
		l, err := resolve("SaveData_V23", path)
		if err != nil || l.Offset != want {
			t.Fatalf("%s: %+v %v want %x", path, l, err, want)
		}
	}
}
func TestTransactionAndUnknownBytes(t *testing.T) {
	s := blank(t)
	s.data[0x22000] = 0xa5
	s.original = bytes.Clone(s.data)
	db, _ := catalog.Load("en_u")
	if err := s.Apply(db, []Operation{{"op": "set", "path": "Player.Money", "value": 50000}}); err != nil {
		t.Fatal(err)
	}
	before := s.FinishedBytes()
	if before[0x22000] != 0xa5 {
		t.Fatal("unknown byte changed")
	}
	if err := s.Apply(db, []Operation{{"op": "set", "path": "Player.Money", "value": 60000}, {"op": "set", "path": "Player.Money", "value": 10000000}}); err == nil {
		t.Fatal("invalid batch accepted")
	}
	if !bytes.Equal(before, s.FinishedBytes()) {
		t.Fatal("failed operation partially applied")
	}
}
func TestNameEncoding(t *testing.T) {
	for _, name := range []string{"Byleth", "贝雷丝🦉", "很长很长很长很长很长很长很长"} {
		b := EncodeName(name)
		decoded := DecodeName(b)
		if !utf8.ValidString(decoded) || len(b) != 40 {
			t.Fatal("invalid UTF-8 or size")
		}
		if len(name) <= 34 && decoded != name {
			t.Fatalf("%q != %q", decoded, name)
		}
	}
}
func TestGiftSplitAndMirrors(t *testing.T) {
	s := blank(t)
	db, _ := catalog.Load("en_u")
	if err := s.Apply(db, []Operation{{"op": "set", "path": "gifts[225]", "value": 37}, {"op": "setSkillRank", "slot": 0, "skill": 2, "rank": 4, "experience": 123}}); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]int64{"Player.GiftItems2[0]": 37, "Characters[0].data.SkillLevel[2]": 4, "Characters[0].data.SkillLevel2[2]": 4, "Characters[0].data.SkillExp[2]": 123, "Characters[0].data.SkillExp2[2]": 123} {
		got, err := s.Number(path)
		if err != nil || got != want {
			t.Fatalf("%s=%d (%v)", path, got, err)
		}
	}
}
