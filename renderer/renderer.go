package renderer

import (
	"bytes"
	"context"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// See https://developers.notion.com/reference/patch-block-children
const MaxBlocksPerUpdate = 100

// See https://developers.notion.com/reference/request-limits#limits-for-property-values
const MaxRichTextContentLength = 2000

type BlockUpdater interface {
	// AddChildren should add the given children to the desired parent block.
	// It is called respecting MaxBlocksPerUpdate and MaxRichTextContentLength.
	AddChildren(ctx context.Context, children []notionapi.Block) error
}

type Renderer struct {
	Config

	block BlockUpdater

	c *Cursor
}

var _ renderer.NodeRenderer = (*Renderer)(nil)

func NewRenderer(ctx context.Context, blocks BlockUpdater, opts ...Option) *Renderer {
	r := &Renderer{
		Config: NewConfig(ctx),
	}

	for _, opt := range opts {
		opt.SetConfig(&r.Config)
	}
	return r
}

func (r *Renderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	// blocks

	reg.Register(ast.KindDocument, r.renderDocument)
	reg.Register(ast.KindHeading, r.renderHeading)
	reg.Register(ast.KindBlockquote, r.renderBlockquote)
	reg.Register(ast.KindCodeBlock, r.renderCodeBlock)
	reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
	reg.Register(ast.KindHTMLBlock, r.renderHTMLBlock)
	reg.Register(ast.KindList, r.renderList)
	reg.Register(ast.KindListItem, r.renderListItem)
	reg.Register(ast.KindParagraph, r.renderParagraph)
	reg.Register(ast.KindTextBlock, r.renderTextBlock)
	reg.Register(ast.KindThematicBreak, r.renderThematicBreak)

	// inline

	reg.Register(ast.KindAutoLink, r.renderAutoLink)
	reg.Register(ast.KindCodeSpan, r.renderCodeSpan)
	reg.Register(ast.KindEmphasis, r.renderEmphasis)
	reg.Register(ast.KindImage, r.renderImage)
	reg.Register(ast.KindLink, r.renderLink)
	reg.Register(ast.KindRawHTML, r.renderRawHTML)
	reg.Register(ast.KindText, r.renderText)
	reg.Register(ast.KindString, r.renderString)
}

func (r *Renderer) renderDocument(_ util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.c = &Cursor{
			rootBlocks: []notionapi.Block{},
			m:          make(map[ast.Node]notionapi.Block),
			cur:        node,
		}
	} else {
		if err := r.writeBlocks(); err != nil {
			return ast.WalkStop, err
		}
	}
	return ast.WalkContinue, nil
}

// writeBlocks performs API calls to the notion API to append the blocks to the page.
//
// Implementation: see https://developers.notion.com/reference/patch-block-children, we cannot append more
// than 100 blocks at a time, so we need to split the blocks into chunks of 100.
func (r *Renderer) writeBlocks() error {
	// If we have less than 100 blocks, we can just append them all at once.
	if len(r.c.rootBlocks) < 100 {
		return r.block.AddChildren(r.ctx, r.c.rootBlocks)
	}

	acc := []notionapi.Block{}
	for _, block := range r.c.rootBlocks {
		if len(acc) < MaxBlocksPerUpdate-1 {
			// Minus one because otherwise, we'll have one too many block when flushing.
			acc = append(acc, block)
		} else {
			if err := r.block.AddChildren(r.ctx, append(acc, block)); err != nil {
				return err
			}
			acc = []notionapi.Block{}
		}
	}

	return nil
}

