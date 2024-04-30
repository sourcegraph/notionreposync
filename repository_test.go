package main

import (
	"testing"
)

func TestNewRepo(t *testing.T) {
	t.Run("walks the root path and add files", func(t *testing.T) {
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

func TestFindDocument(t *testing.T) {
	repo, err := NewRepo("../testdata/", "testref")
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path string
		want string
	}{
		{path: "index.md", want: "index.md"},
		{path: "/index.md", want: "index.md"},
		{path: "./index.md", want: "index.md"},
		{path: "./foo/index.md", want: "foo/index.md"},
		{path: "/foo/index.md", want: "foo/index.md"},
		{path: "./bar/baz/index.md", want: "bar/baz/index.md"},
		{path: "./foo/../bar/baz/index.md", want: "bar/baz/index.md"},
		{path: "dont-exist", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			d := repo.FindDocument(tt.path)
			if d == nil {
				if tt.want != "" {
					t.Logf("expected to find %q, but didn't", tt.want)
					t.Fail()
				}
				return
			}

			if d.Path != tt.want {
				t.Logf("expected path to be %q, but was %q", tt.want, d.Path)
				t.Fail()
			}
		})
	}
}
