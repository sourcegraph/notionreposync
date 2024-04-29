package renderer

import (
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// NewMarkdown returns a new Markdown parser based on the given Notion Renderer.
func NewMarkdown(r *Renderer) goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRenderer(
			renderer.NewRenderer(renderer.WithNodeRenderers(util.Prioritized(r, 1000))),
		),
	)
}
