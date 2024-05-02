package renderer_test

import (
	"context"
	"testing"

	"github.com/hexops/autogold/v2"
	"github.com/jomei/notionapi"
	"github.com/stretchr/testify/assert"

	"github.com/sourcegraph/notionreposync/markdown"
	"github.com/sourcegraph/notionreposync/renderer"
	"github.com/sourcegraph/notionreposync/renderer/renderertest"
)

func TestDiscardLinkResolver(t *testing.T) {
	ctx := context.Background()
	blockUpdater := &renderertest.MockBlockUpdater{}

	p := markdown.NewProcessor(ctx, blockUpdater,
		renderer.WithLinkResolver(renderer.DiscardLinkResolver{}))
	err := p.ProcessMarkdown([]byte("[This link](#foo) should be [discarded](#bar) and [this too](#asdf)"))
	assert.NoError(t, err)

	// Result should not have any Links
	autogold.Expect([]notionapi.Block{&notionapi.ParagraphBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectType("block"),
			Type:   notionapi.BlockType("paragraph"),
		},
		Paragraph: notionapi.Paragraph{RichText: []notionapi.RichText{
			{Text: &notionapi.Text{
				Content: "This link",
			}},
			{Text: &notionapi.Text{Content: " should be "}},
			{Text: &notionapi.Text{Content: "discarded"}},
			{Text: &notionapi.Text{Content: " and "}},
			{Text: &notionapi.Text{Content: "this too"}},
		}},
	}}).Equal(t, blockUpdater.GetAddedBlocks())
}
