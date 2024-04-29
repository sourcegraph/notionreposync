package renderer

import (
	"context"
)

type LinkResolver interface {
	// ResolveLink accepts a relative link and resolves an appropriate absolute
	// link to the relevant resource (e.g. another Notion document or a blob view).
	ResolveLink(link string) (string, error)
}

type noopLinkResolver struct{}

func (noopLinkResolver) ResolveLink(link string) (string, error) { return link, nil }

// Config represents the configuration of the renderer.
type Config struct {
	ctx   context.Context
	links LinkResolver
}

// NewConfig returns a new default Config.
func NewConfig(ctx context.Context) Config {
	return Config{
		ctx:   ctx,
		links: noopLinkResolver{},
	}
}

type Option interface {
	SetConfig(*Config)
}

type OptionFunc func(*Config)

func (o OptionFunc) SetConfig(c *Config) {
	o(c)
}

// WithConfig allows to directly set a Config.
func WithConfig(config *Config) Option {
	return OptionFunc(func(c *Config) {
		*c = *config
	})
}

// WithLinkResolver configures a LinkResolver to use. Otherwise, a default no-op
// one is used that uses links as-is.
func WithLinkResolver(links LinkResolver) Option {
	return OptionFunc(func(c *Config) {
		c.links = links
	})
}
