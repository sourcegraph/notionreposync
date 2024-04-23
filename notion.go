package main

import (
	"context"
	"errors"
	"time"

	"github.com/jomei/notionapi"
	"github.com/sourcegraph/notionreposync/repository"
)

var pagesDBTitle = "Pages"

// NotionDoc represents a Notion page that is the root of all imported pages from a Git repository holding
// markdown files.
type NotionDoc struct {
	PageID   notionapi.PageID
	metadata Metadata
}

type Metadata struct {
	Repository  string
	LastSyncAt  time.Time
	LastSyncRev string
}

func NewNotionDoc(pageID string) *NotionDoc {
	return &NotionDoc{
		PageID: notionapi.PageID(pageID),
	}
}

func (n *NotionDoc) FetchMetadata(ctx context.Context, c *notionapi.Client) error {
	page, err := c.Page.Get(ctx, n.PageID)
	if err != nil {
		return err
	}

	if page.Properties["Repository"].GetType() != "rich_text" {
		return errors.New("'Repository' property is not a text property")
	}
	rt := page.Properties["Repository"].(*notionapi.RichTextProperty)
	if len(rt.RichText) < 1 {
		return errors.New("'Repository' property is empty")
	}
	n.metadata.Repository = rt.RichText[0].Text.Content

	if page.Properties["LastSyncAt"].GetType() != "date" {
		return errors.New("'LastSyncAt' property is not a date property")
	}
	d := page.Properties["LastSyncAt"].(*notionapi.DateProperty)
	if d.Date != nil {
		n.metadata.LastSyncAt = time.Time(*d.Date.Start)
	}

	if page.Properties["LastSyncRev"].GetType() != "rich_text" {
		return errors.New("'LastSyncRev' property is not a text property")
	}
	rt = page.Properties["LastSyncRev"].(*notionapi.RichTextProperty)
	if len(rt.RichText) > 0 {
		n.metadata.LastSyncRev = rt.RichText[0].Text.Content
	}
	return nil
}

func (n *NotionDoc) WriteMetadata(ctx context.Context, c *notionapi.Client) error {
	props := make(notionapi.Properties)
	props["Repository"] = notionapi.RichTextProperty{
		RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: n.metadata.Repository}}},
	}
	props["LastSyncRev"] = notionapi.RichTextProperty{
		RichText: []notionapi.RichText{{Text: &notionapi.Text{Content: n.metadata.LastSyncRev}}},
	}
	d := notionapi.Date(n.metadata.LastSyncAt)
	props["LastSyncAt"] = notionapi.DateProperty{
		Date: &notionapi.DateObject{
			Start: &d,
		},
	}
	_, err := c.Page.Update(ctx, n.PageID, &notionapi.PageUpdateRequest{Properties: props})
	return err
}

func (n *NotionDoc) CreatePagesDB(ctx context.Context, c *notionapi.Client) (notionapi.DatabaseID, error) {
	db, err := c.Database.Create(ctx, &notionapi.DatabaseCreateRequest{
		Parent: notionapi.Parent{
			PageID: n.PageID,
		},
		Title: []notionapi.RichText{{Text: &notionapi.Text{Content: pagesDBTitle}}},
		Properties: notionapi.PropertyConfigs{
			"Title": notionapi.TitlePropertyConfig{Type: "title"},
			"_rev":  notionapi.RichTextPropertyConfig{Type: "rich_text"},
			"_path": notionapi.RichTextPropertyConfig{Type: "rich_text"},
		},
	})

	if err != nil {
		return "", err
	}

	return notionapi.DatabaseID(db.ID), nil
}

func (n *NotionDoc) FindPagesDB(ctx context.Context, c *notionapi.Client) (notionapi.DatabaseID, error) {
	resp, err := c.Block.GetChildren(ctx, notionapi.BlockID(n.PageID), &notionapi.Pagination{})
	if err != nil {
		return "", err
	}

	for _, bl := range resp.Results {
		if bl.GetType() == "child_database" {
			db := bl.(*notionapi.ChildDatabaseBlock)
			if db.ChildDatabase.Title == pagesDBTitle {
				return notionapi.DatabaseID(db.ID), nil
			}
		}
	}
	return "", nil
}

// SyncPagesDB fills the notion page IDs in the pages root with the IDs of their counterpart on Notion.
// If a page is missing, it will be created on the fly.
func (n *NotionDoc) SyncPagesDB(ctx context.Context, c *notionapi.Client, repo *repository.Repo) error {
	dbID, err := n.FindPagesDB(ctx, c)
	if err != nil {
		return err
	}
	if dbID == "" {
		var err error
		dbID, err = n.CreatePagesDB(ctx, c)
		if err != nil {
			return err
		}
	}

	err = repo.Walk(func(p *repository.Document) error {
		page, err := n.findPageInDB(ctx, c, dbID, p.Path)
		if err != nil {
			if !errors.Is(err, ErrPageNotFoundInDB) {
				return err
			}
			page, err = c.Page.Create(context.Background(), &notionapi.PageCreateRequest{
				Parent: notionapi.Parent{DatabaseID: dbID},
				Properties: map[string]notionapi.Property{
					"Title": notionapi.TitleProperty{
						Title: []notionapi.RichText{
							{Text: &notionapi.Text{Content: p.Path}},
						},
					},
					"_path": notionapi.RichTextProperty{
						RichText: []notionapi.RichText{
							{Text: &notionapi.Text{Content: p.Path}},
						},
					},
				},
			})
			if err != nil {
				return err
			}
		}
		p.ID = notionapi.PageID(page.ID)
		return nil
	})
	return err
}

var ErrPageNotFoundInDB = errors.New("not found")
var ErrPageDuplicateFoundInDB = errors.New("not found")

func (n *NotionDoc) findPageInDB(ctx context.Context, c *notionapi.Client, dbID notionapi.DatabaseID, pagePath string) (*notionapi.Page, error) {
	resp, err := c.Database.Query(ctx, dbID, &notionapi.DatabaseQueryRequest{
		Filter: notionapi.PropertyFilter{
			Property: "_path",
			RichText: &notionapi.TextFilterCondition{Equals: pagePath},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Results) < 1 {
		return nil, ErrPageNotFoundInDB
	}
	if len(resp.Results) > 1 {
		return nil, ErrPageDuplicateFoundInDB
	}
	return &resp.Results[0], nil
}
