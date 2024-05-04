package core

import (
	"testing"
	"time"

	"github.com/39alpha/dorothy/core/model"
)

var client *Ipfs

func init() {
	config := &Config{
		Ipfs: IpfsConfig{
			Host: "127.0.0.1",
			Port: 5001,
		},
	}

	var err error
	client, err = NewIpfs(config)
	if err != nil {
		panic(err)
	}
}

func TestCreateEmptyManifest(t *testing.T) {
	manifest, err := client.CreateEmptyManifest()
	defer func() {
		if err := client.Uncommit(manifest, true); err != nil {
			t.Fatal(err)
		}
	}()

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
	manifest, err := client.CreateEmptyManifest()
	defer func() {
		if err := client.Uncommit(manifest, true); err != nil {
			t.Fatal(err)
		}
	}()

	if err != nil {
		t.Fatal(err)
	}

	fetched, err := client.GetManifest(manifest.Hash)
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
	hash := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"

	manifest := &model.Manifest{
		Versions: []*model.Version{
			{
				Author:   "Douglas G. Moore <doug@dglmoore.com>",
				Date:     time.Now(),
				Message:  "Aardvark Wikipedia Article",
				Hash:     hash,
				PathType: model.PathTypeDirectory,
				Parents:  nil,
			},
		},
	}

	returned, err := client.Commit(manifest)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Uncommit(returned, true); err != nil {
			t.Fatal(err)
		}
	}()

	if len(returned.Versions) != 1 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned.Versions))
	}

	for i, version := range manifest.Versions {
		if !version.Equal(returned.Versions[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, version, returned.Versions[0])
		}
	}

	saved, err := client.GetManifest(returned.Hash)
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
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"

	manifest1 := &model.Manifest{
		Versions: []*model.Version{
			{
				Author:   "Douglas G. Moore <doug@dglmoore.com>",
				Date:     time.Now(),
				Message:  "Aardvark Wikipedia Article",
				Hash:     hash1,
				PathType: model.PathTypeFile,
				Parents:  nil,
			},
		},
	}

	manifest2 := &model.Manifest{
		Versions: append(manifest1.Versions, &model.Version{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time.Now(),
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: model.PathTypeFile,
			Parents:  nil,
		}),
	}

	first, err := client.Commit(manifest1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Uncommit(first, true); err != nil {
			t.Fatal(err)
		}
	}()

	returned, err := client.Commit(manifest2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Uncommit(returned, true); err != nil {
			t.Fatal(err)
		}
	}()

	if len(returned.Versions) != 2 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned.Versions))
	}

	for i, version := range manifest2.Versions {
		if !version.Equal(returned.Versions[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, version, returned.Versions[0])
		}
	}

	saved, err := client.GetManifest(returned.Hash)
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

func TestConflictingCommits(t *testing.T) {
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")
	time2, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T11:00:00")
	time3, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T12:00:00")

	versions := []*model.Version{
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Aardvark Wikipedia Article",
			Hash:     hash1,
			PathType: model.PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time2,
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: model.PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time3,
			Message:  "Cold War Wikipedia Article",
			Hash:     hash3,
			PathType: model.PathTypeFile,
			Parents:  nil,
		},
	}

	manifest1 := &model.Manifest{Versions: []*model.Version{versions[0], versions[2]}}
	manifest2 := &model.Manifest{Versions: []*model.Version{versions[0], versions[1]}}

	first, err := client.Commit(manifest1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Uncommit(first, true); err != nil {
			t.Fatal(err)
		}
	}()

	returned, err := client.MergeAndCommit(manifest1, manifest2)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := client.Uncommit(returned, true); err != nil {
			t.Fatal(err)
		}
	}()

	if len(returned.Versions) != 3 {
		t.Fatalf("expected %d entries in manifest, got %d", 3, len(returned.Versions))
	}

	for i, version := range versions {
		if !version.Equal(returned.Versions[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, version, returned.Versions[0])
		}
	}

	saved, err := client.GetManifest(returned.Hash)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved.Versions) != 3 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(saved.Versions))
	}

	for i, version := range versions {
		if !version.Equal(saved.Versions[i]) {
			t.Fatalf("expected saved.Versions[%d] = %v, got %v", i, version, saved.Versions[0])
		}
	}
}
