package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"
	"github.com/libp2p/go-libp2p/core/peer"

	kuboconfig "github.com/ipfs/kubo/config"
	kubo "github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
)

var loadPluginsOnce sync.Once

type Ipfs struct {
	icore.CoreAPI
	Identity peer.ID
	node     *kubo.IpfsNode
	config   IpfsConfig
	options  []IpfsNodeOption
}

func NewIpfs(config *IpfsConfig) Ipfs {
	if config == nil {
		return Ipfs{
			config: IpfsConfig{
				Global: false,
			},
		}
	}

	return Ipfs{
		config: *config,
	}
}

func (s *Ipfs) IsConnected() bool {
	return s != nil && s.CoreAPI != nil
}

func (s *Ipfs) Initialize(dir string) error {
	if !s.config.Global {
		return s.initializeLocal(dir)
	}
	return nil
}

func (s *Ipfs) initializeLocal(dir string) error {
	if err := setupPlugins(); err != nil {
		return err
	}

	cfg, err := kuboconfig.Init(io.Discard, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate default IPFS config: %v", err)
	}

	if err := fsrepo.Init(dir, cfg); err != nil {
		return fmt.Errorf("failed to initialize IPFS repo: %v", err)
	}

	return nil
}

func (s *Ipfs) Connect(ctx context.Context, dir string, options ...IpfsNodeOption) error {
	s.options = options
	if !s.config.Global {
		return s.connectLocal(ctx, dir, options...)
	} else {
		return s.connectGlobal(ctx)
	}
}

func (s *Ipfs) connectGlobal(ctx context.Context) error {
	var err error

	var api *rpc.HttpApi
	if s.config.Host == "" && s.config.Port == 0 {
		api, err = rpc.NewLocalApi()
	} else {
		api, err = rpc.NewURLApiWithClient(s.config.Url(), &http.Client{
			Transport: &http.Transport{
				Proxy:             http.ProxyFromEnvironment,
				DisableKeepAlives: true,
			},
		})
	}

	if err != nil {
		return err
	} else if api == nil {
		return fmt.Errorf("cannot connect to global IPFS instance")
	}

	var resp struct {
		Identity peer.ID `json:"ID"`
	}
	builder := api.Request("id")
	if err := builder.Exec(ctx, &resp); err != nil {
		return fmt.Errorf("cannot get peer id for global node")
	}

	s.CoreAPI = api
	s.Identity = resp.Identity

	return nil
}

func setupPlugins() error {
	var err error

	loadPluginsOnce.Do(func() {
		plugins, err := loader.NewPluginLoader("plugins")
		if err != nil {
			err = fmt.Errorf("error loading IPFS plugins: %v", err)
			return
		}

		if err := plugins.Initialize(); err != nil {
			err = fmt.Errorf("error initializing IPFS plugins: %v", err)
			return
		}

		if err := plugins.Inject(); err != nil {
			err = fmt.Errorf("error injecting IPFS plugins: %v", err)
			return
		}
	})

	return err
}

type IpfsNodeOption func(*kubo.BuildCfg) *kubo.BuildCfg

func IpfsOnline(cfg *kubo.BuildCfg) *kubo.BuildCfg {
	cfg.Online = true
	return cfg
}

func IpfsOffline(cfg *kubo.BuildCfg) *kubo.BuildCfg {
	cfg.Online = false
	return cfg
}

func createNode(ctx context.Context, dir string, options ...IpfsNodeOption) (*kubo.IpfsNode, error) {
	repo, err := fsrepo.Open(dir)
	if err != nil {
		return nil, err
	}

	cfg := &kubo.BuildCfg{
		Online:  true,
		Routing: libp2p.DHTOption,
		Repo:    repo,
	}
	for _, option := range options {
		cfg = option(cfg)
	}

	return kubo.NewNode(ctx, cfg)
}

func (s *Ipfs) connectLocal(ctx context.Context, filepath string, options ...IpfsNodeOption) error {
	err := setupPlugins()
	if err != nil {
		return err
	}

	s.node, err = createNode(ctx, filepath, options...)
	if err != nil {
		return fmt.Errorf("failed to instantiate local IPFS node: %v", err)
	}

	s.CoreAPI, err = coreapi.NewCoreAPI(s.node)
	if err != nil {
		return fmt.Errorf("failed to connect to local IPFS API")
	}

	s.Identity = s.node.Identity

	return nil
}

