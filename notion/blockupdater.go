package notion

import (
	"context"

	"github.com/jomei/notionapi"
	"github.com/sourcegraph/notionreposync/renderer"
)

type PageBlockUpdater struct {
	client *notionapi.Client
	pageID string
}

var _ renderer.BlockUpdater = (*PageBlockUpdater)(nil)

// NewPageBlockUpdater creates a new BlockUpdater for the given Notion page, which
// adds all children to the given pageID, to be provided to 'notionreposync/renderer'.
func NewPageBlockUpdater(client *notionapi.Client, pageID string) *PageBlockUpdater {
	return &PageBlockUpdater{
		client: client,
		pageID: pageID,
	}
}

func (b *PageBlockUpdater) AddChildren(ctx context.Context, children []notionapi.Block) error {
	// As documented in renderer.BlockUpdater, we can trust that the given
	// children adheres to Notion API requirements, and we do not need to do
	// separate batching/etc here.
	_, err := b.client.Block.AppendChildren(ctx, notionapi.BlockID(b.pageID), &notionapi.AppendBlockChildrenRequest{
		Children: children,
	})
	return err
}
