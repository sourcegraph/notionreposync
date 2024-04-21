package repository

import (
	"testing"
)

func TestRepository(t *testing.T) {
	t.Run("NewRepo walks the root path and add files", func(t *testing.T) {
		files := []string{
			"index.md",
			"other.md",
			"bar/index.md",
			"bar/baz/index.md",
			"foo/bar.md",
			"foo/index.md",
		}

		repo, err := NewRepo("../testdata/", "testref")
		if err != nil {
			t.Fatal(err)
		}

		for _, path := range files {
			if repo.FindDocument(path) == nil {
				t.Logf("expected to %s to be found, but wasn't", path)
				t.Fail()
			}
		}
	})
}
