package model

import (
	"testing"
	"time"
)

func TestSortEmptyHistory(t *testing.T) {
	var versions []*Version
	got, err := toposort(versions)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected an empty manifest, found %d elements", len(got))
	}
}

func TestSortSingleVersion(t *testing.T) {
	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")
	versions := []*Version{
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Aardvark Wikipedia Article",
			Hash:     "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y",
			PathType: PathTypeFile,
			Parents:  nil,
		},
	}
	got, err := toposort(versions)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("expected an exactly 3 versions, found %d", len(got))
	}
}

func TestSortSimpleLinearManifestByParentage(t *testing.T) {
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")

	versions := []*Version{
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Cold War Wikipedia Article",
			Hash:     hash3,
			PathType: PathTypeFile,
			Parents:  []string{hash2},
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Aardvark Wikipedia Article",
			Hash:     hash1,
			PathType: PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: PathTypeFile,
			Parents:  []string{hash1},
		},
	}
	got, err := toposort(versions)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected an exactly 3 versions, found %d", len(got))
	}

	expected := []*Version{versions[1], versions[2], versions[0]}
	for i := range got {
		if !got[i].Equal(expected[i]) {
			t.Errorf("expected[%d] = %v; %v", i, expected[i], got[i])
		}
	}
}

func TestSortSimpleLinearManifestByDate(t *testing.T) {
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")
	time2, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T11:00:00")
	time3, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T12:00:00")

	versions := []*Version{
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time3,
			Message:  "Cold War Wikipedia Article",
			Hash:     hash3,
			PathType: PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Aardvark Wikipedia Article",
			Hash:     hash1,
			PathType: PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time2,
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: PathTypeFile,
			Parents:  nil,
		},
	}
	got, err := toposort(versions)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected an exactly 3 versions, found %d", len(got))
	}

	expected := []*Version{versions[1], versions[2], versions[0]}
	for i := range got {
		if !got[i].Equal(expected[i]) {
			t.Errorf("expected[%d] = %v; %v", i, expected[i], got[i])
		}
	}
}

func TestSortDisconnected(t *testing.T) {
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")
	time2, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T11:00:00")
	time3, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T12:00:00")

	versions := []*Version{
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time3,
			Message:  "Cold War Wikipedia Article",
			Hash:     hash3,
			PathType: PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Aardvark Wikipedia Article",
			Hash:     hash1,
			PathType: PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time2,
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: PathTypeFile,
			Parents:  []string{hash1},
		},
	}
	got, err := toposort(versions)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected an exactly 3 versions, found %d", len(got))
	}

	expected := []*Version{versions[1], versions[2], versions[0]}
	for i := range got {
		if !got[i].Equal(expected[i]) {
			t.Errorf("expected[%d] = %v; %v", i, expected[i], got[i])
		}
	}
}

func TestSortMultipleRoots(t *testing.T) {
	hash1 := "bafkreidpvvw3h2f4hdhznb5shvncgqj5j3wht3k7ewxfpy4rk5ep4h7j5y"
	hash2 := "bafybeicysbsujtlq2d7ygbab47lywcb7vehx64zwv4etis6hom45iorjwm"
	hash3 := "bafybeifvnc6qllx2cuwcrkf5fxuocg7jraesroxeuzd3ru7aexnayjnjgu"

	time1, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T10:00:00")
	time2, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T11:00:00")
	time3, _ := time.Parse("2006-01-02T15:04:05", "2023-03-16T12:00:00")

	versions := []*Version{
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time3,
			Message:  "Cold War Wikipedia Article",
			Hash:     hash3,
			PathType: PathTypeFile,
			Parents:  []string{hash1, hash2},
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time2,
			Message:  "Africa Wikipedia Article",
			Hash:     hash2,
			PathType: PathTypeFile,
			Parents:  nil,
		},
		{
			Author:   "Douglas G. Moore <doug@dglmoore.com>",
			Date:     time1,
			Message:  "Aardvark Wikipedia Article",
			Hash:     hash1,
			PathType: PathTypeFile,
			Parents:  nil,
		},
	}
	got, err := toposort(versions)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("expected an exactly 3 versions, found %d", len(got))
	}

	expected := []*Version{versions[2], versions[1], versions[0]}
	for i := range got {
		if !got[i].Equal(expected[i]) {
			t.Errorf("expected[%d] = %v; %v", i, expected[i], got[i])
		}
	}
}
