package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"
)

type PathType = uint8

const (
	D_DIR  PathType = 0
	D_FILE          = 1
)

type Manifest = []Commit

type Commit struct {
	Author    string    `json:"author"`
	Date      time.Time `json:"date"`
	Message   string    `json:"message"`
	Hash      string    `json:"hash"`
	Type      PathType  `json:"path_type"`
	Ancestors []string  `json:"ancestors"`
}

func (c Commit) SameHash(o Commit) bool {
	return c.Hash == o.Hash
}

func (c Commit) Equal(o Commit) bool {
	return c.SameHash(o) &&
		c.Author == o.Author &&
		c.Date.Equal(o.Date) &&
		c.Message == o.Message &&
		c.Type == o.Type
}

func (c Commit) Less(o Commit) bool {
	return !c.SameHash(o) && c.Date.Before(o.Date)
}

func ReadManifestFile(filename string) (Manifest, error) {
	handle, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0755)
	defer handle.Close()
	if err != nil {
		return nil, err
	}

	return ReadManifest(handle)
}

func ReadManifest(r io.Reader) (Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(r)
	return manifest, decoder.Decode(&manifest)
}

func WriteManifestFile(filename string, manifest Manifest) error {
	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0755)
	defer handle.Close()
	if err != nil {
		return err
	}

	return WriteManifest(handle, manifest)
}

func WriteManifest(w io.Writer, manifest Manifest) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(manifest)
}

func Diff(old, new Manifest) ([]Commit, error) {
	if _, ok := Conflicts(old, new); !ok {
		return nil, fmt.Errorf("merge conflict")
	}

	var commits []Commit
	for _, newcommit := range new {
		found := false
		for _, oldcommit := range old {
			if newcommit.Equal(oldcommit) {
				found = true
				break
			}
		}

		if !found {
			commits = append(commits, newcommit)
		}
	}

	return commits, nil
}

func Merge(old, new Manifest) (Manifest, []Conflict, error) {
	if conflicts, ok := Conflicts(old, new); !ok {
		return nil, conflicts, fmt.Errorf("merge conflict")
	}

	var updated Manifest

	for _, commit := range old {
		updated = append(updated, commit)
	}
	for _, newcommit := range new {
		found := false
		for _, oldcommit := range old {
			if newcommit.Equal(oldcommit) {
				found = true
				break
			}
		}
		if !found {
			updated = append(updated, newcommit)
		}
	}

	// TODO: For now we will simply sort by date, but ideally we would do a
	//       deterministic topological sort based on ancestry
	sort.SliceStable(updated, func(i, j int) bool { return updated[i].Less(updated[j]) })

	return updated, nil, nil
}

type Conflict struct {
	Left  Commit
	Right Commit
}

func Conflicts(old, new Manifest) ([]Conflict, bool) {
	var conflicts []Conflict
	for _, newcommit := range new {
		for _, oldcommit := range old {
			if newcommit.SameHash(oldcommit) && !newcommit.Equal(oldcommit) {
				conflicts = append(conflicts, Conflict{
					Left:  oldcommit,
					Right: newcommit,
				})
			}
		}
	}
	return conflicts, len(conflicts) == 0
}
