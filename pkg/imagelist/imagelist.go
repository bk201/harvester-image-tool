// Package imagelist normalizes and writes lists of container image references.
package imagelist

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var docker0Prefixes = []string{"docker.io/", "index.docker.io/"}

// Normalize dedupes and sorts images in plain byte order.
func Normalize(images []string) []string {
	seen := make(map[string]struct{}, len(images))
	out := make([]string, 0, len(images))
	for _, image := range images {
		if _, ok := seen[image]; ok {
			continue
		}
		seen[image] = struct{}{}
		out = append(out, image)
	}
	sort.Strings(out)
	return out
}

// NormalizeRef strips a leading docker.io/ or index.docker.io/ prefix, for
// comparing refs sourced from different registries. It does not otherwise
// canonicalize the ref (e.g. it does not add a default registry or tag).
func NormalizeRef(ref string) string {
	for _, prefix := range docker0Prefixes {
		if trimmed, ok := strings.CutPrefix(ref, prefix); ok {
			return trimmed
		}
	}
	return ref
}

// Write writes headerComments (each prefixed with "# ") followed by one
// image per line.
func Write(w io.Writer, images []string, headerComments []string) error {
	for _, c := range headerComments {
		if _, err := fmt.Fprintf(w, "# %s\n", c); err != nil {
			return err
		}
	}
	for _, image := range images {
		if _, err := fmt.Fprintln(w, image); err != nil {
			return err
		}
	}
	return nil
}

// WriteFile writes the image list to path, creating parent directories as
// needed. The write is atomic: content is written to a temp file in the same
// directory and renamed into place.
func WriteFile(path string, images []string, headerComments []string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file in %s: %w", dir, err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := Write(tmp, images, headerComments); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file %s: %w", tmpPath, err)
	}
	if err := os.Chmod(tmpPath, 0o644); err != nil {
		return fmt.Errorf("chmod temp file %s: %w", tmpPath, err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename %s to %s: %w", tmpPath, path, err)
	}
	return nil
}
