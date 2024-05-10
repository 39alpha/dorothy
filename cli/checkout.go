package cli

import (
	"fmt"
	"strings"

	"github.com/39alpha/dorothy/core"
	ipfs "github.com/ipfs/go-ipfs-api"
)

func Checkout(hash, dest string) error {
	manifest, err := core.ReadManifestFile(MANIFEST_PATH)
	if err != nil {
		return fmt.Errorf("failed to read manifest")
	}

	for _, version := range manifest.Versions {
		if version.Hash == hash {
			return checkout(hash, dest)
		}
	}

	var matches []*core.Version
	for _, version := range manifest.Versions {
		if strings.HasPrefix(version.Hash, hash) {
			matches = append(matches, version)
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
