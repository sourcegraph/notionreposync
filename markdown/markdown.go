package markdown

import (
	"context"
	"io"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	goldmarkrenderer "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"

	"github.com/sourcegraph/notionreposync/renderer"
)

type Processor struct{ md goldmark.Markdown }

// NewProcessor returns a new simple Markdown procesosr that can be used to
// process Markdown test, sending the resulting Notion document blocks to the
// given BlockUpdater.
func NewProcessor(ctx context.Context, blocks renderer.BlockUpdater, opts ...renderer.Option) Processor {
	r := renderer.NewNodeRenderer(ctx, blocks, opts...)
	return Processor{
		goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRenderer(
				goldmarkrenderer.NewRenderer(goldmarkrenderer.WithNodeRenderers(util.Prioritized(r, 1000))),
			),
		),
	}
}

// ProcessMarkdown ingests the given Markdown source, sending converted to
// Notion blocks to the BlockUpdater given in NewProcessor(...)
func (c Processor) ProcessMarkdown(source []byte, opts ...parser.ParseOption) error {
	return c.md.Convert(
		source,
		io.Discard, // no destination - our renderer sends blocks to BlockUpdater
		opts...)
}
