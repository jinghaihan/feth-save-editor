// Package save edits mapped fields while preserving every unknown save byte.
package save

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"
)

const Size = 0x25b20 + 12

type Buffer struct {
	data, original []byte
	ChecksumValid  bool
}
type Difference struct {
	Offset int  `json:"offset"`
	Before byte `json:"before"`
	After  byte `json:"after"`
}

func Checksum(b []byte) uint32 {
	var sum uint32
	for _, v := range b[12:] {
		sum += uint32(v)
	}
	return sum
}
func Digest(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func Open(path string) (*Buffer, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}
func Parse(b []byte) (*Buffer, error) {
	if len(b) != Size {
		return nil, fmt.Errorf("expected a 1.2.0 slot/auto file of %d bytes; got %d. Suspend and system files cannot be edited here", Size, len(b))
	}
	if binary.LittleEndian.Uint32(b[4:]) != 23 || int(binary.LittleEndian.Uint32(b[8:])) != len(b) {
		return nil, fmt.Errorf("unsupported save version or declared size")
	}
	return &Buffer{bytes.Clone(b), bytes.Clone(b), binary.LittleEndian.Uint32(b) == Checksum(b)}, nil
}
func (s *Buffer) Hash() string { return Digest(s.original) }
func (s *Buffer) FinishedBytes() []byte {
	b := bytes.Clone(s.data)
	binary.LittleEndian.PutUint32(b, Checksum(b))
	return b
}
func (s *Buffer) Differences() []Difference {
	out := []Difference{}
	for i := 12; i < len(s.data); i++ {
		if s.data[i] != s.original[i] {
			out = append(out, Difference{i, s.original[i], s.data[i]})
		}
	}
	return out
}
func (s *Buffer) ChangedBytes() int {
	n := 0
	for i := 12; i < len(s.data); i++ {
		if s.data[i] != s.original[i] {
			n++
		}
	}
	return n
}
func (s *Buffer) Clone() *Buffer {
	return &Buffer{bytes.Clone(s.data), bytes.Clone(s.original), s.ChecksumValid}
}

var gift = regexp.MustCompile(`(?i)^gifts\[(\d+)\]$`)

func alias(path string) (string, error) {
	if m := gift.FindStringSubmatch(path); m != nil {
		var i int
		fmt.Sscan(m[1], &i)
		if i >= 245 {
			return "", fmt.Errorf("gift index outside range")
		}
		if i < 225 {
			return fmt.Sprintf("Player.GiftItems[%d]", i), nil
		}
		return fmt.Sprintf("Player.GiftItems2[%d]", i-225), nil
	}
	return path, nil
}
func (s *Buffer) location(path string) (Location, error) {
	path, err := alias(path)
	if err != nil {
		return Location{}, err
	}
	return resolve("SaveData_V23", path)
}
func (s *Buffer) Get(path string) (any, error) {
	if strings.EqualFold(path, "playerName") {
		l, _ := s.location("PlayerName")
		return DecodeName(s.data[l.Offset : l.Offset+l.Length]), nil
	}
	l, err := s.location(path)
	if err != nil {
		return nil, err
	}
	return readNumber(s.data, l)
}
func (s *Buffer) Number(path string) (int64, error) {
	l, err := s.location(path)
	if err != nil {
		return 0, err
	}
	return readNumber(s.data, l)
}
func (s *Buffer) Record(path string) (any, error) {
	l, err := s.location(path)
	if err != nil {
		return nil, err
	}
	return record(s.data, l), nil
}
func DecodeName(b []byte) string {
	var out []byte
	for i := 0; i < len(b) && b[i] != 0; {
		if b[i] == 0x1b {
			i += 3
		} else {
			out = append(out, b[i])
			i++
		}
	}
	return strings.ToValidUTF8(string(out), "�")
}
func EncodeName(name string) []byte {
	out := make([]byte, 40)
	copy(out, []byte{0x1b, 0x4e, 0x30})
	n := 3
	for _, r := range name {
		size := utf8.RuneLen(r)
		if n+size > 37 {
			break
		}
		n += utf8.EncodeRune(out[n:], r)
	}
	copy(out[n:], []byte{0x1b, 0x4e, 0x30})
	return out
}
func (s *Buffer) ExportCharacter(slot int) ([]byte, error) {
	l, err := s.location(fmt.Sprintf("Characters[%d]", slot))
	if err != nil {
		return nil, err
	}
	return bytes.Clone(s.data[l.Offset : l.Offset+l.Size]), nil
}
