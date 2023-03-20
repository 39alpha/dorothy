package cli

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/39alpha/dorthy/core"
)

func Log() error {
	manifest, err := core.ReadManifestFile(manifestpath)
	if err != nil {
		return err
	}

	for i := len(manifest) - 1; i >= 0; i-- {
		version := manifest[i]
		showParents := len(version.Parents) >= 1 && i != 0 && version.Parents[0] != manifest[i-1].Hash
		printCommit(manifest[i], showParents)
	}

	return nil
}

func printCommit(version core.Version, showParents bool) {
	s := strings.Builder{}
	t := tabwriter.NewWriter(&s, 0, 4, 2, ' ', 0)
	fmt.Fprintf(t, "%s\t%s\n", "Hash:", version.Hash)
	fmt.Fprintf(t, "%s\t%s\n", "Author:", version.Author)
	fmt.Fprintf(t, "%s\t%s\n", "Date:", version.Date.Format("Mon Jan 02 15:04:05 2006 -0700"))
	fmt.Fprintf(t, "%s\t%s\n", "Type:", core.PathTypeString(version.Type))
	if showParents {
		for i, parent := range version.Parents {
			if i == 0 {
				fmt.Fprintf(t, "%s\t%s\n", "Parents:", parent)
			} else {
				fmt.Fprintf(t, "%s\t%s\n", "", parent)
			}
		}
	}
	t.Flush()

	fmt.Printf("%s\n    %s\n\n", s.String(), version.Message)
}
