package repository

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/jomei/notionapi"
)

// Document represents a Notion page corresponding to markdown page in the imported repository.
type Document struct {
	// Name is the name of the page, inferred from the original filename: "index.md" -> "index"
	Name string
	// ID is the Notion ID of the page.
	ID notionapi.PageID
	// Path is the Path of the page, relative to the repo root.
	Path string
}

// Folder represents a folder containing other documents.
type Folder struct {
	// Name is the name of the folder.
	Name string
	// Path is the Path of the folder, relative to the repo root.
	Path string
	// ChildFiles is a list of child pages.
	ChildFiles   []*Document
	ChildFolders []*Folder
}

// Repo represents a root of a collection of importable markdown documents in the repository.
// It's basically a common tree structure, starting with a root Folder.
type Repo struct {
	// Reference is the unique identifier for this repo. It enables to identify all imported pages
	// as belonging to the same repo.
	Reference string
	// Folders are a list of folders containing other pages.
	Folder *Folder
}

func NewRepo(path string, ref string) (*Repo, error) {
	repo := &Repo{
		Folder: &Folder{
			Name:         ".",
			Path:         "./",
			ChildFiles:   []*Document{},
			ChildFolders: []*Folder{},
		},
	}

	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		repo.Add(info.IsDir(), strings.TrimPrefix(p, path))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (d *Repo) Add(isDir bool, path string) {
	dir, file := filepath.Split(path)
	cur := d.Folder

	for _, d := range filepath.SplitList(dir) {
		found := false
		for _, f := range cur.ChildFolders {
			if f.Path == filepath.Join(cur.Path, d) {
				found = true
				cur = f
				break
			}
		}
		if !found {
			newCur := &Folder{
				Name:         d,
				Path:         filepath.Join(cur.Path, d),
				ChildFiles:   []*Document{},
				ChildFolders: []*Folder{},
			}
			cur.ChildFolders = append(cur.ChildFolders, newCur)
			cur = newCur
		}
	}

	if !isDir {
		cur.ChildFiles = append(cur.ChildFiles, &Document{
			Name: file,
			Path: path,
		})
	}
}

func (r *Repo) FindDocument(path string) *Document {
	var doc *Document
	r.Walk(func(d *Document) error {
		// TODO: might need to implement a way to stop the walk.
		if doc != nil {
			return nil
		}

		if d.Path == path {
			doc = d
		}
		return nil
	})
	return doc
}

func (p *Repo) Walk(fn func(*Document) error) error {
	var recur func(cur *Folder) error
	recur = func(cur *Folder) error {
		for _, p := range cur.ChildFiles {
			if err := fn(p); err != nil {
				return err
			}
		}

		for _, f := range cur.ChildFolders {
			if err := recur(f); err != nil {
				return err
			}
		}
		return nil
	}
	return recur(p.Folder)
}
