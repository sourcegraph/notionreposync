package renderer

import (
	"context"

	"github.com/jomei/notionapi"
)

// See https://developers.notion.com/reference/patch-block-children
const MaxBlocksPerUpdate = 100

// See https://developers.notion.com/reference/request-limits#limits-for-property-values
const MaxRichTextContentLength = 2000

// BlockUpdater implements the desired handling for Notion blocks converted from
// Markdown. It should represent a single parent block, to which all children
// are added.
type BlockUpdater interface {
	// AddChildren should add the given children to the desired parent block.
	//
	// The caller calls it while respecting MaxBlocksPerUpdate and
	// MaxRichTextContentLength - implementations can assume the set of children
	// being added is of a reasonable size and adhere's to Notion's API limits.
	AddChildren(ctx context.Context, children []notionapi.Block) error
}
