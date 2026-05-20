package billing

import "context"

// NullProvider is a no-op billing provider that allows everything. It is the
// default for self-hosted deployments that have no billing requirement.
//
// All methods return success; CheckQuota always returns QuotaAllow.
type NullProvider struct{}

// NewNullProvider returns a Provider that allows everything.
func NewNullProvider() *NullProvider { return &NullProvider{} }

// Subscribe records the subscription in memory (and forgets it).
func (NullProvider) Subscribe(ctx context.Context, workspaceID string, tier Tier) (*Subscription, error) {
	return &Subscription{WorkspaceID: workspaceID, Tier: tier, Active: true, Seats: 1}, nil
}

// Cancel always succeeds.
func (NullProvider) Cancel(ctx context.Context, workspaceID string) error { return nil }

// ReportUsage discards the event.
func (NullProvider) ReportUsage(ctx context.Context, event UsageEvent) error { return nil }

// CheckQuota always allows.
func (NullProvider) CheckQuota(ctx context.Context, req QuotaRequest) (QuotaDecision, error) {
	return QuotaAllow, nil
}

// GetSubscription returns a synthetic enterprise-tier subscription.
func (NullProvider) GetSubscription(ctx context.Context, workspaceID string) (*Subscription, error) {
	return &Subscription{WorkspaceID: workspaceID, Tier: TierEnterprise, Active: true, Seats: 0}, nil
}

var _ Provider = (*NullProvider)(nil)
