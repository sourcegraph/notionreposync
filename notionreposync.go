package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jomei/notionapi"

	"github.com/sourcegraph/notionreposync/markdown"
	"github.com/sourcegraph/notionreposync/notion"
	notionrenderer "github.com/sourcegraph/notionreposync/renderer"
	"github.com/sourcegraph/notionreposync/repository"
)

// Import turns a repository containing markdown files into a structure of Notion pages containing
// the converted content.
func Import(ctx context.Context, client *notionapi.Client, repo *repository.Repo, nd *NotionDoc) error {
	// Ensure the correct page structure exists on Notion, or create it.
	if err := nd.SyncPagesDB(context.Background(), client, repo); err != nil {
		return err
	}

	err := repo.Walk(func(d *repository.Document) error {
		// if d.Path != "index.md" && d.Path != "ref/ol.md" {
		// 	return nil
		// }
		println("🦀", "rendering", d.Path)

		notionPageID := string(nd.PageID)
		converter := markdown.NewProcessor(
			ctx,
			notion.NewBlockUpdater(client, notionPageID),
			notionrenderer.WithLinkResolver(repository.NewLinkResolver(repo, filepath.Dir(d.Path), notionPageID)),
		)

		b, err := os.ReadFile(filepath.Join(repo.LocalPath, d.Path))
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", d.Path, err)
		}

		if err := converter.ProcessMarkdown(b); err != nil {
			return fmt.Errorf("failed to convert %q: %w", d.Path, err)
		}
		return nil
	})

	return err
}
