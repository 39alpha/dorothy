package cli

import (
	"fmt"

	"github.com/39alpha/dorthy/core"
)

func Log() error {
	manifest, err := core.ReadManifestFile(manifestpath)
	if err != nil {
		return err
	}

	for i := len(manifest) - 1; i >= 0; i-- {
		printCommit(manifest[i])
	}

	return nil
}

func printCommit(commit core.Commit) {
	fmt.Printf("hash:   %v\n", commit.Hash)
	fmt.Printf("Author: %v\n", commit.Author)
	fmt.Printf("Date:   %s\n\n", commit.Date.Format("Mon Jan 02 15:04:05 2006 -0700"))
	fmt.Printf("    %s\n\n", commit.Message)
}
