package main

import (
	"context"

	"github.com/jomei/notionapi"
)

// Config represents the configuration of the renderer.
type Config struct {
	testBlocks *[]notionapi.Block
	Context    context.Context
}

// NewConfig returns a new default Config.
func NewConfig() Config {
	return Config{
		testBlocks: nil,
		Context:    context.Background(),
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

// WithContext allows to set the context that will be used by the renderer
// when production API calls to convert the markdown to Notion blocks.
//
// impl: Because we cannot pass the context elsewhere, our only option is
// to pass it a configuration time.
func WithContext(ctx context.Context) Option {
	return OptionFunc(func(c *Config) {
		c.Context = ctx
	})
}

// WithoutAPI short circuits making any API call and instead dumps the blocks
// in the given slice reference, for further inspection, usually within tests.
func WithoutAPI(testBlocks *[]notionapi.Block) Option {
	return OptionFunc(func(c *Config) {
		c.testBlocks = testBlocks
	})
}