func (r *Renderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		n := node.(*ast.Heading)
		var block notionapi.Block

		switch n.Level {
		case 1:
			block = &notionapi.Heading1Block{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeHeading1,
				},
				Heading1: notionapi.Heading{
					RichText: []notionapi.RichText{},
				},
			}
		case 2:
			block = &notionapi.Heading2Block{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeHeading2,
				},
				Heading2: notionapi.Heading{
					RichText: []notionapi.RichText{},
				},
			}
		case 3:
			block = &notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeHeading3,
				},
				Heading3: notionapi.Heading{
					RichText: []notionapi.RichText{},
				},
			}
		default:
			// TODO could we use bold or something else to mimick that level?
			block = &notionapi.Heading3Block{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeHeading3,
				},
				Heading3: notionapi.Heading{
					RichText: []notionapi.RichText{},
				},
			}
		}

		r.c.Set(node, block)
		r.c.Descend(node)
		r.c.AppendBlock(block)
	} else {
		r.c.Ascend()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		block := &notionapi.QuoteBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockQuote,
			},
			Quote: notionapi.Quote{
				RichText: []notionapi.RichText{},
			},
		}
		r.c.Set(node, block)
		r.c.AppendBlock(block)
		r.c.Descend(node)
	} else {
		r.c.Ascend()
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			sb.Write(line.Value(source))
		}

		rts := []notionapi.RichText{{Text: &notionapi.Text{Content: sb.String()}}}

		if sb.Len() > MaxRichTextContentLength {
			rts = []notionapi.RichText{}
			chunks := chunkText(sb.String())
			for _, chunk := range chunks {
				rts = append(rts, notionapi.RichText{Text: &notionapi.Text{Content: chunk}})
			}
		}

		block := &notionapi.CodeBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeCode,
			},
			Code: notionapi.Code{
				Language: "plain text",
			},
		}
		block.Code.RichText = rts

		r.c.Set(node, block)
		r.c.AppendBlock(block)
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			sb.Write(line.Value(source))
		}

		rts := []notionapi.RichText{{Text: &notionapi.Text{Content: sb.String()}}}

		if sb.Len() > MaxRichTextContentLength {
			rts = []notionapi.RichText{}
			chunks := chunkText(sb.String())
			for _, chunk := range chunks {
				rts = append(rts, notionapi.RichText{Text: &notionapi.Text{Content: chunk}})
			}
		}

		block := &notionapi.CodeBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeCode,
			},
			Code: notionapi.Code{
				Language: supportedLanguageOrPlainText(string(n.Language(source))),
			},
		}
		block.Code.RichText = rts

		r.c.Set(node, block)
		r.c.AppendBlock(block)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return r.renderCodeBlock(w, source, node, entering)
}

func (r *Renderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.Parent().(*ast.List)
	if entering {
		var block notionapi.Block
		if n.IsOrdered() {
			block = &notionapi.NumberedListItemBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeNumberedListItem,
				},
				NumberedListItem: notionapi.ListItem{
					RichText: []notionapi.RichText{
						// {Text: &notionapi.Text{}},
					},
				},
			}
		} else {
			block = &notionapi.BulletedListItemBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockTypeBulletedListItem,
				},
				BulletedListItem: notionapi.ListItem{
					RichText: []notionapi.RichText{
						// {Text: &notionapi.Text{}},
					},
				},
			}
		}

		r.c.Set(node, block)
		r.c.AppendBlock(block, "here")
		r.c.Descend(node)
	} else {
		r.c.Ascend()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Markdown AST has paragraphs inside blockquotes, but Notion doesn't, so instead, we just pass through.
	if node.Parent().Kind() == ast.KindBlockquote {
		return ast.WalkContinue, nil
	}
	// Markdown AST has paragraphs inside list items, but Notion doesn't, so instead, we just pass through.
	if node.Parent().Kind() == ast.KindListItem {
		return ast.WalkContinue, nil
	}

	if entering {
		block := &notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{},
		}
		r.c.Set(node, block)
		r.c.AppendBlock(block)
		r.c.Descend(node)
	} else {
		r.c.Ascend()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTextBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// r.c.Set(node, r.c.Block())
		// r.c.Descend(node)
	} else {
		// r.c.Ascend()
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}

	block := &notionapi.DividerBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeDivider,
		},
		Divider: notionapi.Divider{},
	}

	r.c.Set(node, block)
	r.c.AppendBlock(block)

	return ast.WalkContinue, nil
}

func (r *Renderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		var txt string
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			segment := c.(*ast.Text).Segment
			txt = txt + string(segment.Value(source))
		}

		r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: txt}, Annotations: &notionapi.Annotations{Code: true}})
		return ast.WalkSkipChildren, nil
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Emphasis)

	if !entering {
		rt := r.c.RichText()
		if rt.Annotations == nil {
			rt.Annotations = &notionapi.Annotations{}
		}

		if n.Level == 1 {
			rt.Annotations.Italic = true
		} else {
			rt.Annotations.Bold = true
		}
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Link)
	if entering {
		if IsDangerousURL(n.Destination) {
			r.renderCodeSpan(w, source, node, entering)
			return ast.WalkContinue, nil
		}

		dest := string(n.Destination)
		linkText := string(node.Text(source))
		if linkText == "" {
			linkText = dest
		}

		dest, err := r.links.ResolveLink(dest)
		if err != nil {
			return ast.WalkStop, err
		}

		r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: linkText, Link: &notionapi.Link{Url: dest}}})
		return ast.WalkSkipChildren, nil
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// Notion doesn't support this, so we just create a code block instead.

	if entering {
		n := node.(*ast.RawHTML)
		l := n.Segments.Len()
		var txt string
		for i := 0; i < l; i++ {
			segment := n.Segments.At(i)
			txt += string(segment.Value(source))
		}
		r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: txt}, Annotations: &notionapi.Annotations{Code: true}})
		return ast.WalkSkipChildren, nil
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Text)
	segment := n.Segment

	if !entering {
		return ast.WalkContinue, nil
	}

	r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})

	return ast.WalkContinue, nil
}

