package core

import (
	"context"
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

func TestCreateOrganization(t *testing.T) {
	ctx := context.TODO()

	defer func() {
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	id := "team0"
	path, err := client.CreateOrganization(ctx, &model.Organization{ID: id})
	if err != nil {
		t.Fatal(err)
	}

	if path.IpfsDir != "/dorothy" {
		t.Errorf("expected name %q, got %q", "/dorothy", path.IpfsDir)
	}
	if path.WebDir != "/" {
		t.Errorf("expected %q file type, got %q", "/", path.WebDir)
	}
	if path.Name != id {
		t.Errorf("expected name %q, got %q", id, path.Name)
	}
	if path.Type != model.PathTypeDirectory {
		t.Errorf("expected %q file type, got %q", model.PathTypeDirectory, path.Type)
	}
}

func TestCreateDatasetAlreadyExists(t *testing.T) {
	ctx := context.TODO()

	defer func() {
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	organizationId := "team0"
	id := "CSS0"
	_, err := client.CreateDataset(ctx, &model.Dataset{ID: id, OrganizationID: organizationId})
	if err == nil {
		t.Errorf("expected an error, got nil")
	}
}

func TestCreateDataset(t *testing.T) {
	ctx := context.TODO()

	organizationId := "team0"
	id := "CSS0"

	defer func() {
		err := client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, "manifest.json", organizationId, id))
		if err != nil {
			panic(err)
		}
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, &model.Organization{ID: organizationId}); err != nil {
		t.Fatal(err)
	}

	path, err := client.CreateDataset(ctx, &model.Dataset{ID: id, OrganizationID: organizationId})
	if err != nil {
		t.Fatal(err)
	}

	ipfsDir := "/dorothy/" + organizationId
	webDir := "/" + organizationId

	if path.IpfsDir != ipfsDir {
		t.Errorf("expected name %q, got %q", ipfsDir, path.IpfsDir)
	}
	if path.WebDir != webDir {
		t.Errorf("expected %q file type, got %q", webDir, path.WebDir)
	}
	if path.Name != id {
		t.Errorf("expected name %q, got %q", id, path.Name)
	}
	if path.Type != model.PathTypeDirectory {
		t.Errorf("expected %q file type, got %q", model.PathTypeDirectory, path.Type)
	}
}

func TestCreateDatasetCreatesEmptyManifest(t *testing.T) {
	ctx := context.TODO()

	organizationId := "team0"
	id := "CSS0"

	defer func() {
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, "manifest.json", organizationId, id))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, &model.Organization{ID: organizationId}); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateDataset(ctx, &model.Dataset{ID: id, OrganizationID: organizationId}); err != nil {
		t.Fatal(err)
	}

	manifest, err := client.GetManifest(ctx, organizationId, id)
	if err != nil {
		t.Fatal(err)
	}

	if len(manifest.Versions) != 0 {
		t.Errorf("expected empty manifest, got len(manifest) = %d", len(manifest.Versions))
	}
}

func TestCommit(t *testing.T) {
	ctx := context.TODO()

	organizationId := "team0"
	id := "CSS0"
	hash := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"

	defer func() {
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, hash, organizationId, id))
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, "manifest.json", organizationId, id))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, &model.Organization{ID: organizationId}); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateDataset(ctx, &model.Dataset{ID: id, OrganizationID: organizationId}); err != nil {
		t.Fatal(err)
	}

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

	returned, err := client.Commit(ctx, organizationId, id, manifest)
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

	saved, err := client.GetManifest(ctx, organizationId, id)
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
	ctx := context.TODO()

	organizationId := "team0"
	id := "CSS0"
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"

	defer func() {
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, hash1, organizationId, id))
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, hash2, organizationId, id))
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, "manifest.json", organizationId, id))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, &model.Organization{ID: organizationId}); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateDataset(ctx, &model.Dataset{ID: id, OrganizationID: organizationId}); err != nil {
		t.Fatal(err)
	}

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

	if _, err := client.Commit(ctx, organizationId, id, manifest1); err != nil {
		t.Fatal(err)
	}

	returned, err := client.Commit(ctx, organizationId, id, manifest2)
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

	saved, err := client.GetManifest(ctx, organizationId, id)
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
	ctx := context.TODO()

	organizationId := "team0"
	id := "CSS0"
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	defer func() {
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, hash1, organizationId, id))
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, hash2, organizationId, id))
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, hash3, organizationId, id))
		client.RemovePath(ctx, NewDorothyPath(model.PathTypeFile, "manifest.json", organizationId, id))
		client.FilesRm(ctx, FS_ROOT, true)
	}()

	if _, err := client.CreateOrganization(ctx, &model.Organization{ID: organizationId}); err != nil {
		t.Fatal(err)
	}

	if _, err := client.CreateDataset(ctx, &model.Dataset{ID: id, OrganizationID: organizationId}); err != nil {
		t.Fatal(err)
	}

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

	if _, err := client.Commit(ctx, organizationId, id, manifest1); err != nil {
		t.Fatal(err)
	}

	returned, err := client.Commit(ctx, organizationId, id, manifest2)
	if err != nil {
		t.Fatal(err)
	}

	if len(returned.Versions) != 3 {
		t.Fatalf("expected %d entries in manifest, got %d", 1, len(returned.Versions))
	}

	for i, version := range versions {
		if !version.Equal(returned.Versions[i]) {
			t.Fatalf("expected returned[%d] = %v, got %v", i, version, returned.Versions[0])
		}
	}

	saved, err := client.GetManifest(ctx, organizationId, id)
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
