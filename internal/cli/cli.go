// Package cli implements the desktop editor's JSON command-line interface.
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jinghaihan/feth-save-editor/internal/catalog"
	"github.com/jinghaihan/feth-save-editor/internal/save"
)

type Patch struct {
	ExpectedSHA256 *string          `json:"expectedSha256"`
	Operations     []save.Operation `json:"operations"`
}

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" {
		fmt.Fprintln(stdout, "FethEditor.Cli inspect|get|apply|inspect-system|apply-system|catalog|export-character|import-character --input <save> [options]")
		return 0
	}
	result, err := execute(args)
	if err != nil {
		json.NewEncoder(stderr).Encode(map[string]string{"error": err.Error(), "type": "EditorError"})
		return 1
	}
	if err = json.NewEncoder(stdout).Encode(result); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}
func options(args []string) (map[string]string, error) {
	out := map[string]string{}
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") {
			return nil, fmt.Errorf("expected an --option, got %s", args[i])
		}
		key := strings.ToLower(strings.TrimPrefix(args[i], "--"))
		if key == "dry-run" || key == "in-place" {
			out[key] = "true"
		} else {
			i++
			if i >= len(args) {
				return nil, fmt.Errorf("missing value for --%s", key)
			}
			out[key] = args[i]
		}
	}
	return out, nil
}
func required(o map[string]string, key string) (string, error) {
	v := o[key]
	if strings.TrimSpace(v) == "" {
		return "", fmt.Errorf("missing --%s", key)
	}
	return v, nil
}
func patch(o map[string]string, hash string) (Patch, error) {
	var p Patch
	path, err := required(o, "patch")
	if err != nil {
		return p, err
	}
	f, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer f.Close()
	d := json.NewDecoder(f)
	d.UseNumber()
	if err = d.Decode(&p); err != nil {
		return p, err
	}
	if p.ExpectedSHA256 != nil && !strings.EqualFold(*p.ExpectedSHA256, hash) {
		return p, fmt.Errorf("input SHA-256 differs from patch expectation")
	}
	if p.Operations == nil {
		return p, fmt.Errorf("patch needs an operations array")
	}
	return p, nil
}
func destination(input string, o map[string]string) (string, error) {
	inplace := o["in-place"] != ""
	if inplace && o["output"] != "" {
		return "", fmt.Errorf("use either --in-place or --output, not both")
	}
	path := o["output"]
	if inplace {
		path = input
	}
	if path == "" {
		return "", fmt.Errorf("provide --output or --in-place")
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	source, err := filepath.Abs(input)
	if err != nil {
		return "", err
	}
	if !inplace && strings.EqualFold(source, path) {
		return "", fmt.Errorf("use --in-place to modify the input file")
	}
	if !inplace {
		if _, err := os.Stat(path); err == nil {
			return "", fmt.Errorf("output already exists: %s", path)
		} else if !os.IsNotExist(err) {
			return "", err
		}
	}
	return path, nil
}
func finish(s *save.Buffer, input string, o map[string]string) (any, error) {
	if o["in-place"] != "" && o["output"] != "" {
		return nil, fmt.Errorf("use either --in-place or --output, not both")
	}
	b := s.FinishedBytes()
	result := map[string]any{"inputSha256": s.Hash(), "resultSha256": save.Digest(b), "changedBytes": s.ChangedBytes()}
	if o["dry-run"] != "" {
		result["dryRun"] = true
		result["differences"] = s.Differences()
		return result, nil
	}
	if s.ChangedBytes() == 0 {
		return nil, fmt.Errorf("patch changed no data; output was not written")
	}
	path, err := destination(input, o)
	if err != nil {
		return nil, err
	}
	backup, err := save.WriteVerified(path, b, o["in-place"] != "", o["in-place"] != "", func(b []byte) error { _, err := save.Parse(b); return err })
	if err != nil {
		return nil, err
	}
	result["output"] = path
	var v any
	if backup != "" {
		v = backup
	}
	result["backup"] = v
	return result, nil
}
func execute(args []string) (any, error) {
	o, err := options(args[1:])
	if err != nil {
		return nil, err
	}
	db, err := catalog.Load(o["language"])
	if err != nil {
		return nil, err
	}
	command := strings.ToLower(args[0])
	section := o["section"]
	if section == "" {
		section = "all"
	}
	if command == "catalog" {
		kind, err := required(o, "type")
		if err != nil {
			return nil, err
		}
		if raw, ok := o["id"]; ok {
			id, err := strconv.Atoi(raw)
			if err != nil {
				return nil, err
			}
			return db.Detail(strings.ToLower(kind), id)
		}
		return save.Catalog(db, kind)
	}
	input, err := required(o, "input")
	if err != nil {
		return nil, err
	}
	if command == "inspect-system" || command == "apply-system" {
		s, err := save.OpenSystem(input)
		if err != nil {
			return nil, err
		}
		if command == "inspect-system" {
			return s.Inspect(db, section)
		}
		p, err := patch(o, s.Hash())
		if err != nil {
			return nil, err
		}
		changed, err := s.Apply(p.Operations)
		if err != nil {
			return nil, err
		}
		b := s.FinishedBytes()
		result := map[string]any{"inputSha256": s.Hash(), "resultSha256": save.Digest(b), "outputVersion": 7, "changedFlags": changed}
		if o["dry-run"] != "" {
			result["dryRun"] = true
			result["sourceVersion"] = s.SourceVersion
			return result, nil
		}
		if len(changed) == 0 {
			return nil, fmt.Errorf("patch changed no system flags; output was not written")
		}
		path, err := destination(input, o)
		if err != nil {
			return nil, err
		}
		backup, err := save.WriteVerified(path, b, o["in-place"] != "", o["in-place"] != "", func(b []byte) error { _, err := save.ParseSystem(b); return err })
		if err != nil {
			return nil, err
		}
		var v any
		if backup != "" {
			v = backup
		}
		result["output"] = path
		result["backup"] = v
		return result, nil
	}
	s, err := save.Open(input)
	if err != nil {
		return nil, err
	}
	switch command {
	case "inspect":
		return s.Inspect(db, section)
	case "get":
		path, err := required(o, "path")
		if err != nil {
			return nil, err
		}
		v, err := s.Get(path)
		if err != nil {
			return nil, err
		}
		return map[string]any{"path": path, "value": v, "sha256": s.Hash()}, nil
	case "apply":
		p, err := patch(o, s.Hash())
		if err != nil {
			return nil, err
		}
		if err = s.Apply(db, p.Operations); err != nil {
			return nil, err
		}
		return finish(s, input, o)
	case "export-character", "import-character":
		raw, err := required(o, "slot")
		if err != nil {
			return nil, err
		}
		slot, err := strconv.Atoi(raw)
		if err != nil {
			return nil, err
		}
		if command == "import-character" {
			file, err := required(o, "character")
			if err != nil {
				return nil, err
			}
			if err = s.Apply(db, []save.Operation{{"op": "importCharacter", "slot": slot, "file": file}}); err != nil {
				return nil, err
			}
			return finish(s, input, o)
		}
		path, err := required(o, "output")
		if err != nil {
			return nil, err
		}
		b, err := s.ExportCharacter(slot)
		if err != nil {
			return nil, err
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		_, err = save.WriteVerified(path, b, false, false, func([]byte) error { return nil })
		if err != nil {
			return nil, err
		}
		return map[string]any{"output": path, "slot": slot, "bytes": len(b)}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", command)
	}
}
