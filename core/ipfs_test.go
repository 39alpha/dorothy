package core

import (
	"context"
	"os"
	"testing"
	"time"
)

func setup(t *testing.T) (*Ipfs, context.Context) {
	ctx, cancel := context.WithCancel(context.Background())

	working_dir, err := os.MkdirTemp(os.TempDir(), "dorothy-ipfs-")
	if err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		os.RemoveAll(working_dir)
	})

	client := NewIpfs(&IpfsConfig{
		Global: false,
	})
	if err := client.Initialize(working_dir); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}
	if err := client.Connect(ctx, working_dir, IpfsOffline); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}

	return client, ctx
}

func TestCreateEmptyManifest(t *testing.T) {
	client, ctx := setup(t)

	manifest, err := client.CreateEmptyManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if manifest.Hash == "" {
		t.Errorf("expected non-empty hash")
	}

	if len(manifest.Versions) != 0 {
		t.Errorf("expected empty manifest, got len(manifest) = %d", len(manifest.Versions))
	}
}

func TestGetManifest(t *testing.T) {
	client, ctx := setup(t)

	manifest, err := client.CreateEmptyManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}

	fetched, err := client.GetManifest(ctx, manifest.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if manifest.Hash != fetched.Hash {
		t.Errorf("expected hash %q, got %q", manifest.Hash, fetched.Hash)
	}

	if len(manifest.Versions) != 0 {
		t.Errorf("expected empty manifest, got len(manifest) = %d", len(manifest.Versions))
	}
}

func TestCommit(t *testing.T) {
	client, ctx := setup(t)

	hash := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"

	manifest := &Manifest{
		Versions: []*Version{
			{
				Author:   "Douglas G. Moore <doug@dglmoore.com>",
				Date:     time.Now(),
				Message:  "Aardvark Wikipedia Article",
				Hash:     hash,
				PathType: PathTypeDirectory,
				Parents:  nil,
			},
		},
	}

	returned, err := client.SaveManifest(ctx, manifest)
	if err != nil {
		t.Fatal(err)
	}

	if len(returned.Versions) != 1 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned.Versions))
	}

	for i, version := range manifest.Versions {
		if !version.Equal(returned.Versions[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, version, returned.Versions[0])
		}
	}

	saved, err := client.GetManifest(ctx, returned.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved.Versions) != 1 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(saved.Versions))
	}

	for i, version := range manifest.Versions {
		if !version.Equal(saved.Versions[i]) {
			t.Fatalf("expected saved[%d] = %v, got %v", i, version, saved.Versions[0])
		}
	}
}

func TestMultipleCommits(t *testing.T) {
	client, ctx := setup(t)

	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"

	manifest1 := &Manifest{
		Versions: []*Version{
			{
				Author:   "Douglas G. Moore <doug@dglmoore.com>",
				Date:     time.Now(),
				Message:  "Aardvark Wikipedia Article",
				Hash:     hash1,
				PathType: PathTypeFile,
				Parents:  nil,
			},
		},
	}

	manifest2 := &Manifest{
		Versions: append(manifest1.Versions, &Version{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time.Now(),
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: PathTypeFile,
			Parents:  nil,
		}),
	}

	_, err := client.SaveManifest(ctx, manifest1)
	if err != nil {
		t.Fatal(err)
	}

	returned, err := client.SaveManifest(ctx, manifest2)
	if err != nil {
		t.Fatal(err)
	}

	if len(returned.Versions) != 2 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned.Versions))
	}

	for i, version := range manifest2.Versions {
		if !version.Equal(returned.Versions[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, version, returned.Versions[0])
		}
	}

	saved, err := client.GetManifest(ctx, returned.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved.Versions) != 2 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(saved.Versions))
	}

	for i, version := range manifest2.Versions {
		if !version.Equal(saved.Versions[i]) {
			t.Fatalf("expected saved[%d] = %v, got %v", i, version, saved.Versions[0])
		}
	}
}
