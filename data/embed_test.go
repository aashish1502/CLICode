package data

import (
	"io/fs"
	"strings"
	"testing"
)

func TestArtIsEmbedded(t *testing.T) {
	art, ok := Art()
	if !ok {
		t.Fatal("no embedded art")
	}
	if strings.TrimSpace(art) == "" {
		t.Error("embedded art is blank")
	}
}

// Problems() is rooted at the problems directory, so entries are bare filenames.
func TestProblemsAreEmbedded(t *testing.T) {
	entries, err := fs.ReadDir(Problems(), ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	var found bool
	for _, e := range entries {
		if e.Name() == "problems_list.json" {
			found = true
		}
	}
	if !found {
		t.Errorf("problems_list.json not embedded; got %d entries", len(entries))
	}
}
