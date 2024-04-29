package repository

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sourcegraph/notionreposync/renderer"
)

type LinkResolver struct {
	repo *Repo

	// basepath is the path of the current file we're rendering, for the
	// sole purpose of handling relative links.
	basepath string

	notionPageID string
}

var _ renderer.LinkResolver = (*LinkResolver)(nil)

func NewLinkResolver(repo *Repo, basepath string, notionPageID string) *LinkResolver {
	return &LinkResolver{
		repo:         repo,
		basepath:     basepath,
		notionPageID: notionPageID,
	}
}

func (r *LinkResolver) ResolveLink(link string) (string, error) {
	// If this is an external link, we just pass through.
	if strings.HasPrefix(link, "http") {
		return link, nil
	}

	if filepath.Ext(link) != ".md" {
		// https://sourcegraph.com/github.com/sourcegraph/sourcegraph/-/blob/cmd/worker/main.go
		return fmt.Sprintf("https://sourcegraph.com/%s/-/blob/%s", r.repo.GitHub, filepath.Clean(link)), nil
	}

	link, _, err := parseLinkAndAnchor(link)
	if err != nil {
		return "", err
	}

	if link == "" {
		return fmt.Sprintf("/%s", strings.ReplaceAll(string(r.notionPageID), "-", "")), nil
	}

	d := r.repo.FindDocument(filepath.Join(r.basepath, link))
	return fmt.Sprintf("/%s", strings.ReplaceAll(string(d.ID), "-", "")), nil
}

var anchorRe = regexp.MustCompile(`([^#]*)#(.*)?`)

func parseLinkAndAnchor(link string) (string, string, error) {
	matches := anchorRe.FindStringSubmatch(link)
	switch len(matches) {
	case 0:
		return link, "", nil
	case 3:
		return matches[1], matches[2], nil
	}
	return "", "", fmt.Errorf("invalid link: %q", link)
}