package quota

import "context"

// Unlimited is an Enforcer that allows everything. Default for self-hosted
// deployments without quota requirements.
type Unlimited struct{}

// NewUnlimited returns an Enforcer that always allows.
func NewUnlimited() *Unlimited { return &Unlimited{} }

func (Unlimited) Check(ctx context.Context, req Request) (Decision, error) {
	return Allow, nil
}

func (Unlimited) Observe(ctx context.Context, req Request) error { return nil }

var _ Enforcer = (*Unlimited)(nil)
