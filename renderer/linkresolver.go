package renderer

// LinkResolver can be implemented to modify Markdown links.
type LinkResolver interface {
	// ResolveLink accepts a link and returns it as-is or modified as desired,
	// for example to resolve an appropriate absolute link to the relevant
	// resource (e.g. another Notion document or a blob view).
	ResolveLink(link string) (string, error)
}

// noopLinkResolver returns all links as-is and unmodified.
type noopLinkResolver struct{}

func (noopLinkResolver) ResolveLink(link string) (string, error) { return link, nil }
