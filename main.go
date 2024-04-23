package main

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/jomei/notionapi"
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

	blocks := []notionapi.Block{}
	repo, err := repository.NewRepo(repoPath, "sourcegraph/sourcegraph")
	if err := Import(context.Background(), client, repo, nd); err != nil {
		panic(err)
	}

	if len(blocks) > 0 {
		if err := json.NewEncoder(os.Stdout).Encode(blocks); err != nil {
			panic(err)
		}
	}
}
