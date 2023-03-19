package core

import (
	"context"
	"testing"
	"time"
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

func TestCreateOrganization(t *testing.T) {
	ctx := context.TODO()

	defer func() {
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	name := "team0"
	path, err := client.CreateOrganization(ctx, name)
	if err != nil {
		t.Fatal(err)
	}

	if path.IpfsDir != "/dorthy" {
		t.Errorf("expected name %q, got %q", "/dorthy", path.IpfsDir)
	}
	if path.WebDir != "/" {
		t.Errorf("expected %q file type, got %q", "/", path.WebDir)
	}
	if path.Name != name {
		t.Errorf("expected name %q, got %q", name, path.Name)
	}
	if path.Type != D_DIR {
		t.Errorf("expected %d file type, got %d", D_DIR, path.Type)
	}
}

func TestCreateRepositoryAlreadyExists(t *testing.T) {
	ctx := context.TODO()

	defer func() {
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	organization := "team0"
	name := "CSS0"
	_, err := client.CreateRepository(ctx, organization, name)
	if err == nil {
		t.Errorf("expected an error, got nil")
	}
}

func TestCreateRepository(t *testing.T) {
	ctx := context.TODO()

	organization := "team0"
	name := "CSS0"

	defer func() {
		err := client.RemovePath(ctx, NewDorthyPath(D_FILE, "manifest.json", organization, name))
		if err != nil {
			panic(err)
		}
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, organization); err != nil {
		t.Fatal(err)
	}

	path, err := client.CreateRepository(ctx, organization, name)
	if err != nil {
		t.Fatal(err)
	}

	ipfsDir := "/dorthy/" + organization
	webDir := "/" + organization

	if path.IpfsDir != ipfsDir {
		t.Errorf("expected name %q, got %q", ipfsDir, path.IpfsDir)
	}
	if path.WebDir != webDir {
		t.Errorf("expected %q file type, got %q", webDir, path.WebDir)
	}
	if path.Name != name {
		t.Errorf("expected name %q, got %q", name, path.Name)
	}
	if path.Type != D_DIR {
		t.Errorf("expected %d file type, got %d", D_DIR, path.Type)
	}
}

func TestCreateRepositoryCreatesEmptyManifest(t *testing.T) {
	ctx := context.TODO()

	organization := "team0"
	name := "CSS0"

	defer func() {
		client.RemovePath(ctx, NewDorthyPath(D_FILE, "manifest.json", organization, name))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, organization); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateRepository(ctx, organization, name); err != nil {
		t.Fatal(err)
	}

	manifest, err := client.GetManifest(ctx, organization, name)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifest) != 0 {
		t.Errorf("expected empty manifest, got len(manifest) = %d", len(manifest))
	}
}

func TestCommit(t *testing.T) {
	ctx := context.TODO()

	organization := "team0"
	name := "CSS0"
	hash := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"

	defer func() {
		client.RemovePath(ctx, NewDorthyPath(D_FILE, hash, organization, name))
		client.RemovePath(ctx, NewDorthyPath(D_FILE, "manifest.json", organization, name))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, organization); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateRepository(ctx, organization, name); err != nil {
		t.Fatal(err)
	}

	manifest := Manifest{
		Commit{
			Author:  "Douglas G. Moore <doug@dglmoore.com>",
			Date:    time.Now(),
			Message: "Aardvark Wikipedia Article",
			Hash:    hash,
			Type:    D_FILE,
			Parents: nil,
		},
	}

	returned, err := client.Commit(ctx, organization, name, manifest)
	if err != nil {
		t.Fatal(err)
	}

	if len(returned) != 1 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned))
	}

	for i, commit := range manifest {
		if !commit.Equal(returned[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, commit, returned[0])
		}
	}

	saved, err := client.GetManifest(ctx, organization, name)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved) != 1 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(saved))
	}

	for i, commit := range manifest {
		if !commit.Equal(saved[i]) {
			t.Fatalf("expected saved[%d] = %v, got %v", i, commit, saved[0])
		}
	}
}