func (s *Ipfs) CreateEmptyManifest(ctx context.Context) (*Manifest, error) {
	return s.SaveManifest(ctx, &Manifest{Versions: []*Version{}})
}

func (s *Ipfs) SaveManifest(ctx context.Context, manifest *Manifest) (*Manifest, error) {
	buffer := new(bytes.Buffer)

	if err := manifest.Encode(buffer); err != nil {
		return nil, err
	}

	path, err := s.Unixfs().Add(
		ctx,
		files.NewReaderFile(buffer),
		options.Unixfs.Pin(true),
		options.Unixfs.Progress(true),
	)
	if err != nil {
		return nil, err
	}

	manifest.Hash = path.RootCid().String()
	return manifest, nil
}

func (s *Ipfs) GetManifest(ctx context.Context, hash string) (*Manifest, error) {
	manifestPath, err := path.NewPath("/ipfs/" + hash)
	if err != nil {
		return nil, err
	}

	fileNode, err := s.Unixfs().Get(ctx, manifestPath)
	if err != nil {
		return nil, err
	}

	file := files.ToFile(fileNode)
	if file == nil {
		return nil, fmt.Errorf("the node directed to by the manifest hash is not a file")
	}

	var manifest Manifest
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	manifest.Hash = hash
	return &manifest, err
}

func (s *Ipfs) Get(ctx context.Context, hash, dest string) error {
	dataPath, err := path.NewPath("/ipfs/" + hash)
	if err != nil {
		return err
	}

	file, err := s.Unixfs().Get(ctx, dataPath)
	if err != nil {
		return err
	}

	return files.WriteTo(file, dest)
}

func getUnixFileNode(filepath string) (files.Node, error) {
	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	return files.NewSerialFile(filepath, true, stat)
}

func (s *Ipfs) Add(ctx context.Context, filename string, options ...options.UnixfsAddOption) (string, error) {
	filenode, err := getUnixFileNode(filename)
	if err != nil {
		return "", err
	}

	cidfile, err := s.Unixfs().Add(ctx, filenode, options...)

	if err != nil {
		return "", err
	}

	return cidfile.RootCid().String(), nil
}

func (s Ipfs) AddMany(ctx context.Context, filenames []string, options ...options.UnixfsAddOption) (string, error) {
	if len(filenames) == 0 {
		return "", fmt.Errorf("no files provided")
	}

	var entries []files.DirEntry
	for _, path := range filenames {
		node, err := getUnixFileNode(path)
		if err != nil {
			return "", err
		}

		entries = append(entries, files.FileEntry(filepath.Base(path), node))
	}

	dir := files.NewSliceDirectory(entries)

	cidfile, err := s.Unixfs().Add(ctx, dir, options...)

	if err != nil {
		return "", err
	}

	return cidfile.RootCid().String(), nil
}

func (s *Ipfs) ConnectToPeerById(ctx context.Context, id peer.ID) error {
	addrInfo, err := s.Routing().FindPeer(ctx, id)
	if err != nil {
		return err
	}
	return s.Swarm().Connect(ctx, addrInfo)
}

func (s Ipfs) MergeAndCommit(ctx context.Context, old, new *Manifest) (*Manifest, []Conflict, error) {
	merged, conflicts, err := old.Merge(new)
	if err != nil || len(conflicts) != 0 {
		return nil, conflicts, err
	}

	delta, err := old.Diff(new)
	if err != nil {
		return nil, nil, err
	}

	errs := make([]error, 0, len(delta))
	for _, version := range delta {
		_, err := s.CommitVersion(ctx, version)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) != 0 {
		return nil, nil, errors.Join(errs...)
	}

	manifest, err := s.SaveManifest(ctx, merged)
	return manifest, nil, err
}

func (s Ipfs) Commit(ctx context.Context, manifest *Manifest) (*Manifest, error) {
	merged, _, err := s.MergeAndCommit(ctx, &Manifest{}, manifest)
	return merged, err
}

func (s Ipfs) CommitVersion(ctx context.Context, version *Version) (string, error) {
	versionPath, err := path.NewPath("/ipfs/" + version.Hash)
	if err != nil {
		return "", err
	}
	return version.Hash, s.Pin().Add(ctx, versionPath)
}
