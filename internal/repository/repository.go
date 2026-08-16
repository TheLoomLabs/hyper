// Package repository finds the repository root and enumerates the artefact
// files hyper check walks.
package repository

import (
	"os"
	"path/filepath"
)

// FindGitRoot walks up from start, bounded by the git root, and returns the
// directory holding .git (§9's "the repository root is found by walking up
// from the working directory, bounded by the git root"; ADR-0014). It
// returns ok=false where no .git is found before the filesystem root.
func FindGitRoot(start string) (root string, ok bool) {
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// artefactDirs are four of the five artefact locations §12 fixes. The fifth,
// hyper.yaml, lives at the repository root rather than in a directory.
var artefactDirs = []string{"definitions", "procedures", "targets", "providers"}

// ArtefactFiles returns every artefact file's path, relative to repoRoot with
// forward slashes, across the five artefact locations (§9). A directory that
// does not exist contributes nothing rather than an error — a repository
// author who has not yet written any Target declaration still gets a clean
// check. The walk is not recursive: an artefact directory holds files, not
// further directories (§12's kind:-to-directory mapping names no nesting).
func ArtefactFiles(repoRoot string) ([]string, error) {
	var files []string
	for _, dir := range artefactDirs {
		entries, err := os.ReadDir(filepath.Join(repoRoot, dir))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			if filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}
			files = append(files, dir+"/"+entry.Name())
		}
	}
	if info, err := os.Stat(filepath.Join(repoRoot, "hyper.yaml")); err == nil && !info.IsDir() {
		files = append(files, "hyper.yaml")
	}
	return files, nil
}
