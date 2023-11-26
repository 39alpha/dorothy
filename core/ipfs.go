package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/39alpha/dorothy/core/model"
	ipfs "github.com/ipfs/go-ipfs-api"
)

const (
	FS_ROOT  string = "/dorothy"
	WEB_ROOT        = "/"
)

type DorothyPath struct {
	IpfsDir string         `json:"-"`
	WebDir  string         `json:"path"`
	Name    string         `json:"name"`
	Type    model.PathType `json:"-"`
}

func NewDorothyPath(ptype model.PathType, name string, parts ...string) DorothyPath {
	return DorothyPath{
		IpfsDir: filepath.Join(FS_ROOT, filepath.Join(parts...)),
		WebDir:  filepath.Join(WEB_ROOT, filepath.Join(parts...)),
		Name:    name,
		Type:    ptype,
	}
}

func (p DorothyPath) ToIpfsPath() string {
	return filepath.Join(p.IpfsDir, p.Name)
}

func (p DorothyPath) ToWebPath() string {
	return filepath.Join(p.WebDir, p.Name)
}

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

func (s Ipfs) CreateOrganization(ctx context.Context, org *model.Organization) (DorothyPath, error) {
	path := NewDorothyPath(model.PathTypeDirectory, org.ID)

	return path, s.FilesMkdir(ctx, path.ToIpfsPath(), func(r *ipfs.RequestBuilder) error {
		r.Option("parents", true)
		return nil
	})
}

func (s Ipfs) CreateDataset(ctx context.Context, dataset *model.Dataset) (DorothyPath, error) {
	path := NewDorothyPath(model.PathTypeDirectory, dataset.ID, dataset.OrganizationID)
	if err := s.FilesMkdir(ctx, path.ToIpfsPath()); err != nil {
		return path, err
	}

	_, err := s.CreateManifest(ctx, dataset.OrganizationID, dataset.ID)
	return path, err
}

func (s Ipfs) CreateManifest(ctx context.Context, organization, dataset string) (DorothyPath, error) {
	manifest := model.Manifest{Versions: []*model.Version{}}
	return s.saveManifest(ctx, organization, dataset, &manifest)
}

func (s Ipfs) saveManifest(ctx context.Context, organization, dataset string, manifest *model.Manifest) (DorothyPath, error) {
	path := NewDorothyPath(model.PathTypeFile, "manifest.json", organization, dataset)

	buffer := new(bytes.Buffer)

	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(manifest); err != nil {
		return path, err
	}

	if err := s.RemovePath(ctx, path); err != nil {
		return path, err
	}

	hash, err := s.Add(buffer, ipfs.Pin(true))
	if err != nil {
		return path, err
	}

	return path, s.FilesCp(ctx, "/ipfs/"+hash, path.ToIpfsPath())
}

func (s Ipfs) GetDataset(ctx context.Context, organization, dataset string) (*DorothyPath, error) {
	path := NewDorothyPath(model.PathTypeDirectory, filepath.Join(organization, dataset))
	_, err := s.FilesStat(ctx, path.ToIpfsPath())
	if err != nil {
		return nil, err
	}
	path.Name = dataset
	return &path, err
}

func (s Ipfs) GetManifest(ctx context.Context, organization, dataset string) (*model.Manifest, error) {
	path := NewDorothyPath(model.PathTypeFile, "manifest.json", organization, dataset)
	r, err := s.FilesRead(ctx, path.ToIpfsPath())
	if err != nil {
		return nil, err
	}

	var manifest model.Manifest
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	return &manifest, err
}

func (s Ipfs) Commit(ctx context.Context, organization, dataset string, new *model.Manifest) (*model.Manifest, error) {
	old, err := s.GetManifest(ctx, organization, dataset)
	if err != nil {
		return nil, err
	}

	delta, err := old.Diff(new)
	if err != nil {
		return nil, err
	}

	var errors []error
	var paths []DorothyPath
	for _, version := range delta {
		path, err := s.AddCommit(ctx, organization, dataset, version)
		if err != nil {
			errors = append(errors, err)
			continue
		}
		paths = append(paths, path)
	}
	if len(errors) > 0 {
		for _, path := range paths {
			s.RemovePath(ctx, path)
		}
	}

	merged, _, err := old.Merge(new)
	if err != nil {
		return nil, err
	}

	_, err = s.saveManifest(ctx, organization, dataset, merged)
	return merged, err
}

func (s Ipfs) AddCommit(ctx context.Context, organization, dataset string, version *model.Version) (DorothyPath, error) {
	path := NewDorothyPath(version.PathType, version.Hash, organization, dataset)

	ipfspath := "/ipfs/" + version.Hash

	if err := s.Pin(ipfspath); err != nil {
		return path, err
	}

	return path, s.FilesCp(ctx, "/ipfs/"+version.Hash, path.ToIpfsPath())
}

func (s Ipfs) RemovePath(ctx context.Context, path DorothyPath) error {
	stat, err := s.FilesStat(ctx, path.ToIpfsPath())
	if stat != nil && err == nil {
		if err = s.Unpin(stat.Hash); err != nil {
			return fmt.Errorf("failed to unpin file")
		}
	}

	return s.FilesRm(ctx, path.ToIpfsPath(), true)
}
