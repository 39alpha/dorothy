package core

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/charmbracelet/lipgloss"
	ts "v.io/x/lib/toposort"
)

type Manifest struct {
	Versions []*Version `json:"versions"`
	Hash     string     `json:"-"`
}

type Version struct {
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	Message  string    `json:"message"`
	Hash     string    `json:"hash"`
	PathType PathType  `json:"path_type"`
	Parents  []string  `json:"parents"`
}

type PathType string

const (
	PathTypeDirectory PathType = "DIRECTORY"
	PathTypeFile      PathType = "FILE"
)

var AllPathType = []PathType{
	PathTypeDirectory,
	PathTypeFile,
}

func (e PathType) IsValid() bool {
	switch e {
	case PathTypeDirectory, PathTypeFile:
		return true
	}
	return false
}

func (e PathType) String() string {
	return string(e)
}

func (e *PathType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = PathType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid PathType", str)
	}
	return nil
}

func (e PathType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

func (v *Version) SameHash(o *Version) bool {
	return v.Hash == o.Hash
}

func (v *Version) SameParents(o *Version) bool {
	if len(v.Parents) != len(o.Parents) {
		return false
	}
	var these, those []string
	these = append(these, v.Parents...)
	those = append(these, o.Parents...)
	for i := range these {
		if these[i] != those[i] {
			return false
		}
	}
	return true
}

func (v *Version) Equal(o *Version) bool {
	return v.SameHash(o) &&
		v.Author == o.Author &&
		v.Date.Equal(o.Date) &&
		v.Message == o.Message &&
		v.PathType == o.PathType &&
		v.SameParents(o)
}

func (v *Version) Less(o *Version) bool {
	return !v.SameHash(o) && v.Date.Before(o.Date)
}

func ReadManifestFile(filename string) (*Manifest, error) {
	handle, err := os.OpenFile(filename, os.O_RDONLY|os.O_CREATE, 0755)
	defer handle.Close()
	if err != nil {
		return nil, err
	}

	return ReadManifest(handle)
}

func ReadManifest(r io.Reader) (*Manifest, error) {
	var manifest Manifest
	decoder := json.NewDecoder(r)
	return &manifest, decoder.Decode(&manifest)
}

func WriteManifestFile(filename string, manifest *Manifest) error {
	handle, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0755)
	defer handle.Close()
	if err != nil {
		return err
	}

	return WriteManifest(handle, manifest)
}

func WriteManifest(w io.Writer, manifest *Manifest) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(manifest)
}

func (old *Manifest) Diff(new *Manifest) ([]*Version, error) {
	if _, ok := old.Conflicts(new); !ok {
		return nil, fmt.Errorf("merge conflict")
	}

	var verisons []*Version
	for _, newverison := range new.Versions {
		found := false
		for _, oldverison := range old.Versions {
			if newverison.Equal(oldverison) {
				found = true
				break
			}
		}

		if !found {
			verisons = append(verisons, newverison)
		}
	}

	return verisons, nil
}

func (old *Manifest) Merge(new *Manifest) (*Manifest, []Conflict, error) {
	if conflicts, ok := old.Conflicts(new); !ok {
		return nil, conflicts, fmt.Errorf("merge conflict")
	}

	var updated []*Version

	for _, verison := range old.Versions {
		updated = append(updated, verison)
	}
	for _, newverison := range new.Versions {
		found := false
		for _, oldverison := range old.Versions {
			if newverison.Equal(oldverison) {
				found = true
				break
			}
		}
		if !found {
			updated = append(updated, newverison)
		}
	}

	sorted, err := toposort(updated)
	if err != nil {
		return nil, nil, err
	}

	return &Manifest{Versions: sorted}, nil, nil
}

type Conflict struct {
	Left  *Version
	Right *Version
}

func (old *Manifest) Conflicts(new *Manifest) ([]Conflict, bool) {
	var conflicts []Conflict
	for _, newverison := range new.Versions {
		for _, oldverison := range old.Versions {
			if newverison.SameHash(oldverison) && !newverison.Equal(oldverison) {
				conflicts = append(conflicts, Conflict{
					Left:  oldverison,
					Right: newverison,
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
	fmt.Fprintf(t, "    %s\t%s\t%s\n", bold("Type:"), c.Left.PathType.String(), c.Right.PathType.String())

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

func toposort(versions []*Version) ([]*Version, error) {
	sorter := &ts.Sorter{}

	for i := range versions {
		sorter.AddNode(&versions[i])
	}

	for i := range versions {
		for j := range versions {
			for _, hash := range versions[i].Parents {
				if versions[j].Hash == hash {
					sorter.AddEdge(&versions[j], &versions[i])
				}
			}
		}
	}

	for i := range versions {
		for j := range versions {
			if versions[j].Date.Before(versions[i].Date) {
				sorter.AddEdge(&versions[j], &versions[i])
			}
		}
	}

	sorted, cycles := sorter.Sort()
	if len(cycles) != 0 {
		return nil, fmt.Errorf("cycle(s) detected in manifest")
	}

	N := len(sorted)
	versions = make([]*Version, N)
	for i := N - 1; i >= 0; i-- {
		versions[N-1-i] = *sorted[i].(**Version)
	}

	return versions, nil
}
