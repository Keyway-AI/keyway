package discovery

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// WalkYAML invokes fn for each YAML document found under the given paths. A path
// may be a file or a directory; directories are walked recursively for
// .yaml/.yml/.json files. Multi-document YAML files (--- separated) yield one
// call per document.
func WalkYAML(paths []string, fn func(path string, doc []byte) error) error {
	for _, p := range paths {
		info, err := os.Stat(p)
		if err != nil {
			return fmt.Errorf("discovery: stat %s: %w", p, err)
		}
		if info.IsDir() {
			if err := walkDir(p, fn); err != nil {
				return err
			}
			continue
		}
		if err := splitDocs(p, fn); err != nil {
			return err
		}
	}
	return nil
}

func walkDir(root string, fn func(path string, doc []byte) error) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml", ".json":
			return splitDocs(path, fn)
		}
		return nil
	})
}

func splitDocs(path string, fn func(path string, doc []byte) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("discovery: open %s: %w", path, err)
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	for {
		var node yaml.Node
		derr := dec.Decode(&node)
		if derr == io.EOF {
			return nil
		}
		if derr != nil {
			return fmt.Errorf("discovery: decode %s: %w", path, derr)
		}
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		if err := enc.Encode(&node); err != nil {
			return fmt.Errorf("discovery: re-encode %s: %w", path, err)
		}
		_ = enc.Close()
		if strings.TrimSpace(buf.String()) == "" {
			continue
		}
		if err := fn(path, buf.Bytes()); err != nil {
			return err
		}
	}
}
