package resource

import (
	"github.com/electrolux-oss/ik-tui/internal/client"
	"github.com/electrolux-oss/ik-tui/internal/resource/core"
	"github.com/electrolux-oss/ik-tui/internal/resource/integrations"
	"github.com/electrolux-oss/ik-tui/internal/resource/resources"
	"github.com/electrolux-oss/ik-tui/internal/resource/secrets"
	"github.com/electrolux-oss/ik-tui/internal/resource/source_code_versions"
	"github.com/electrolux-oss/ik-tui/internal/resource/source_codes"
	"github.com/electrolux-oss/ik-tui/internal/resource/storages"
	"github.com/electrolux-oss/ik-tui/internal/resource/templates"
	"github.com/electrolux-oss/ik-tui/internal/resource/workers"
)

type Descriptor = core.Descriptor

type Registry = core.Registry

func DefaultRegistry(c *client.Client) *Registry {
	return core.NewRegistry(
		resources.Descriptor(c),
		source_codes.Descriptor(c),
		source_code_versions.Descriptor(c),
		templates.Descriptor(c),
		secrets.Descriptor(c),
		integrations.Descriptor(c),
		storages.Descriptor(c),
		workers.Descriptor(c),
	)
}

func NewRegistry(descriptors ...*Descriptor) *Registry { return core.NewRegistry(descriptors...) }

func ResolveSortField(descriptor *Descriptor, name string) (string, bool) {
	return core.ResolveSortField(descriptor, name)
}

func ResolveFilter(descriptor *Descriptor, filters map[string]string) (map[string]any, error) {
	return core.ResolveFilter(descriptor, filters)
}
