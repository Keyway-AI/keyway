package attribution

import (
	"bytes"
	"context"

	"github.com/Keyway-AI/keyway/internal/model"
)

// Chain tries each attributor in order and returns the first that binds the
// change to a cause; if none do, the result is Unattributed. Order encodes
// precedence — e.g. a git commit is more specific than a deploy annotation.
type Chain struct {
	attributors []Attributor
}

// NewChain constructs a chain from the given attributors in precedence order.
func NewChain(attributors ...Attributor) *Chain { return &Chain{attributors: attributors} }

var _ Attributor = (*Chain)(nil)

// Attribute returns the first non-unattributed result.
func (c *Chain) Attribute(ctx context.Context, ev model.ChangeEvent) (*model.Attribution, error) {
	for _, a := range c.attributors {
		r, err := a.Attribute(ctx, ev)
		if err != nil {
			continue
		}
		if r != nil && r.Kind != "unattributed" {
			return r, nil
		}
	}
	return Unattributed(), nil
}

// newReader avoids importing bytes at each call site.
func newReader(b []byte) *bytes.Reader { return bytes.NewReader(b) }