func TestMultipleCommits(t *testing.T) {
	ctx := context.TODO()

	organization := "team0"
	name := "CSS0"
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"

	defer func() {
		client.RemovePath(ctx, NewDorthyPath(D_FILE, hash1, organization, name))
		client.RemovePath(ctx, NewDorthyPath(D_FILE, hash2, organization, name))
		client.RemovePath(ctx, NewDorthyPath(D_FILE, "manifest.json", organization, name))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, organization); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateRepository(ctx, organization, name); err != nil {
		t.Fatal(err)
	}

	manifest1 := Manifest{
		{
			Author:  "Douglas G. Moore <doug@dglmoore.com>",
			Date:    time.Now(),
			Message: "Aardvark Wikipedia Article",
			Hash:    hash1,
			Type:    D_FILE,
			Parents: nil,
		},
	}

	manifest2 := append(manifest1, Commit{
		Author:  "Douglas G. Moore <doug@dglmoore.com>",
		Date:    time.Now(),
		Message: "Africa Wikipedia Article",
		Hash:    hash2,
		Type:    D_FILE,
		Parents: nil,
	})

	if _, err := client.Commit(ctx, organization, name, manifest1); err != nil {
		t.Fatal(err)
	}

	returned, err := client.Commit(ctx, organization, name, manifest2)
	if err != nil {
		t.Fatal(err)
	}

	if len(returned) != 2 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned))
	}

	for i, commit := range manifest2 {
		if !commit.Equal(returned[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, commit, returned[0])
		}
	}

	saved, err := client.GetManifest(ctx, organization, name)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved) != 2 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(saved))
	}

	for i, commit := range manifest2 {
		if !commit.Equal(saved[i]) {
			t.Fatalf("expected saved[%d] = %v, got %v", i, commit, saved[0])
		}
	}
}

func TestConflictingCommits(t *testing.T) {
	ctx := context.TODO()

	organization := "team0"
	name := "CSS0"
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	defer func() {
		client.RemovePath(ctx, NewDorthyPath(D_FILE, hash1, organization, name))
		client.RemovePath(ctx, NewDorthyPath(D_FILE, hash2, organization, name))
		client.RemovePath(ctx, NewDorthyPath(D_FILE, hash3, organization, name))
		client.RemovePath(ctx, NewDorthyPath(D_FILE, "manifest.json", organization, name))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, organization); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateRepository(ctx, organization, name); err != nil {
		t.Fatal(err)
	}

	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")
	time2, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T11:00:00")
	time3, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T12:00:00")

	commits := Manifest{
		{
			Author:  "Douglas G. Moore <doug@dglmoore.com>",
			Date:    time1,
			Message: "Aardvark Wikipedia Article",
			Hash:    hash1,
			Type:    D_FILE,
			Parents: nil,
		},
		{
			Author:  "Douglas G. Moore <doug@dglmoore.com>",
			Date:    time2,
			Message: "Africa Wikipedia Article",
			Hash:    hash2,
			Type:    D_FILE,
			Parents: nil,
		},
		{
			Author:  "Douglas G. Moore <doug@dglmoore.com>",
			Date:    time3,
			Message: "Cold War Wikipedia Article",
			Hash:    hash3,
			Type:    D_FILE,
			Parents: nil,
		},
	}

	manifest1 := Manifest{commits[0], commits[2]}
	manifest2 := Manifest{commits[0], commits[1]}

	if _, err := client.Commit(ctx, organization, name, manifest1); err != nil {
		t.Fatal(err)
	}

	returned, err := client.Commit(ctx, organization, name, manifest2)
	if err != nil {
		t.Fatal(err)
	}

	if len(returned) != 3 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned))
	}

	for i, commit := range commits {
		if !commit.Equal(returned[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, commit, returned[0])
		}
	}

	saved, err := client.GetManifest(ctx, organization, name)
	if err != nil {
		t.Fatal(err)
	}

	if len(saved) != 3 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(saved))
	}

	for i, commit := range commits {
		if !commit.Equal(saved[i]) {
			t.Fatalf("expected saved[%d] = %v, got %v", i, commit, saved[0])
		}
	}
}
