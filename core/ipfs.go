package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/ipfs/boxo/files"
	"github.com/ipfs/boxo/path"
	"github.com/ipfs/kubo/client/rpc"
	"github.com/ipfs/kubo/core/coreiface/options"
)

type Ipfs struct {
	*rpc.HttpApi
}

func NewIpfs(config *IpfsConfig) (*Ipfs, error) {
	var api *rpc.HttpApi
	var err error

	if config == nil {
		api, err = rpc.NewLocalApi()
	} else {
		addr, err := config.Multiaddr()
		if err != nil {
			return nil, err
		}
		api, err = rpc.NewApi(addr)
	}

	if err != nil {
		return nil, err
	} else if api == nil {
		return nil, fmt.Errorf("cannot connect to IPFS")
	}
	return &Ipfs{api}, nil
}

func NewLocalIpfs() (*Ipfs, error) {
	return NewIpfs(nil)
}

func (s Ipfs) CreateEmptyManifest(ctx context.Context) (*Manifest, error) {
	return s.SaveManifest(ctx, &Manifest{Versions: []*Version{}})
}

func (s Ipfs) SaveManifest(ctx context.Context, manifest *Manifest) (*Manifest, error) {
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

func (s Ipfs) GetManifest(ctx context.Context, hash string) (*Manifest, error) {
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

func (s Ipfs) Get(ctx context.Context, hash, dest string) error {
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

func (s Ipfs) Add(ctx context.Context, filepath string, options ...options.UnixfsAddOption) (string, error) {
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
