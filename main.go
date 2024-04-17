package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
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

	pagesRoot := DocRoot{
		Folder: &Folder{
			Name:         ".",
			path:         "./",
			childFiles:   []*DocFile{},
			childFolders: []*Folder{},
		},
	}

	filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		pagesRoot.add(info.IsDir(), strings.TrimPrefix(path, repoPath))
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

	nd.SyncPagesDB(context.Background(), client, &pagesRoot)
	// pagesRoot.Walk(func(p *Page) error {
	// 	println(p.ID)
	// 	return nil
	// })
	//
	notionRenderer := Renderer{
		docRoot: &pagesRoot,
		client:  client,
		pageID:  pagesRoot.Folder.childFiles[0].ID,
		ctx:     context.Background(),
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRenderer(
			renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(&notionRenderer, 1000))),
		),
	)

	println(pagesRoot.Folder.childFiles[0].path)
	b, err := os.ReadFile(filepath.Join(repoPath, pagesRoot.Folder.childFiles[0].path))
	if err != nil {
		panic(err)
	}

	if err := md.Convert(b, io.Discard); err != nil {
		panic(err)
	}
}
