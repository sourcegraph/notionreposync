package notion

import (
	"context"

	"github.com/jomei/notionapi"
	"github.com/sourcegraph/notionreposync/renderer"
)

type BlockUpdater struct {
	client *notionapi.Client
	pageID string
}

var _ renderer.BlockUpdater = (*BlockUpdater)(nil)

func NewBlockUpdater(client *notionapi.Client, pageID string) *BlockUpdater {
	return &BlockUpdater{
		client: client,
		pageID: pageID,
	}
}

func (b *BlockUpdater) AddChildren(ctx context.Context, children []notionapi.Block) error {
	_, err := b.client.Block.AppendChildren(ctx, notionapi.BlockID(b.pageID), &notionapi.AppendBlockChildrenRequest{
		Children: children,
	})
	return err
}
