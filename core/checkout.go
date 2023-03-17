package core

import (
	"fmt"
	"strings"

	ipfs "github.com/ipfs/go-ipfs-api"
)

func Checkout(hash, dest string) error {
	manifest, err := ReadManifestFile(manifestpath)
	if err != nil {
		return fmt.Errorf("failed to read manifest")
	}

	for _, entry := range manifest {
		if entry.Hash == hash {
			return checkout(hash, dest)
		}
	}

	var matches []Commit
	for _, entry := range manifest {
		if strings.HasPrefix(entry.Hash, hash) {
			matches = append(matches, entry)
		}
	}

	if len(matches) == 0 {
		return fmt.Errorf("hash %q not found in manifest", hash)
	} else if len(matches) == 1 {
		return checkout(matches[0].Hash, dest)
	} else {
		return fmt.Errorf("hash matches multiple commits; aborting")
	}
}

func checkout(hash, dest string) error {
	client := ipfs.NewLocalShell()
	return client.Get(hash, dest)
}
