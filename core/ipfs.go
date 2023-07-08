package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	ipfs "github.com/ipfs/go-ipfs-api"
)

const (
	FS_ROOT  string = "/dorothy"
	WEB_ROOT        = "/"
)

type DorthyPath struct {
	IpfsDir string   `json:"-"`
	WebDir  string   `json:"path"`
	Name    string   `json:"name"`
	Type    PathType `json:"-"`
}

func NewDorthyPath(ptype PathType, name string, parts ...string) DorthyPath {
	return DorthyPath{
		IpfsDir: filepath.Join(FS_ROOT, filepath.Join(parts...)),
		WebDir:  filepath.Join(WEB_ROOT, filepath.Join(parts...)),
		Name:    name,
		Type:    ptype,
	}
}

func (p DorthyPath) ToIpfsPath() string {
	return filepath.Join(p.IpfsDir, p.Name)
}

func (p DorthyPath) ToWebPath() string {
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

func (s Ipfs) CreateOrganization(ctx context.Context, name string) (DorthyPath, error) {
	path := NewDorthyPath(D_DIR, name)

	return path, s.FilesMkdir(ctx, path.ToIpfsPath(), func(r *ipfs.RequestBuilder) error {
		r.Option("parents", true)
		return nil
	})
}

func (s Ipfs) GetOrganizations(ctx context.Context) ([]DorthyPath, error) {
	path := NewDorthyPath(D_DIR, "")
	entries, err := s.FilesLs(ctx, path.ToIpfsPath())
	if err != nil {
		return nil, err
	}

	var paths []DorthyPath
	for _, entry := range entries {
		paths = append(paths, DorthyPath{
			IpfsDir: path.ToIpfsPath(),
			WebDir:  path.ToWebPath(),
			Name:    entry.Name,
			Type:    D_DIR,
		})
	}

	return paths, nil
}

func (s Ipfs) CreateRepository(ctx context.Context, organization, repo string) (DorthyPath, error) {
	path := NewDorthyPath(D_DIR, repo, organization)
	if err := s.FilesMkdir(ctx, path.ToIpfsPath()); err != nil {
		return path, err
	}

	_, err := s.CreateManifest(ctx, organization, repo)
	return path, err
}

func (s Ipfs) GetRepositories(ctx context.Context, organization string) ([]DorthyPath, error) {
	path := NewDorthyPath(D_DIR, organization)
	entries, err := s.FilesLs(ctx, path.ToIpfsPath())
	if err != nil {
		return nil, err
	}

	var paths []DorthyPath
	for _, entry := range entries {
		paths = append(paths, DorthyPath{
			IpfsDir: path.ToIpfsPath(),
			WebDir:  path.ToWebPath(),
			Name:    entry.Name,
			Type:    D_DIR,
		})
	}

	return paths, nil
}

func (s Ipfs) CreateManifest(ctx context.Context, organization, repo string) (DorthyPath, error) {
	return s.saveManifest(ctx, organization, repo, []Version{})
}

func (s Ipfs) saveManifest(ctx context.Context, organization, repo string, manifest Manifest) (DorthyPath, error) {
	path := NewDorthyPath(D_FILE, "manifest.json", organization, repo)

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

func (s Ipfs) GetManifest(ctx context.Context, organization, repo string) (Manifest, error) {
	path := NewDorthyPath(D_FILE, "manifest.json", organization, repo)
	r, err := s.FilesRead(ctx, path.ToIpfsPath())
	if err != nil {
		return nil, err
	}

	var manifest Manifest
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	return manifest, err
}

func (s Ipfs) Commit(ctx context.Context, organization, repo string, new Manifest) (Manifest, error) {
	old, err := s.GetManifest(ctx, organization, repo)
	if err != nil {
		return nil, err
	}

	delta, err := Diff(old, new)
	if err != nil {
		return nil, err
	}

	var errors []error
	var paths []DorthyPath
	for _, version := range delta {
		path, err := s.AddCommit(ctx, organization, repo, version)
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

	merged, _, err := Merge(old, new)
	if err != nil {
		return nil, err
	}

	_, err = s.saveManifest(ctx, organization, repo, merged)
	return merged, err
}

func (s Ipfs) AddCommit(ctx context.Context, organization, repo string, version Version) (DorthyPath, error) {
	path := NewDorthyPath(version.Type, version.Hash, organization, repo)

	ipfspath := "/ipfs/" + version.Hash

	if err := s.Pin(ipfspath); err != nil {
		return path, err
	}

	return path, s.FilesCp(ctx, "/ipfs/"+version.Hash, path.ToIpfsPath())
}

func (s Ipfs) RemovePath(ctx context.Context, path DorthyPath) error {
	stat, err := s.FilesStat(ctx, path.ToIpfsPath())
	if stat != nil && err == nil {
		if err = s.Unpin(stat.Hash); err != nil {
			return fmt.Errorf("failed to unpin file")
		}
	}

	return s.FilesRm(ctx, path.ToIpfsPath(), true)
}
