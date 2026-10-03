package discovery

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("discovery: open %s: %w", path, err)
	}
	// Split into documents textually (on `---`) rather than with a streaming YAML
	// decoder: the decoder aborts the whole stream on the first undecodable
	// document, so one Helm-templated or malformed doc used to drop every
	// RequestAuthentication/AuthorizationPolicy after it (and, via WalkDir,
	// abort the rest of the repo). Each document is handed to fn independently;
	// fn tolerates the ones it cannot parse.
	for _, doc := range splitYAMLDocuments(data) {
		if strings.TrimSpace(string(doc)) == "" {
			continue
		}
		if err := fn(path, doc); err != nil {
			return err
		}
	}
	return nil
}

// splitYAMLDocuments splits a multi-document YAML stream into individual documents
// on `---` separator lines (and `...` end markers), so a single malformed or
// templated document does not prevent the others from being read.
func splitYAMLDocuments(data []byte) [][]byte {
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	var docs [][]byte
	var cur []string
	flush := func() { docs = append(docs, []byte(strings.Join(cur, "\n"))); cur = nil }
	for _, ln := range lines {
		t := strings.TrimRight(ln, " \t")
		if t == "---" || strings.HasPrefix(t, "--- ") || t == "..." {
			flush()
			continue
		}
		cur = append(cur, ln)
	}
	flush()
	return docs
}
