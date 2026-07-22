package migrations

import (
	"testing"
	"testing/fstest"
)

func TestReadSeedFile(t *testing.T) {
	originalSeedFiles := seedFiles
	t.Cleanup(func() {
		seedFiles = originalSeedFiles
	})

	t.Run("missing file", func(t *testing.T) {
		seedFiles = fstest.MapFS{}
		_, available, err := readSeedFile("reference/missing.json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if available {
			t.Fatal("expected missing seed file to be unavailable")
		}
	})

	t.Run("available file", func(t *testing.T) {
		seedFiles = fstest.MapFS{
			"reference/data.json": &fstest.MapFile{Data: []byte("seed data")},
		}
		data, available, err := readSeedFile("reference/data.json")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !available {
			t.Fatal("expected seed file to be available")
		}
		if got, want := string(data), "seed data"; got != want {
			t.Fatalf("expected data %q, got %q", want, got)
		}
	})
}
