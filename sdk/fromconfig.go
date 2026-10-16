package sdk

import "github.com/rustyrien/ikctl/sdk/config"

// NewFromConfig creates a Client using the ikctl configuration file and the
// IKCTL_* environment variables, with explicit overrides taking precedence.
// Extra options are applied after the configuration.
func NewFromConfig(o config.Overrides, opts ...Option) (*Client, error) {
	r, err := config.Resolve(o)
	if err != nil {
		return nil, err
	}
	base := []Option{
		WithServer(r.Server),
		WithToken(r.Token),
		WithInsecureSkipTLSVerify(r.InsecureSkipTLSVerify),
	}
	return New(append(base, opts...)...)
}
