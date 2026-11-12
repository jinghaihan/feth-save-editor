package save

import (
	"encoding/binary"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/jinghaihan/feth-save-editor/internal/catalog"
)

type Field struct {
	Name         string
	Offset       int
	Type         string
	Length, Size int
}
type Layout struct {
	Size   int
	Fields []Field
}
type Location struct {
	Offset       int
	Type         string
	Length, Size int
}

var Layouts map[string]Layout
var segment = regexp.MustCompile(`^([A-Za-z_][A-Za-z_0-9]*)(?:\[(\d+)\])?$`)

func init() {
	if err := catalog.Decode("schema", &Layouts); err != nil {
		panic(err)
	}
}
func resolve(root, path string) (Location, error) {
	l := Location{Offset: 12, Type: root}
	for _, part := range strings.Split(path, ".") {
		m := segment.FindStringSubmatch(part)
		if m == nil {
			return l, fmt.Errorf("invalid field path: %s", path)
		}
		layout, ok := Layouts[l.Type]
		if !ok || l.Length > 0 {
			return l, fmt.Errorf("unknown field: %s", path)
		}
		var f Field
		found := false
		for _, v := range layout.Fields {
			if strings.EqualFold(v.Name, m[1]) {
				f = v
				found = true
				break
			}
		}
		if !found {
			return l, fmt.Errorf("unknown field: %s", path)
		}
		l = Location{l.Offset + f.Offset, f.Type, f.Length, f.Size}
		if m[2] != "" {
			i, err := strconv.Atoi(m[2])
			if err != nil || i < 0 || i >= f.Length {
				return l, fmt.Errorf("index outside %s[%d]", f.Name, f.Length)
			}
			l.Offset += i * f.Size
			l.Length = 0
		}
	}
	return l, nil
}
func readNumber(b []byte, l Location) (int64, error) {
	if l.Length != 0 || l.Offset < 0 || l.Offset+l.Size > len(b) {
		return 0, fmt.Errorf("path must point to a numeric field")
	}
	x := b[l.Offset:]
	switch l.Type {
	case "Byte":
		return int64(x[0]), nil
	case "SByte":
		return int64(int8(x[0])), nil
	case "Int16":
		return int64(int16(binary.LittleEndian.Uint16(x))), nil
	case "UInt16":
		return int64(binary.LittleEndian.Uint16(x)), nil
	case "Int32":
		return int64(int32(binary.LittleEndian.Uint32(x))), nil
	case "UInt32":
		return int64(binary.LittleEndian.Uint32(x)), nil
	}
	return 0, fmt.Errorf("path must point to a numeric field")
}
func writeNumber(b []byte, l Location, v int64) error {
	if _, err := readNumber(b, l); err != nil {
		return err
	}
	min, max := int64(0), int64(0)
	switch l.Type {
	case "Byte":
		max = 255
	case "SByte":
		min, max = -128, 127
	case "Int16":
		min, max = -32768, 32767
	case "UInt16":
		max = 65535
	case "Int32":
		min, max = -2147483648, 2147483647
	case "UInt32":
		max = 4294967295
	}
	if v < min || v > max {
		return fmt.Errorf("value %d outside %s range", v, l.Type)
	}
	x := b[l.Offset:]
	switch l.Size {
	case 1:
		x[0] = byte(v)
	case 2:
		binary.LittleEndian.PutUint16(x, uint16(v))
	case 4:
		binary.LittleEndian.PutUint32(x, uint32(v))
	}
	return nil
}

// Record returns a decoded view; writing always patches the original bytes.
func record(b []byte, l Location) any {
	if l.Length > 0 {
		out := make([]any, l.Length)
		for i := range out {
			out[i] = record(b, Location{l.Offset + i*l.Size, l.Type, 0, l.Size})
		}
		return out
	}
	if layout, ok := Layouts[l.Type]; ok {
		out := map[string]any{}
		for _, f := range layout.Fields {
			out[f.Name] = record(b, Location{l.Offset + f.Offset, f.Type, f.Length, f.Size})
		}
		return out
	}
	v, _ := readNumber(b, l)
	return v
}
