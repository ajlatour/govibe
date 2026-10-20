// Package template is the golang-vibe-starter template govibe generates
// services from: a snapshot of it, at a pinned ref, embedded as a tarball
// (make sync makes it), with the list of its optional features, and what
// unpacks it.
package template

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	//go:embed snapshot/golang-vibe-starter.tar.gz
	archive []byte

	//go:embed snapshot/REF
	ref string

	//go:embed snapshot/features.json
	featuresJSON []byte
)

// Ref is the template's version in the snapshot: a tag, or a commit.
func Ref() string { return strings.TrimSpace(ref) }

// Feature is one the template's make init can leave out.
type Feature struct {
	Name    string
	Summary string
}

// Features lists the template's optional features, as its make init lists
// them (initservice -list-features).
func Features() ([]Feature, error) {
	var f []Feature
	if err := json.Unmarshal(featuresJSON, &f); err != nil {
		return nil, fmt.Errorf("the snapshot's features.json: %w", err)
	}
	return f, nil
}

// Unpack writes the template into dir, which must be empty or not exist.
func Unpack(dir string) error {
	return unpack(bytes.NewReader(archive), dir)
}

// maxFile bounds a file in the archive: the snapshot is ours, but a bound
// keeps a corrupt one from filling the disk.
const maxFile = 64 << 20

func unpack(r io.Reader, dir string) error {
	if entries, err := os.ReadDir(dir); err == nil && len(entries) > 0 {
		return fmt.Errorf("%s isn't empty: give a new directory", dir)
	} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("the snapshot: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("the snapshot: %w", err)
		}
		if h.Typeflag == tar.TypeXGlobalHeader {
			continue // git archive's, with the commit: nothing to write
		}
		name := path.Clean(h.Name)
		if name == "." || path.IsAbs(name) || name == ".." || strings.HasPrefix(name, "../") {
			return fmt.Errorf("the snapshot names %q, outside the directory", h.Name)
		}
		target := filepath.Join(dir, filepath.FromSlash(name))
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if h.Size > maxFile {
				return fmt.Errorf("the snapshot's %s is %d bytes, more than a source file should be", name, h.Size)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			mode := os.FileMode(0o644)
			if h.FileInfo().Mode()&0o111 != 0 {
				mode = 0o755 // scripts stay executable
			}
			f, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return err
			}
			_, err = io.Copy(f, io.LimitReader(tr, maxFile))
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return err
			}
		default:
			// Links and the rest: a template has none, and they could point
			// anywhere.
			return fmt.Errorf("the snapshot's %s isn't a file or a directory", name)
		}
	}
}
