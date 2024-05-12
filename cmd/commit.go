package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"

	"github.com/39alpha/dorothy/core"
	"github.com/spf13/cobra"
)

func checkParentage(d *core.Dorothy, parents []string, pick bool) ([]string, bool, error) {
	if !d.Manifest.IsEmpty() && (pick || len(parents) == 0) {
		var picked []string
		for {
			var err error
			picked, err = d.ChooseVersions("Which versions are parents of this version?", false)
			if err != nil {
				return nil, false, err
			} else if len(picked) == 0 {
				fmt.Print("No parents selected. Do you want to continue (y/N) ")
				var res string
				fmt.Scanln(&res)
				if ok, err := regexp.MatchString("(?i)^y(es)?$", res); err == nil && ok {
					break
				}
			} else {
				break
			}
		}
		parents = append(parents, picked...)
	}

	unknown := d.UnknownCommits(parents)

	if len(unknown) != 0 {
		fmt.Println("The following parents are not in the manifest")
		for _, parent := range unknown {
			fmt.Printf("  %s\n", parent)
		}
		for {
			fmt.Print("Do you want to continue (y/N) ")
			var res string
			fmt.Scanln(&res)
			if ok, err := regexp.MatchString("(?i)^y(es)?$", res); err == nil && ok {
				return parents, true, nil
			} else if ok, err := regexp.MatchString("(?i)^n(o)?$", res); err == nil && ok {
				return nil, false, nil
			}
		}
	}

	sort.Strings(parents)

	return parents, true, nil
}

var commitCmd = &cobra.Command{
	Use:   "commit path",
	Short: "commit a dataset",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		message, err := cmd.Flags().GetString("message")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		nopin, err := cmd.Flags().GetBool("no-pin")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		parents, err := cmd.Flags().GetStringSlice("parents")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
		pick, err := cmd.Flags().GetBool("pick")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		d, err := core.NewDorothy()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}

		if message == "" {
			message, err = d.ReadFromEditor("commit-msg")
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v; aborting commit\n", err)
				os.Exit(1)
			}
		}

		parents, ok, err := checkParentage(d, parents, pick)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v; aborting commit\n", err)
			os.Exit(1)
		} else if !ok {
			os.Exit(0)
		}

		if err := d.Commit(args[0], message, nopin, parents); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringP("message", "m", "", "commit message")
	commitCmd.Flags().BoolP("no-pin", "n", false, "do not pin the data to your local node")
	commitCmd.Flags().StringSliceP("parents", "p", nil, "parents of this commit")
	commitCmd.Flags().BoolP("pick", "P", false, "interactively choose parents (implied by empty --partents)")
}
