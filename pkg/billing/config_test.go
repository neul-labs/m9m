package billing

import (
	"context"
	"testing"
)

func TestConfigProvider_QuotaLifecycle(t *testing.T) {
	cp := NewConfigProvider(map[Tier]map[string]int64{
		TierFree: {"execution": 100},
		TierPro:  {"execution": 1000},
	})
	ctx := context.Background()

	// New workspace defaults to free tier.
	sub, err := cp.GetSubscription(ctx, "ws1")
	if err != nil {
		t.Fatal(err)
	}
	if sub.Tier != TierFree {
		t.Errorf("default tier = %s, want free", sub.Tier)
	}

	// Within quota: allow.
	dec, _ := cp.CheckQuota(ctx, QuotaRequest{WorkspaceID: "ws1", Kind: "execution", Quantity: 1})
	if dec != QuotaAllow {
		t.Errorf("first call decision = %v, want Allow", dec)
	}

	// Push past 80% to get Warn.
	for i := 0; i < 81; i++ {
		_ = cp.ReportUsage(ctx, UsageEvent{WorkspaceID: "ws1", Kind: "execution", Quantity: 1})
	}
	dec, _ = cp.CheckQuota(ctx, QuotaRequest{WorkspaceID: "ws1", Kind: "execution", Quantity: 1})
	if dec != QuotaWarn {
		t.Errorf("at 81/100 + 1 decision = %v, want Warn", dec)
	}

	// Push past limit to get Deny.
	for i := 0; i < 20; i++ {
		_ = cp.ReportUsage(ctx, UsageEvent{WorkspaceID: "ws1", Kind: "execution", Quantity: 1})
	}
	dec, _ = cp.CheckQuota(ctx, QuotaRequest{WorkspaceID: "ws1", Kind: "execution", Quantity: 1})
	if dec != QuotaDeny {
		t.Errorf("at 101/100 + 1 decision = %v, want Deny", dec)
	}

	// Upgrade tier → new limits apply.
	_, _ = cp.Subscribe(ctx, "ws1", TierPro)
	dec, _ = cp.CheckQuota(ctx, QuotaRequest{WorkspaceID: "ws1", Kind: "execution", Quantity: 1})
	if dec != QuotaAllow {
		t.Errorf("after upgrade decision = %v, want Allow", dec)
	}

	// Reset clears usage.
	cp.ResetUsage()
	dec, _ = cp.CheckQuota(ctx, QuotaRequest{WorkspaceID: "ws1", Kind: "execution", Quantity: 50})
	if dec != QuotaAllow {
		t.Errorf("after reset decision = %v, want Allow", dec)
	}
}

func TestNullProvider_AlwaysAllow(t *testing.T) {
	np := NewNullProvider()
	dec, err := np.CheckQuota(context.Background(), QuotaRequest{WorkspaceID: "x", Kind: "execution", Quantity: 999_999_999})
	if err != nil {
		t.Fatal(err)
	}
	if dec != QuotaAllow {
		t.Errorf("null provider denied a request")
	}
}
