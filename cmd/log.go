package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

func printCommit(version *core.Version, showParents bool) {
	s := strings.Builder{}
	t := tabwriter.NewWriter(&s, 0, 4, 2, ' ', 0)
	fmt.Fprintf(t, "%s\t%s\n", "Hash:", version.Hash)
	fmt.Fprintf(t, "%s\t%s\n", "Author:", version.Author)
	fmt.Fprintf(t, "%s\t%s\n", "Date:", version.Date.Format("Mon Jan 02 15:04:05 2006 -0700"))
	fmt.Fprintf(t, "%s\t%s\n", "Type:", version.PathType.String())
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

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "display the manifest",
	Run: func(cmd *cobra.Command, args []string) {
		d, initialized, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		} else if !initialized {
			fmt.Fprintf(os.Stderr, "not a dorothy repository\n")
			os.Exit(1)
		}

		var versions []*core.Version
		if d.Manifest != nil {
			versions = d.Manifest.Versions
		}
		if len(versions) == 0 {
			fmt.Printf("no versions")
		} else {
			for i := len(versions) - 1; i >= 0; i-- {
				version := versions[i]
				showParents := len(version.Parents) >= 1 && i != 0 && version.Parents[0] != versions[i-1].Hash
				printCommit(versions[i], showParents)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
