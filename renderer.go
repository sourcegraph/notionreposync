package main

import (
	"context"

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

	page   *notionapi.Page
	blocks []notionapi.Block
	idx    int
}

func NewRenderer(opts ...renderer.Option) renderer.NodeRenderer {
	r := &Renderer{}
	return r
}

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
			Children: r.blocks,
		})
		if err != nil {
			return ast.WalkStop, err
		}
	}

	return ast.WalkContinue, nil
}

func (r *Renderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering heading...", string(node.Text(source)))
	if entering {
		block := notionapi.Heading1Block{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeHeading1,
			},
			Heading1: notionapi.Heading{
				RichText: []notionapi.RichText{
					{Text: &notionapi.Text{Content: string(node.Text(source))}},
				},
			},
		}
		r.blocks = append(r.blocks, block)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderBlockquote(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderFencedCodeBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
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
		r.blocks = append(r.blocks, block)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderParagraph(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering paragraph...", string(node.Text(source)))
	if entering {
		block := notionapi.ParagraphBlock{
			BasicBlock: notionapi.BasicBlock{
				Object: notionapi.ObjectTypeBlock,
				Type:   notionapi.BlockTypeParagraph,
			},
			Paragraph: notionapi.Paragraph{},
		}
		r.blocks = append(r.blocks, &block)
	}
	return ast.WalkContinue, nil
}

func (r *Renderer) renderTextBlock(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	// cur := r.blocks[len(r.blocks)-1]

	return ast.WalkContinue, nil
}

func (r *Renderer) renderThematicBreak(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderAutoLink(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderCodeSpan(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	return ast.WalkContinue, nil
}

func (r *Renderer) renderEmphasis(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	println("rendering emphasis...", string(node.Text(source)))
	n := node.(*ast.Emphasis)

	if entering {
		cur := r.blocks[len(r.blocks)-1]
		switch cur.GetType() {
		case notionapi.BlockTypeParagraph:
			block := cur.(*notionapi.ParagraphBlock)
			annotations := &notionapi.Annotations{}
			if n.Level == 1 {
				annotations.Italic = true
			} else {
				annotations.Bold = true
			}
			rt := notionapi.RichText{Annotations: annotations}
			block.Paragraph.RichText = append(block.Paragraph.RichText, rt)
		}
	} else {

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
	if !entering {
		return ast.WalkContinue, nil
	}

	cur := r.blocks[len(r.blocks)-1]
	n := node.(*ast.Text)
	segment := n.Segment

	println("parent", n.Parent().Kind().String(), "kind", n.Kind().String(), "text", string(segment.Value(source)))

	switch n.Parent().Kind() {
	case ast.KindParagraph:
		block := cur.(*notionapi.ParagraphBlock)
		block.Paragraph.RichText = append(block.Paragraph.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
	case ast.KindEmphasis:
		switch block := cur.(type) {
		case *notionapi.ParagraphBlock:
			block.Paragraph.RichText[len(block.Paragraph.RichText)-1].Text = &notionapi.Text{Content: string(segment.Value(source))}
		case *notionapi.BulletedListItemBlock:
			block.BulletedListItem.RichText[len(block.BulletedListItem.RichText)-1].Text = &notionapi.Text{Content: string(segment.Value(source))}
		case *notionapi.NumberedListItemBlock:
			block.NumberedListItem.RichText[len(block.NumberedListItem.RichText)-1].Text = &notionapi.Text{Content: string(segment.Value(source))}
		}
	case ast.KindTextBlock:
		if n.Parent().Parent() != nil && n.Parent().Parent().Kind() == ast.KindListItem {
			switch block := cur.(type) {
			case *notionapi.BulletedListItemBlock:
				block.BulletedListItem.RichText = append(block.BulletedListItem.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
			case *notionapi.NumberedListItemBlock:
				block.NumberedListItem.RichText = append(block.NumberedListItem.RichText, notionapi.RichText{Text: &notionapi.Text{Content: string(segment.Value(source))}})
			}
		}
	case ast.KindList:
		println("rendering text...", string(segment.Value(source)))
	}

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

func (r *Renderer) curRichText() *notionapi.RichText {
	switch block := r.blocks[len(r.blocks)-1].(type) {
	case *notionapi.ParagraphBlock:
		return &block.Paragraph.RichText[len(block.Paragraph.RichText)-1]
	case *notionapi.BulletedListItemBlock:
		return &block.BulletedListItem.RichText[len(block.BulletedListItem.RichText)-1]
	case *notionapi.NumberedListItemBlock:
		return &block.NumberedListItem.RichText[len(block.NumberedListItem.RichText)-1]
	case *notionapi.Heading1Block:
		return &block.Heading1.RichText[len(block.Heading1.RichText)-1]
	}
	return nil
}
