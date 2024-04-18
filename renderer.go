package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/jomei/notionapi"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type Renderer struct {
	docRoot *DocRoot
	client  *notionapi.Client
	pageID  notionapi.PageID
	ctx     context.Context

	page         *notionapi.Page
	rootBlocks   []notionapi.Block
	parentBlocks *[]notionapi.Block
	m            map[ast.Node]notionapi.Block
	idx          int

	c *Cursor
}

type Cursor struct {
	rootBlocks []notionapi.Block
	m          map[ast.Node]notionapi.Block
	cur        ast.Node
}

func (c *Cursor) RichText() *notionapi.RichText {
	switch block := c.m[c.cur].(type) {
	case *notionapi.ParagraphBlock:
		return &block.Paragraph.RichText[len(block.Paragraph.RichText)-1]
	case *notionapi.BulletedListItemBlock:
		return &block.BulletedListItem.RichText[len(block.BulletedListItem.RichText)-1]
	case *notionapi.NumberedListItemBlock:
		return &block.NumberedListItem.RichText[len(block.NumberedListItem.RichText)-1]
	case *notionapi.Heading1Block:
		return &block.Heading1.RichText[len(block.Heading1.RichText)-1]
	default:
		fmt.Printf("unknown block type: %T\n", block)
	}
	return nil
}

func (c *Cursor) Block() notionapi.Block {
	return c.m[c.cur]
}

func (c *Cursor) AppendRichText(rt *notionapi.RichText) {
	switch block := c.m[c.cur].(type) {
	case *notionapi.ParagraphBlock:
		block.Paragraph.RichText = append(block.Paragraph.RichText, *rt)
	case *notionapi.BulletedListItemBlock:
		block.BulletedListItem.RichText = append(block.BulletedListItem.RichText, *rt)
	case *notionapi.NumberedListItemBlock:
		block.NumberedListItem.RichText = append(block.NumberedListItem.RichText, *rt)
	case *notionapi.Heading1Block:
		block.Heading1.RichText = append(block.Heading1.RichText, *rt)
	default:
		fmt.Printf("unknown block type: %T\n", block)
		panic("here")
	}
}

func (c *Cursor) AppendBlock(b notionapi.Block, things ...string) {
	if c.cur.Kind() == ast.KindDocument {
		c.rootBlocks = append(c.rootBlocks, b)
	} else if c.cur.Parent().Kind() == ast.KindDocument {
		c.rootBlocks = append(c.rootBlocks, b)
	} else {
		switch block := c.Block().(type) {
		case *notionapi.ParagraphBlock:
			println("appending block to paragraph")
			block.Paragraph.Children = append(block.Paragraph.Children, b)
		case *notionapi.BulletedListItemBlock:
			block.BulletedListItem.Children = append(block.BulletedListItem.Children, b)
		case *notionapi.NumberedListItemBlock:
			block.NumberedListItem.Children = append(block.NumberedListItem.Children, b)
		case *notionapi.Heading1Block:
			block.Heading1.Children = append(block.Heading1.Children, b)
		default:
			println("unknown block type: %T\n", block)
			panic("here")
		}
	}
}

func (c *Cursor) Set(node ast.Node, block notionapi.Block) {
	c.m[node] = block
}

func (c *Cursor) Descend(node ast.Node) {
	c.cur = node
}

func (c *Cursor) Ascend() {
	for {
		if c.cur.Parent() != nil {
			c.cur = c.cur.Parent()
			if c.m[c.cur] != nil {
				return
			}
		} else {
			return
		}
	}
}

func NewRenderer(opts ...renderer.Option) renderer.NodeRenderer {
	r := &Renderer{}
	return r
}

// func (r *Renderer) appendBlock(node ast.Node, block notionapi.Block) {
// 	if bl, ok := r.myblocks[node]; !ok {
// 		r.rootBlocks = append(r.rootBlocks, block)
// 	} else {
// 		switch bl := bl.(type) {
// 		case *notionapi.ParagraphBlock:
// 			bl.Paragraph.Children = append(bl.Paragraph.Children, block)
// 		case *notionapi.BulletedListItemBlock:
// 			bl.BulletedListItem.Children = append(bl.BulletedListItem.Children, block)
// 		case *notionapi.NumberedListItemBlock:
// 			bl.NumberedListItem.Children = append(bl.NumberedListItem.Children, block)
// 			// case *notionapi.Heading1Block:
// 			// 	bl.Heading1.Children = append(bl.Heading1.Children, block)
// 		default:
// 			panic("unknown block type: " + fmt.Sprintf("%T", bl))
// 		}
// 	}
// }

