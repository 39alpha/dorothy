package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"
	ts "v.io/x/lib/toposort"
)

type PathType = uint8

const (
	D_DIR  PathType = 0
	D_FILE          = 1
)

func PathTypeString(p PathType) string {
	switch p {
	case D_DIR:
		return "directory"
	case D_FILE:
		return "file"
	default:
		return "<unknown>"
	}
}

type Manifest = []Commit

type Commit struct {
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
	Hash    string    `json:"hash"`
	Type    PathType  `json:"path_type"`
	Parents []string  `json:"parents"`
}

func (c Commit) SameHash(o Commit) bool {
	return c.Hash == o.Hash
}

func (c Commit) SameParents(o Commit) bool {
	if len(c.Parents) != len(o.Parents) {
		return false
	}
	var these, those []string
	these = append(these, c.Parents...)
	those = append(these, o.Parents...)
	for i := range these {
		if these[i] != those[i] {
			return false
		}
	}
	return true
}

func (c Commit) Equal(o Commit) bool {
	return c.SameHash(o) &&
		c.Author == o.Author &&
		c.Date.Equal(o.Date) &&
		c.Message == o.Message &&
		c.Type == o.Type &&
		c.SameParents(c)
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

	sorted, err := toposort(updated)
	if err != nil {
		return nil, nil, err
	}

	return sorted, nil, nil
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

func (c Conflict) String() string {
	bold := lipgloss.NewStyle().Bold(true).Render

	s := strings.Builder{}
	t := tabwriter.NewWriter(&s, 0, 4, 1, ' ', 0)
	fmt.Fprintf(t, "    %s\t%s\t%s\n", "", bold("original"), bold("new"))
	fmt.Fprintf(t, "    %s\t%s\t%s\n", bold("Hash:"), c.Left.Hash, c.Right.Hash)
	fmt.Fprintf(t, "    %s\t%s\t%s\n", bold("Author:"), c.Left.Author, c.Right.Author)
	fmt.Fprintf(
		t,
		"    %s\t%s\t%s\n",
		bold("Date:"),
		c.Left.Date.Format("Mon Jan 02 15:04:05 2006 -0700"),
		c.Right.Date.Format("Mon Jan 02 15:04:05 2006 -0700"),
	)
	fmt.Fprintf(t, "    %s\t%s\t%s\n", bold("Message:"), c.Left.Message, c.Right.Message)
	fmt.Fprintf(t, "    %s\t%s\t%s\n", bold("Type:"), PathTypeString(c.Left.Type), PathTypeString(c.Right.Type))

	label := "Parents:"
	if len(c.Left.Parents) == 0 && len(c.Right.Parents) == 0 {
		fmt.Fprintf(t, "    %s\t%s\t%s\n", bold(label), "", "")
	} else {
		i, j := 0, 0
		for i < len(c.Left.Parents) || j < len(c.Right.Parents) {
			oldparent := ""
			if i < len(c.Left.Parents) {
				oldparent = c.Left.Parents[i]
			}
			newparent := ""
			if j < len(c.Right.Parents) {
				newparent = c.Right.Parents[j]
			}

			if i != 0 || j != 0 {
				label = ""
			}
			fmt.Fprintf(t, "    %s\t%s\t%s\n", bold(label), oldparent, newparent)

			i++
			j++
		}
	}

	t.Flush()
	return s.String()
}

func toposort(manifest Manifest) (Manifest, error) {
	sorter := &ts.Sorter{}

	for i := range manifest {
		sorter.AddNode(&manifest[i])
	}

	for i := range manifest {
		for j := range manifest {
			for _, hash := range manifest[i].Parents {
				if manifest[j].Hash == hash {
					sorter.AddEdge(&manifest[j], &manifest[i])
				}
			}
		}
	}

	for i := range manifest {
		for j := range manifest {
			if manifest[j].Date.Before(manifest[i].Date) {
				sorter.AddEdge(&manifest[j], &manifest[i])
			}
		}
	}

	sorted, cycles := sorter.Sort()
	if len(cycles) != 0 {
		return nil, fmt.Errorf("cycle(s) detected in manifest")
	}

	var newmanifest []Commit
	for i := len(sorted) - 1; i >= 0; i-- {
		newmanifest = append(newmanifest, *sorted[i].(*Commit))
	}

	return newmanifest, nil
}
