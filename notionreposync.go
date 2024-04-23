package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jomei/notionapi"
	"github.com/sourcegraph/notionreposync/repository"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// Import turns a repository containing markdown files into a structure of Notion pages containing
// the converted content.
func Import(ctx context.Context, client *notionapi.Client, repo *repository.Repo, nd *NotionDoc, opts ...Option) error {
	// Ensure the correct page structure exists on Notion, or create it.
	if err := nd.SyncPagesDB(context.Background(), client, repo); err != nil {
		return err
	}

	opts = append(opts, WithContext(ctx))
	err := repo.Walk(func(d *repository.Document) error {
		// if d.Path != "index.md" && d.Path != "ref/ol.md" {
		// 	return nil
		// }
		println("🦀", "rendering", d.Path)

		r := NewRenderer(
			repo,
			client,
			filepath.Dir(d.Path),
			d.ID,
			opts...,
		)

		md := goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRenderer(
				renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(r, 1000))),
			),
		)

		b, err := os.ReadFile(filepath.Join(repo.LocalPath, d.Path))
		if err != nil {
			return fmt.Errorf("failed to read %q: %w", d.Path, err)
		}

		if err := md.Convert(b, io.Discard); err != nil {
			return fmt.Errorf("failed to convert %q: %w", d.Path, err)
		}
		return nil
	})

	return err
}