func (r *Renderer) AddOptions(...renderer.Option) {}

// func (r *Renderer) Render(w io.Writer, source []byte, n ast.Node) error {
// 	println("rendering ...")
// 	return nil
// }

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
	println("rendering document...", string(node.Text(source)))
	if entering {
		r.c = &Cursor{
			rootBlocks: []notionapi.Block{},
			m:          make(map[ast.Node]notionapi.Block),
			cur:        node,
		}

		// pagination := notionapi.Pagination{}
		// for {
		// 	resp, err := r.client.Block.GetChildren(r.ctx, notionapi.BlockID(r.pageID), &pagination)
		// 	if err != nil {
		// 		return ast.WalkStop, err
		// 	}
		// 	r.blocks = append(r.blocks, resp.Results...)
		// 	if !resp.HasMore {
		// 		break
		// 	}
		// }
	} else {
		_, err := r.client.Block.AppendChildren(r.ctx, notionapi.BlockID(r.pageID), &notionapi.AppendBlockChildrenRequest{
			Children: r.c.rootBlocks,
		})
		if err != nil {
			return ast.WalkStop, err
		}

		// json.NewEncoder(os.Stdout).Encode(r.c.rootBlocks)
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering heading...", entering, string(node.Text(source)))
	if entering {
		block := &notionapi.Heading1Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading1,
			},
			Heading1: notionapi.Heading{
				RichText: []notionapi.RichText{},
			},
		}
		r.c.Set(node, block)
		r.c.Descend(node)
		r.c.AppendBlock(block)
		// r.blocks = append(r.blocks, block)
		// r.appendBlock(node.Parent(), block)
	} else {
		r.c.Ascend()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering blockquote...", entering, string(node.Text(source)))
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering codeblock...", entering, string(node.Text(source)))

	if entering {
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			sb.Write(line.Value(source))
		}
		block := &notionapi.CodeBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeCode,
			},
			Code: notionapi.Code{
				RichText: []notionapi.RichText{
					{
						Text: &notionapi.Text{Content: sb.String()},
					},
				},
				Language: "plain text",
			},
		}
		r.c.Set(node, block)
		r.c.AppendBlock(block)
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering fendecdcodeblock...", entering, string(node.Text(source)))

	n := node.(*ast.FencedCodeBlock)
	if entering {
		var sb strings.Builder
		for i := 0; i < node.Lines().Len(); i++ {
			line := node.Lines().At(i)
			sb.Write(line.Value(source))
		}

		block := &notionapi.CodeBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeCode,
			},
			Code: notionapi.Code{
				RichText: []notionapi.RichText{
					{
						Text: &notionapi.Text{Content: sb.String()},
					},
				},
				Language: supportedLanguageOrPlainText(string(n.Language(source))),
			},
		}
		r.c.Set(node, block)
		r.c.AppendBlock(block)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderHTMLBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderList(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("renderlist", entering)
	return ast.WalkContinue, nil
}

func (r *Renderer) renderListItem(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("renderlistitem", entering)
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
		// if n.FirstChild() == node {
		// 	println("first child")
		// 	r.c.Set(node.Parent(), block)
		// 	r.c.Descend(node)
		// } else {
		// 	println("after child")
		// }

		r.c.Set(node, block)
		r.c.AppendBlock(block, "here")
		r.c.Descend(node)
	} else {
		// if n.LastChild() == node {
		// 	println("getting out of the list")
		// 	r.c.Ascend()
		// }
		r.c.Ascend()
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering paragraph...", string(node.Text(source)))
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
		// r.rootBlocks = append(r.rootBlocks, &block)
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
	return ast.WalkContinue, nil
}

func (r *Renderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering codespan...", entering, string(node.Text(source)))

	if entering {
		var txt string
		for c := node.FirstChild(); c != nil; c = c.NextSibling() {
			segment := c.(*ast.Text).Segment
			txt = txt + string(segment.Value(source))
		}
		r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: txt}, Annotations: &notionapi.Annotations{Code: true}})
		return ast.WalkSkipChildren, nil
	}

	// n := node.(*ast.Text)
	// segment := n.Segment
	//
	// println("rendering text", entering, string(segment.Value(source)))
	// if !entering {
	// 	return ast.WalkContinue, nil
	// }
	//
	// r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
	//
	// println("parent", n.Parent().Kind().String(), "kind", n.Kind().String(), "text", string(segment.Value(source)))
	return ast.WalkContinue, nil
}

