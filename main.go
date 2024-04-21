package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/sourcegraph/notionreposync/repository"
)

func main() {
	notionKey := os.Getenv("NOTION_SECRET")
	repoPath := os.Args[1]

	// We'll compare paths with the separator, so let's add it if it's missing.
	if !strings.HasSuffix(repoPath, "/") {
		repoPath = repoPath + "/"
	}

	pageID := os.Getenv("NOTION_PAGE_ID")
	client := notionapi.NewClient(notionapi.Token(notionKey))

	nd := NewNotionDoc(pageID)
	err := nd.FetchMetadata(context.Background(), client)
	if err != nil {
		panic(err)
	}

	repo := repository.Repo{
		Folder: &repository.Folder{
			Name:         ".",
			Path:         "./",
			ChildFiles:   []*repository.Document{},
			ChildFolders: []*repository.Folder{},
		},
	}

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		repo.Add(info.IsDir(), strings.TrimPrefix(path, repoPath))
		return nil
	})

	// for _, p := range pagesRoot.Folder.childFiles {
	// 	println(p.Name)
	// 	println("\tpath:", p.path)
	// }
	// for _, f := range pagesRoot.Folder.childFolders {
	// 	println(f.Name)
	// 	println("\tpath:", f.path)
	// 	for _, p := range f.childFiles {
	// 		println("\t", p.Name)
	// 	}
	//
	// }

	// err = nd.CreatePagesDB(context.Background(), client)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// dbID, err := nd.FindPagesDB(context.Background(), client)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// if dbID == "" {
	// 	panic("cannot find pages database")
	// }

	nd.SyncPagesDB(context.Background(), client, &repo)
	blocks := []notionapi.Block{}

	ren := NewRenderer(
		&repo,
		client,
		repo.Folder.ChildFiles[0].ID,
		// WithoutAPI(&blocks),
	)

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRenderer(
			renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(ren, 1000))),
		),
	)

	b, err := os.ReadFile(filepath.Join(repoPath, repo.Folder.ChildFiles[0].Path))
	if err != nil {
		panic(err)
	}

	if err := md.Convert(b, io.Discard); err != nil {
		panic(err)
	}

	if err := json.NewEncoder(os.Stdout).Encode(blocks); err != nil {
		panic(err)
	}
}
