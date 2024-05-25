package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/ipfs/kubo/core/coreapi"
	"github.com/ipfs/kubo/core/coreiface/options"
	"github.com/ipfs/kubo/core/node/libp2p"
	"github.com/ipfs/kubo/plugin/loader"
	"github.com/ipfs/kubo/repo/fsrepo"

	kuboconfig "github.com/ipfs/kubo/config"
	kubo "github.com/ipfs/kubo/core"
	icore "github.com/ipfs/kubo/core/coreiface"
)

var loadPluginsOnce sync.Once

type Ipfs struct {
	icore.CoreAPI
	node   *kubo.IpfsNode
	config IpfsConfig
}

func NewIpfs(config *IpfsConfig) *Ipfs {
	if config == nil {
		return &Ipfs{
			config: IpfsConfig{
				Global: false,
			},
		}
	}

	return &Ipfs{nil, nil, *config}
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
	if !s.config.Global {
		return s.connectLocal(ctx, dir, options...)
	} else {
		return s.connectGlobal()
	}
}

func (s *Ipfs) connectGlobal() error {
	var err error

	if s.config.Host == "" && s.config.Port == 0 {
		s.CoreAPI, err = rpc.NewLocalApi()
	} else {
		addr, err := s.config.Multiaddr()
		if err != nil {
			return err
		}
		s.CoreAPI, err = rpc.NewApi(addr)
	}

	if err != nil {
		return err
	} else if s.CoreAPI == nil {
		return fmt.Errorf("cannot connect to global IPFS instance")
	}

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

func Online(online bool) IpfsNodeOption {
	return func(cfg *kubo.BuildCfg) *kubo.BuildCfg {
		cfg.Online = online
		return cfg
	}
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

	return nil
}

func (s *Ipfs) CreateEmptyManifest(ctx context.Context) (*Manifest, error) {
	return s.SaveManifest(ctx, &Manifest{Versions: []*Version{}})
}

func (s *Ipfs) SaveManifest(ctx context.Context, manifest *Manifest) (*Manifest, error) {
	buffer := new(bytes.Buffer)

	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(manifest); err != nil {
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
	manifestPath, err := path.NewPath(hash)

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

func (s *Ipfs) Add(ctx context.Context, filepath string, options ...options.UnixfsAddOption) (string, error) {
	filenode, err := getUnixFileNode(filepath)
	if err != nil {
		return "", err
	}

	cidfile, err := s.Unixfs().Add(ctx, filenode, options...)

	if err != nil {
		return "", err
	}

	return cidfile.RootCid().String(), nil
}