func (r *Renderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering emphasis...", entering, string(node.Text(source)))
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
	return ast.WalkContinue, nil
}

func (r *Renderer) renderRawHTML(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderText(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Text)
	segment := n.Segment

	println("rendering text", entering, string(segment.Value(source)))
	if !entering {
		return ast.WalkContinue, nil
	}

	r.c.AppendRichText(&notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})

	println("parent", n.Parent().Kind().String(), "kind", n.Kind().String(), "text", string(segment.Value(source)))

	// switch n.Parent().Kind() {
	// case ast.KindParagraph:
	// 	block := cur.(*notionapi.ParagraphBlock)
	// 	block.Paragraph.RichText = append(block.Paragraph.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
	// case ast.KindEmphasis:
	// 	switch block := cur.(type) {
	// 	case *notionapi.ParagraphBlock:
	// 		block.Paragraph.RichText[len(block.Paragraph.RichText)-1].Text = &notionapi.Text{Content: string(segment.Value(source))}
	// 	case *notionapi.BulletedListItemBlock:
	// 		block.BulletedListItem.RichText[len(block.BulletedListItem.RichText)-1].Text = &notionapi.Text{Content: string(segment.Value(source))}
	// 	case *notionapi.NumberedListItemBlock:
	// 		block.NumberedListItem.RichText[len(block.NumberedListItem.RichText)-1].Text = &notionapi.Text{Content: string(segment.Value(source))}
	// 	}
	// case ast.KindTextBlock:
	// 	if n.Parent().Parent() != nil && n.Parent().Parent().Kind() == ast.KindListItem {
	// 		switch block := cur.(type) {
	// 		case *notionapi.BulletedListItemBlock:
	// 			block.BulletedListItem.RichText = append(block.BulletedListItem.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
	// 		case *notionapi.NumberedListItemBlock:
	// 			block.NumberedListItem.RichText = append(block.NumberedListItem.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
	// 		}
	// 	}
	// case ast.KindList:
	// 	println("rendering text...", string(segment.Value(source)))
	// }
	//
	// switch cur.GetType() {
	// case notionapi.BlockTypeParagraph:
	// 	println("rendering text...", string(segment.Value(source)))
	// 	block := cur.(*notionapi.ParagraphBlock)
	// 	block.Paragraph.RichText = append(block.Paragraph.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
	// }

	return ast.WalkContinue, nil
}

func (r *Renderer) renderString(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering string...", string(node.Text(source)))
	return ast.WalkContinue, nil
}

// func (r *Renderer) curBlock() notionapi.Block {
// 	block := r.blocks[len(r.blocks)-1]
// 	if block.GetHasChildren() {
// 		switch block := block.(type) {
// 		case *notionapi.ParagraphBlock:
// 			return block.Paragraph.Children[len(block.Paragraph.Children)-1]
// 		case *notionapi.BulletedListItemBlock:
// 			return block.BulletedListItem.Children[len(block.BulletedListItem.Children)-1]
// 		case *notionapi.NumberedListItemBlock:
// 			return block.NumberedListItem.Children[len(block.NumberedListItem.Children)-1]
// 		case *notionapi.Heading1Block:
// 			return block.Heading1.Children[len(block.Heading1.Children)-1]
// 		}
// 	}
// 	return block
// }
//
// func (r *Renderer) curRichText() *notionapi.RichText {
// 	rt := *r.curBlockRichText()
// 	return &rt[len(rt)-1]
// }
//
// func (r *Renderer) curBlockRichText() *[]notionapi.RichText {
// 	switch block := r.rootBlocks[len(r.rootBlocks)-1].(type) {
// 	case *notionapi.ParagraphBlock:
// 		return &block.Paragraph.RichText
// 	case *notionapi.BulletedListItemBlock:
// 		return &block.BulletedListItem.RichText
// 	case *notionapi.NumberedListItemBlock:
// 		return &block.NumberedListItem.RichText
// 	case *notionapi.Heading1Block:
// 		return &block.Heading1.RichText
// 	default:
// 		fmt.Printf("unknown block type: %T\n", block)
// 	}
// 	println(len(r.blocks), "blocks")
// 	panic("nil rich text")
// }
//

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
