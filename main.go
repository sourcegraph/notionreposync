package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/urfave/cli/v2"
)

var app = cli.App{
	Name:  "notionreposync",
	Usage: "notionreposync --page-id <notion-page-id> --api-key <notion-api-key> <repo-path>",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "page-id",
			Required: true,
			EnvVars:  []string{"NOTION_PAGE_ID"},
			Action: func(ctx *cli.Context, v string) error {
				if v == "" {
					return fmt.Errorf("page-id (NOTION_PAGE_ID) cannot be empty")
				}
				return nil
			},
		},
		&cli.StringFlag{
			Name:     "api-key",
			Required: true,
			EnvVars:  []string{"NOTION_SECRET"},
			Action: func(ctx *cli.Context, v string) error {
				if v == "" {
					return fmt.Errorf("api-key (NOTION_SECRET) cannot be empty")
				}
				return nil
			},
		},
	},
	Action: run,
}

func run(ctx *cli.Context) error {
	notionKey := ctx.String("api-key")
	if ctx.Args().Len() < 1 {
		return fmt.Errorf("no argument provided for repo-path")
	}
	repoPath := ctx.Args().First()
	// We'll compare paths with the separator, so let's add it if it's missing.
	if !strings.HasSuffix(repoPath, "/") {
		repoPath = repoPath + "/"
	}

	pageID := ctx.String("page-id")
	logger := slog.Default().WithGroup("sync").With(slog.String("pageID", pageID))

	err := doSync(logger, notionKey, pageID, repoPath)
	if err != nil {
		logger.Error("syncing failed", slog.Any("error", err))
	}

	return err
}

func doSync(logger *slog.Logger, notionKey, pageID, repoPath string) error {
	logger.Info("starting sync")
	logger.Debug("creating notion api client")
	client := notionapi.NewClient(notionapi.Token(notionKey))

	nd := NewNotionDoc(pageID)
	logger.Debug("retrieving metdata for page")
	err := nd.FetchMetadata(context.Background(), client)
	if err != nil {
		return err
	}

	blocks := []notionapi.Block{}
	logger.Info("loading repo 'sourcegraph/sourcegraph'", slog.String("repoPath", repoPath))
	repo, err := NewRepo(repoPath, "sourcegraph/sourcegraph")
	logger.Info("loading complete", slog.String("repoPath", repoPath))
	if err != nil {
		return err
	}
	logger.Info("starting import from repo to page", slog.String("repoPath", repoPath))
	if err := Import(context.Background(), client, repo, nd); err != nil {
		logger.Info("import failed", slog.String("repoPath", repoPath), slog.Any("error", err))
		return err
	}
	logger.Info("import complete", slog.String("repoPath", repoPath))

	if len(blocks) > 0 {
		if err := json.NewEncoder(os.Stdout).Encode(blocks); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		panic(err)
	}
}
