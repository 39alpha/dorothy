package core

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/39alpha/dorothy/core/model"
	ipfs "github.com/ipfs/go-ipfs-api"
)

type Ipfs struct {
	*ipfs.Shell
}

func NewIpfs(config *Config) (*Ipfs, error) {
	shell := ipfs.NewShell(config.Ipfs.Url())
	if shell != nil && shell.IsUp() {
		return &Ipfs{shell}, nil
	}
	return nil, fmt.Errorf("cannot connect to IPFS")
}

func (s Ipfs) CreateEmptyManifest() (*model.Manifest, error) {
	return s.SaveManifest(&model.Manifest{Versions: []*model.Version{}})
}

func (s Ipfs) SaveManifest(manifest *model.Manifest) (*model.Manifest, error) {
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

func (s Ipfs) GetManifest(hash string) (*model.Manifest, error) {
	r, err := s.Cat(hash)
	if err != nil {
		return nil, err
	}

	var manifest model.Manifest
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	manifest.Hash = hash

	return &manifest, err
}

func (s Ipfs) MergeAndCommit(old, new *model.Manifest) (*model.Manifest, error) {
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

func (s Ipfs) Commit(manifest *model.Manifest) (*model.Manifest, error) {
	return s.MergeAndCommit(&model.Manifest{}, manifest)
}

func (s Ipfs) Uncommit(manifest *model.Manifest, recursive bool) error {
	if recursive {
		for _, version := range manifest.Versions {
			s.RemoveVersion(version)
		}
	}

	return s.Unpin(manifest.Hash)
}

func (s Ipfs) CommitVersion(version *model.Version) (string, error) {
	return version.Hash, s.Pin(version.Hash)
}

func (s Ipfs) RemoveVersion(version *model.Version) error {
	return s.Unpin(version.Hash)
}