func (r *Renderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

var supportedLanguages = []string{
	"abap",
	"agda",
	"arduino",
	"assembly",
	"bash",
	"basic",
	"bnf",
	"c",
	"c#",
	"c++",
	"clojure",
	"coffeescript",
	"coq",
	"css",
	"dart",
	"dhall",
	"diff",
	"docker",
	"ebnf",
	"elixir",
	"elm",
	"erlang",
	"f#",
	"flow",
	"fortran",
	"gherkin",
	"glsl",
	"go",
	"graphql",
	"groovy",
	"haskell",
	"html",
	"idris",
	"java",
	"javascript",
	"json",
	"julia",
	"kotlin",
	"latex",
	"less",
	"lisp",
	"livescript",
	"llvm ir",
	"lua",
	"makefile",
	"markdown",
	"markup",
	"matlab",
	"mathematica",
	"mermaid",
	"nix",
	"notion formula",
	"objective-c",
	"ocaml",
	"pascal",
	"perl",
	"php",
	"plain text",
	"powershell",
	"prolog",
	"protobuf",
	"purescript",
	"python",
	"r",
	"racket",
	"reason",
	"ruby",
	"rust",
	"sass",
	"scala",
	"scheme",
	"scss",
	"shell",
	"solidity",
	"sql",
	"swift",
	"toml",
	"typescript",
	"vb.net",
	"verilog",
	"vhdl",
	"visual basic",
	"webassembly",
	"xml",
	"yaml",
	"java",
	"c",
	"c++",
	"c#",
}

func supportedLanguageOrPlainText(lang string) string {
	for _, l := range supportedLanguages {
		if lang == l {
			return lang
		}
	}
	return "plain text"
}

var bDataImage = []byte("data:image/")
var bPng = []byte("png;")
var bGif = []byte("gif;")
var bJpeg = []byte("jpeg;")
var bWebp = []byte("webp;")
var bSvg = []byte("svg+xml;")
var bJs = []byte("javascript:")
var bVb = []byte("vbscript:")
var bFile = []byte("file:")
var bData = []byte("data:")

func hasPrefix(s, prefix []byte) bool {
	return len(s) >= len(prefix) && bytes.Equal(bytes.ToLower(s[0:len(prefix)]), bytes.ToLower(prefix))
}

// IsDangerousURL returns true if the given url seems a potentially dangerous url,
// otherwise false.
// Copied from https://sourcegraph.com/github.com/yuin/goldmark/-/blob/renderer/html/html.go?L997
func IsDangerousURL(url []byte) bool {
	if hasPrefix(url, bDataImage) && len(url) >= 11 {
		v := url[11:]
		if hasPrefix(v, bPng) || hasPrefix(v, bGif) ||
			hasPrefix(v, bJpeg) || hasPrefix(v, bWebp) ||
			hasPrefix(v, bSvg) {
			return false
		}
		return true
	}
	return hasPrefix(url, bJs) || hasPrefix(url, bVb) ||
		hasPrefix(url, bFile) || hasPrefix(url, bData)
}

func chunkText(txt string) []string {
	runes := []rune(txt)
	chunks := []string{}
	limit := MaxRichTextContentLength - 1

	var sb strings.Builder
	for i, r := range runes {
		sb.WriteRune(r)
		if i%limit == 0 && i != 0 {
			chunks = append(chunks, sb.String())
			sb.Reset()
		}
	}

	// If the last rune index is exactly maxRichTextContentLength, it's been appended
	// already, but if otherwise, we need to do it manually.
	if len(runes)%limit != 0 {
		chunks = append(chunks, sb.String())
	}

	return chunks
}