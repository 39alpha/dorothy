package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	ipfs "github.com/ipfs/go-ipfs-api"
)

type Ipfs struct {
	*ipfs.Shell
}

func NewIpfs(config *IpfsConfig) (*Ipfs, error) {
	var shell *ipfs.Shell

	if config == nil {
		shell = ipfs.NewLocalShell()
	} else {
		shell = ipfs.NewShell(config.Url())
	}

	if shell != nil && shell.IsUp() {
		return &Ipfs{shell}, nil
	}
	return nil, fmt.Errorf("cannot connect to IPFS")
}

func NewLocalIpfs() (*Ipfs, error) {
	return NewIpfs(nil)
}

func (s Ipfs) CreateEmptyManifest() (*Manifest, error) {
	return s.SaveManifest(&Manifest{Versions: []*Version{}})
}

func (s Ipfs) SaveManifest(manifest *Manifest) (*Manifest, error) {
	buffer := new(bytes.Buffer)

	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(manifest); err != nil {
		return nil, err
	}

	hash, err := s.Add(buffer, ipfs.Pin(true))
	if err != nil {
		return nil, err
	}

	manifest.Hash = hash
	return manifest, nil
}

func (s Ipfs) GetManifest(hash string) (*Manifest, error) {
	r, err := s.Cat(hash)
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	manifest.Hash = hash

	return &manifest, err
}

func (s Ipfs) MergeAndCommit(old, new *Manifest) (*Manifest, error) {
	merged, _, err := old.Merge(new)
	if err != nil {
		return nil, err
	}

	delta, err := old.Diff(new)
	if err != nil {
		return nil, err
	}

	for _, version := range delta {
		s.CommitVersion(version)
	}

	return s.SaveManifest(merged)
}

func (s Ipfs) Commit(manifest *Manifest) (*Manifest, error) {
	return s.MergeAndCommit(&Manifest{}, manifest)
}

func (s Ipfs) Uncommit(manifest *Manifest, recursive bool) error {
	if recursive {
		for _, version := range manifest.Versions {
			s.RemoveVersion(version)
		}
	}

	return s.Unpin(manifest.Hash)
}

func (s Ipfs) CommitVersion(version *Version) (string, error) {
	return version.Hash, s.Pin(version.Hash)
}

func (s Ipfs) RemoveVersion(version *Version) error {
	return s.Unpin(version.Hash)
}
