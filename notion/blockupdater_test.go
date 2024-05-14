package notion

import (
	"testing"

	"github.com/jomei/notionapi"
)

func Test_getDepth(t *testing.T) {
	block := notionapi.BulletedListItemBlock{
		BasicBlock: notionapi.BasicBlock{
			Object: notionapi.ObjectTypeBlock,
			Type:   notionapi.BlockTypeBulletedListItem,
		},
		BulletedListItem: notionapi.ListItem{
			Children: []notionapi.Block{
				&notionapi.BulletedListItemBlock{
					BasicBlock: notionapi.BasicBlock{
						Object: notionapi.ObjectTypeBlock,
						Type:   notionapi.BlockTypeBulletedListItem,
					},
					BulletedListItem: notionapi.ListItem{
						Children: []notionapi.Block{
							&notionapi.BulletedListItemBlock{
								BasicBlock: notionapi.BasicBlock{
									Object: notionapi.ObjectTypeBlock,
									Type:   notionapi.BlockTypeBulletedListItem,
								},
								BulletedListItem: notionapi.ListItem{
									Children: nil,
								},
							},
						},
					},
				},
			},
		},
	}

	want := 2
	got := getDepth(block)

	if got != want {
		t.Logf("want: %d, got: %d", want, got)
		t.Fail()
	}
}
