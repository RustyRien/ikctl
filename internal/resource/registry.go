package resource

import (
	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/resource/core"
	"github.com/electrolux-oss/ik-tui/internal/resource/integrations"
	"github.com/electrolux-oss/ik-tui/internal/resource/resources"
	"github.com/electrolux-oss/ik-tui/internal/resource/templates"
)

type Descriptor = core.Descriptor

type Registry = core.Registry

func DefaultRegistry(c *client.Client) *Registry {
	return core.NewRegistry(
		resources.Descriptor(c),
		templates.Descriptor(c),
		integrations.Descriptor(c),
	)
}

func NewRegistry(descriptors ...*Descriptor) *Registry { return core.NewRegistry(descriptors...) }

func ResolveSortField(descriptor *Descriptor, name string) (string, bool) {
	return core.ResolveSortField(descriptor, name)
}

func ResolveFilter(descriptor *Descriptor, filters map[string]string) (map[string]any, error) {
	return core.ResolveFilter(descriptor, filters)
}
