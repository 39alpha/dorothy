package core

import (
	"fmt"
)

func Log() error {
	manifest, err := ReadManifestFile(manifestpath)
	if err != nil {
		return err
	}

	for i := len(manifest) - 1; i >= 0; i-- {
		printCommit(manifest[i])
	}

	return nil
}

func printCommit(commit Commit) {
	fmt.Printf("hash:   %v\n", commit.Hash)
	fmt.Printf("Author: %v\n", commit.Author)
	fmt.Printf("Date:   %s\n\n", commit.Date.Format("Mon Jan 02 15:04:05 2006 -0700"))
	fmt.Printf("    %s\n\n", commit.Message)
}
