package renderer

import (
	"bytes"
	"context"
	"regexp"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark/ast"
	goldmark "github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

// nodeRenderer encapsulates all logic required to convert Markdown content
// into Notion document blocks by implementing Goldmark's NodeRenderer interface.
//
// nodeRenderer MUST remain free of notionreposync's domain-specific logic for
// external use, and instead focus on converting Markdown to Notion blocks,
// processing them in config interfaces like LinkResolver and batching converted
// blocks to BlockUpdater.
type nodeRenderer struct {
	conf  config
	block BlockUpdater
	c     *cursor
}

var _ goldmark.NodeRenderer = (*nodeRenderer)(nil)

// NewNodeRenderer returns a new NodeRenderer that ingests Markdown and applies
// converted Notion blocks to BlockUpdater.
//
// Callers that just want to process Markdown should use markdown.NewProcessor
// instead.
func NewNodeRenderer(ctx context.Context, block BlockUpdater, opts ...Option) goldmark.NodeRenderer {
	r := &nodeRenderer{
		conf:  newConfig(ctx),
		block: block,
		c:     &cursor{},
	}
	for _, opt := range opts {
		opt.setConfig(&r.conf)
	}
	return r
}

func (r *nodeRenderer) RegisterFuncs(reg goldmark.NodeRendererFuncRegisterer) {
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

func (r *nodeRenderer) renderDocument(_ util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		r.c = &cursor{
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
func (r *nodeRenderer) writeBlocks() error {
	// If we have less than 100 blocks, we can just append them all at once.
	if len(r.c.rootBlocks) < 100 {
		return r.block.AddChildren(r.conf.ctx, r.c.rootBlocks)
	}

	acc := []notionapi.Block{}
	for _, block := range r.c.rootBlocks {
		if len(acc) < MaxBlocksPerUpdate-1 {
			// Minus one because otherwise, we'll have one too many block when flushing.
			acc = append(acc, block)
		} else {
			if err := r.block.AddChildren(r.conf.ctx, append(acc, block)); err != nil {
				return err
			}
			acc = []notionapi.Block{}
		}
	}

	return nil
}

func (r *nodeRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

// Match for e.g. '[!IMPORTANT]'-style GitHub callouts in blockquotes.
var calloutRegexp = regexp.MustCompile(`^\[!([A-Z]+)\]`)

func (r *nodeRenderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Look for a callout-looking thing.
		var block notionapi.Block
		if matches := calloutRegexp.FindSubmatch(node.Text(source)); len(matches) > 0 {
			block = &notionapi.CalloutBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockCallout,
				},
				Callout: notionapi.Callout{
					RichText: []notionapi.RichText{},
					Icon: func() *notionapi.Icon {
						var emoji string
						// matches[1] is the capture group in calloutRegexp
						switch string(matches[1]) {
						case "NOTE":
							emoji = "🔔"
						case "IMPORTANT":
							emoji = "⭐"
						case "WARNING":
							emoji = "🚨"
						default:
							emoji = "💡"
						}
						e := notionapi.Emoji(emoji)
						return &notionapi.Icon{
							Type:  "emoji",
							Emoji: &e,
						}
					}(),
				},
			}
		} else {
			block = &notionapi.QuoteBlock{
				BasicBlock: notionapi.BasicBlock{
					Object: notionapi.ObjectTypeBlock,
					Type:   notionapi.BlockQuote,
				},
				Quote: notionapi.Quote{
					RichText: []notionapi.RichText{},
				},
			}
		}

		r.c.Set(node, block)
		r.c.AppendBlock(block)
		r.c.Descend(node)
	} else {
		r.c.Ascend()
	}

	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			lineContents := line.Value(source)
			if i == node.Lines().Len()-1 { // trim trailing newlines
				lineContents = bytes.TrimRight(lineContents, "\n")
			}
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

func (r *nodeRenderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.FencedCodeBlock)
	if entering {
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			lineContents := line.Value(source)
			if i == node.Lines().Len()-1 { // trim trailing newlines
				lineContents = bytes.TrimRight(lineContents, "\n")
			}
			sb.Write(lineContents)
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

func (r *nodeRenderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return r.renderCodeBlock(w, source, node, entering)
}

func (r *nodeRenderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *nodeRenderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *nodeRenderer) renderTextBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		// r.c.Set(node, r.c.Block())
		// r.c.Descend(node)
	} else {
		// r.c.Ascend()
	}

	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *nodeRenderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *nodeRenderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *nodeRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

		dest, err := r.conf.links.ResolveLink(dest)
		if err != nil {
			return ast.WalkStop, err
		}

		r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: linkText, Link: &notionapi.Link{Url: dest}}})
		return ast.WalkSkipChildren, nil
	}

	return ast.WalkContinue, nil
}

func (r *nodeRenderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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

func (r *nodeRenderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Text)
	segment := n.Segment

	if !entering {
		return ast.WalkContinue, nil
	}

	// Special handling for callouts - we want to ignore the callout indicator,
	// as defined in calloutRegexp, by skipping any content within the range of
	// the match. To do this, we need to figure out a sliding window because
	// the AST parses the callout indicator as 3 nodes:
	// - "["
	// - "!NOTE"
	// - "]"
	// So we assemble all possible 3-node window (3-grams) and check each for
	// a callout indicator. If there is one, we drop this content.
	for _, gram := range getNeighbouring3Grams(node) {
		if nodeMatchesRegexp(calloutRegexp, source, gram[0], gram[1], gram[2]) {
			return ast.WalkSkipChildren, nil
		}
	}

	r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})

	return ast.WalkContinue, nil
}

// getNeighbouring3Grams returns a list of 3-grams of the given node's neighbours
// including the node itself. Elements of each 3-gram may be nil - callers should
// check before using each node.
func getNeighbouring3Grams(node ast.Node) [][3]ast.Node {
	var windows [][3]ast.Node
	var prev, middle, next = node.PreviousSibling(), node, node.NextSibling()
	windows = append(windows, [3]ast.Node{prev, middle, next})
	if next != nil {
		windows = append(windows, [3]ast.Node{node, next, next.NextSibling()})
	}
	if prev != nil {
		windows = append(windows, [3]ast.Node{prev.PreviousSibling(), prev, node})
	}
	return windows
}

// nodeMatchesRegexp returns true if the given nodes match the callout
// indicator as defined by calloutRegexp.
func nodeMatchesRegexp(re *regexp.Regexp, source []byte, prev, middle, next ast.Node) bool {
	if prev == nil || middle == nil || next == nil {
		return false
	}
	ptext, ntext := prev.Text(source), next.Text(source)
	if bytes.Equal(ptext, []byte("[")) && bytes.Equal(ntext, []byte("]")) {
		maybeCallout := string(ptext) + string(middle.Text(source)) + string(ntext)
		if matches := re.FindStringSubmatch(maybeCallout); len(matches) > 0 {
			return true
		}
	}
	return false
}

func (r *nodeRenderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
