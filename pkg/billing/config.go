package billing

import (
	"context"
	"sync"
)

// ConfigProvider implements billing.Provider using static per-tier limits
// from YAML/JSON config. It does not call any external service.
//
// Intended for self-hosted multi-tenant deployments where the operator wants
// quotas without payment processing — e.g. an internal "automation platform"
// where the SRE team caps each business unit's monthly executions.
type ConfigProvider struct {
	mu           sync.RWMutex
	tierLimits   map[Tier]map[string]int64 // tier → kind → monthly limit
	usage        map[string]map[string]int64 // workspaceID → kind → month-to-date
	subscription map[string]*Subscription
	// PeriodResetUnix is the unix-seconds timestamp at which usage counters
	// reset. Typically the first of the month. Embedders should call
	// ResetUsage() at that time.
	PeriodResetUnix int64
}

// NewConfigProvider returns a ConfigProvider with the given tier limits.
//
// Example:
//
//	cp := NewConfigProvider(map[Tier]map[string]int64{
//	    TierFree: { "execution": 10_000 },
//	    TierPro:  { "execution": 250_000 },
//	})
func NewConfigProvider(tierLimits map[Tier]map[string]int64) *ConfigProvider {
	if tierLimits == nil {
		tierLimits = map[Tier]map[string]int64{}
	}
	return &ConfigProvider{
		tierLimits:   tierLimits,
		usage:        map[string]map[string]int64{},
		subscription: map[string]*Subscription{},
	}
}

// Subscribe records the workspace's tier.
func (cp *ConfigProvider) Subscribe(ctx context.Context, workspaceID string, tier Tier) (*Subscription, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	sub := &Subscription{WorkspaceID: workspaceID, Tier: tier, Active: true, Seats: 1}
	cp.subscription[workspaceID] = sub
	return sub, nil
}

// Cancel marks the workspace as inactive; tier falls back to TierFree.
func (cp *ConfigProvider) Cancel(ctx context.Context, workspaceID string) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if sub, ok := cp.subscription[workspaceID]; ok {
		sub.Active = false
		sub.Tier = TierFree
	}
	return nil
}

// ReportUsage increments the workspace's month-to-date counter for the given
// kind.
func (cp *ConfigProvider) ReportUsage(ctx context.Context, event UsageEvent) error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	ws := cp.usage[event.WorkspaceID]
	if ws == nil {
		ws = map[string]int64{}
		cp.usage[event.WorkspaceID] = ws
	}
	ws[event.Kind] += event.Quantity
	return nil
}

// CheckQuota compares workspace usage against tier limits.
func (cp *ConfigProvider) CheckQuota(ctx context.Context, req QuotaRequest) (QuotaDecision, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	sub := cp.subscription[req.WorkspaceID]
	tier := TierFree
	if sub != nil && sub.Active {
		tier = sub.Tier
	}
	tierLimits := cp.tierLimits[tier]
	limit, hasLimit := tierLimits[req.Kind]
	if !hasLimit {
		return QuotaAllow, nil
	}
	used := int64(0)
	if ws := cp.usage[req.WorkspaceID]; ws != nil {
		used = ws[req.Kind]
	}
	projected := used + req.Quantity
	switch {
	case projected > limit:
		return QuotaDeny, nil
	case projected > (limit*4)/5: // 80% threshold
		return QuotaWarn, nil
	default:
		return QuotaAllow, nil
	}
}

// GetSubscription returns the workspace's subscription, or a free-tier
// default.
func (cp *ConfigProvider) GetSubscription(ctx context.Context, workspaceID string) (*Subscription, error) {
	cp.mu.RLock()
	defer cp.mu.RUnlock()
	if sub, ok := cp.subscription[workspaceID]; ok {
		return sub, nil
	}
	return &Subscription{WorkspaceID: workspaceID, Tier: TierFree, Active: true, Seats: 1}, nil
}

// ResetUsage clears all month-to-date counters. Embedders call this at the
// start of each billing period.
func (cp *ConfigProvider) ResetUsage() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.usage = map[string]map[string]int64{}
}

var _ Provider = (*ConfigProvider)(nil)
