package main

import (
	"path/filepath"

	"github.com/jomei/notionapi"
)

// DocFile represents a Notion page corresponding to markdown page in the imported repository.
type DocFile struct {
	// Name is the name of the page, inferred from the original filename: "index.md" -> "index"
	Name string
	// ID is the Notion ID of the page.
	ID notionapi.PageID
	// path is the path of the page, relative to the repo root.
	path string
}

type Folder struct {
	// Name is the name of the folder.
	Name string
	// path is the path of the folder, relative to the repo root.
	path string
	// childFiles is a list of child pages.
	childFiles   []*DocFile
	childFolders []*Folder
}

// DocRoot represents a root of a collection of importable markdown documents in the repository.
type DocRoot struct {
	// Reference is the unique identifier for this repo. It enables to identify all imported pages
	// as belonging to the same repo.
	Reference string
	// Folders are a list of folders containing other pages.
	Folder *Folder
}

func (d *DocRoot) add(isDir bool, path string) {
	dir, file := filepath.Split(path)
	cur := d.Folder

	for _, d := range filepath.SplitList(dir) {
		found := false
		for _, f := range cur.childFolders {
			if f.path == filepath.Join(cur.path, d) {
				found = true
				cur = f
				break
			}
		}
		if !found {
			newCur := &Folder{
				Name:         d,
				path:         filepath.Join(cur.path, d),
				childFiles:   []*DocFile{},
				childFolders: []*Folder{},
			}
			cur.childFolders = append(cur.childFolders, newCur)
			cur = newCur
		}
	}

	if !isDir {
		cur.childFiles = append(cur.childFiles, &DocFile{
			Name: file,
			path: path,
		})
	}
}

func (p *DocRoot) Walk(fn func(*DocFile) error) error {
	var recur func(cur *Folder) error
	recur = func(cur *Folder) error {
		for _, p := range cur.childFiles {
			if err := fn(p); err != nil {
				return err
			}
		}

		for _, f := range cur.childFolders {
			if err := recur(f); err != nil {
				return err
			}
		}
		return nil
	}
	return recur(p.Folder)
}
