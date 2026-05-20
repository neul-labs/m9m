package quota

import (
	"context"
	"sync"
)

// ConfigEnforcer is an in-memory Enforcer driven by a static per-workspace
// (or per-tier) limit map. Intended for self-hosted multi-tenant
// deployments where the operator wants quotas without a billing backend.
//
// Cloud-mode worker uses a billing-aware Enforcer that reads from
// billing.Provider; ConfigEnforcer is the OSS fallback.
type ConfigEnforcer struct {
	mu         sync.RWMutex
	limits     map[string]map[string]int64 // workspaceID → kind → monthly limit
	defaults   map[string]int64            // kind → default monthly limit when workspace has no specific entry
	usage      map[string]map[string]int64 // workspaceID → kind → month-to-date
	warnAtFrac float64
}

// NewConfigEnforcer returns an Enforcer driven by static per-workspace limits.
//
// defaults is the limit applied when a workspace has no specific entry in
// limits — useful for "every workspace gets 10k executions" deployments.
// warnAtFrac (e.g. 0.8) emits a Warn decision when usage exceeds that
// fraction of the limit. Zero or negative disables warnings.
func NewConfigEnforcer(limits map[string]map[string]int64, defaults map[string]int64, warnAtFrac float64) *ConfigEnforcer {
	if limits == nil {
		limits = map[string]map[string]int64{}
	}
	if defaults == nil {
		defaults = map[string]int64{}
	}
	return &ConfigEnforcer{
		limits:     limits,
		defaults:   defaults,
		usage:      map[string]map[string]int64{},
		warnAtFrac: warnAtFrac,
	}
}

func (c *ConfigEnforcer) limit(workspaceID, kind string) (int64, bool) {
	if ws, ok := c.limits[workspaceID]; ok {
		if v, ok := ws[kind]; ok {
			return v, true
		}
	}
	if v, ok := c.defaults[kind]; ok {
		return v, true
	}
	return 0, false
}

// Check compares projected usage against the configured limit.
func (c *ConfigEnforcer) Check(ctx context.Context, req Request) (Decision, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	limit, hasLimit := c.limit(req.WorkspaceID, req.Kind)
	if !hasLimit {
		return Allow, nil
	}
	used := int64(0)
	if ws := c.usage[req.WorkspaceID]; ws != nil {
		used = ws[req.Kind]
	}
	projected := used + req.Quantity
	if projected > limit {
		return Deny, nil
	}
	if c.warnAtFrac > 0 && c.warnAtFrac < 1 {
		threshold := int64(float64(limit) * c.warnAtFrac)
		if projected >= threshold {
			return Warn, nil
		}
	}
	return Allow, nil
}

// Observe increments the workspace's month-to-date counter.
func (c *ConfigEnforcer) Observe(ctx context.Context, req Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	ws := c.usage[req.WorkspaceID]
	if ws == nil {
		ws = map[string]int64{}
		c.usage[req.WorkspaceID] = ws
	}
	ws[req.Kind] += req.Quantity
	return nil
}

// ResetUsage clears all month-to-date counters. Embedders call this at the
// start of each billing period.
func (c *ConfigEnforcer) ResetUsage() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.usage = map[string]map[string]int64{}
}

var _ Enforcer = (*ConfigEnforcer)(nil)
