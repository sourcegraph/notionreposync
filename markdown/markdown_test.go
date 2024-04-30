package markdown_test

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/hexops/autogold/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sourcegraph/notionreposync/markdown"
	"github.com/sourcegraph/notionreposync/renderer/renderertest"
)

func TestProcessor(t *testing.T) {
	filepath.WalkDir("../testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".md" {
			return nil
		}

		t.Run(d.Name(), func(t *testing.T) {
			blockUpdater := &renderertest.MockBlockUpdater{}

			content, err := os.ReadFile(path)
			require.NoError(t, err)

			err = markdown.NewProcessor(context.Background(), blockUpdater).
				ProcessMarkdown(content)
			assert.NoError(t, err)

			autogold.ExpectFile(t, blockUpdater.GetAddedBlocks(), autogold.Dir("golden"))
		})

		return nil
	})
}
