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
			selected := d.Manifest.LeafVersions()
			if len(selected) > 1 {
				selected = nil
			}
			picked, err = d.ChooseVersionsWithSelected("Which versions are parents of this version?", false, selected)
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
	Args:  cobra.MinimumNArgs(1),
	Run: HandleErrors(func(cmd *cobra.Command, args []string) error {
		configpath, err := cmd.Flags().GetString("config")
		if err != nil {
			return err
		}
		noinherit, err := cmd.Flags().GetBool("noinherit")
		if err != nil {
			return err
		}
		message, err := cmd.Flags().GetString("message")
		if err != nil {
			return err
		}
		nopin, err := cmd.Flags().GetBool("no-pin")
		if err != nil {
			return err
		}
		parents, err := cmd.Flags().GetStringSlice("parents")
		if err != nil {
			return err
		}
		pick, err := cmd.Flags().GetBool("pick")
		if err != nil {
			return err
		}

		dorothy, err := core.NewDorothy()
		if err != nil {
			return err
		}

		if noinherit {
			if err := dorothy.ResetConfig(); err != nil {
				return err
			}
		}
		if configpath != "" {
			if err := dorothy.LoadConfigFile(configpath); err != nil {
				return err
			}
		}

		if err := dorothy.Setup(core.IpfsOffline); err != nil {
			return err
		}

		parents, ok, err := checkParentage(dorothy, parents, pick)
		if err != nil {
			return fmt.Errorf("%v; aborting commit\n", err)
		} else if !ok {
			return nil
		}

		if message == "" {
			message, err = dorothy.ReadFromEditor("commit-msg")
			if err != nil {
				return fmt.Errorf("%v; aborting commit\n", err)
			}
		}

		conflicts, err := dorothy.Commit(args, message, nopin, parents)
		if len(conflicts) != 0 {
			fmt.Fprintf(os.Stderr, "conflicts:\n")
			for _, conflict := range conflicts {
				fmt.Fprintf(os.Stderr, "  %s", conflict)
			}
		}
		return err
	}),
}

func init() {
	rootCmd.AddCommand(commitCmd)
	commitCmd.Flags().StringP("message", "m", "", "commit message")
	commitCmd.Flags().BoolP("no-pin", "N", false, "do not pin the data to your local node")
	commitCmd.Flags().StringSliceP("parents", "p", nil, "parents of this commit")
	commitCmd.Flags().BoolP("pick", "P", false, "interactively choose parents (implied by empty --partents)")
}
