package quota

import (
	"context"
	"testing"
)

func TestConfigEnforcer_DefaultLimit(t *testing.T) {
	e := NewConfigEnforcer(nil, map[string]int64{"execution": 10}, 0.8)
	ctx := context.Background()

	// Below 80%: allow.
	for i := 0; i < 7; i++ {
		dec, _ := e.Check(ctx, Request{WorkspaceID: "ws", Kind: "execution", Quantity: 1})
		if dec != Allow {
			t.Errorf("iter %d: dec = %v, want Allow", i, dec)
		}
		_ = e.Observe(ctx, Request{WorkspaceID: "ws", Kind: "execution", Quantity: 1})
	}

	// At 80% threshold (8/10 + 1 = 9 projected, threshold = 8): warn.
	dec, _ := e.Check(ctx, Request{WorkspaceID: "ws", Kind: "execution", Quantity: 1})
	if dec != Warn {
		t.Errorf("at threshold dec = %v, want Warn", dec)
	}
	_ = e.Observe(ctx, Request{WorkspaceID: "ws", Kind: "execution", Quantity: 1})

	// Past limit: deny.
	_ = e.Observe(ctx, Request{WorkspaceID: "ws", Kind: "execution", Quantity: 5})
	dec, _ = e.Check(ctx, Request{WorkspaceID: "ws", Kind: "execution", Quantity: 1})
	if dec != Deny {
		t.Errorf("past limit dec = %v, want Deny", dec)
	}
}

func TestConfigEnforcer_PerWorkspaceOverride(t *testing.T) {
	e := NewConfigEnforcer(
		map[string]map[string]int64{
			"big-customer": {"execution": 1_000_000},
		},
		map[string]int64{"execution": 10},
		0,
	)
	ctx := context.Background()
	dec, _ := e.Check(ctx, Request{WorkspaceID: "big-customer", Kind: "execution", Quantity: 50_000})
	if dec != Allow {
		t.Errorf("big-customer dec = %v, want Allow", dec)
	}
	dec, _ = e.Check(ctx, Request{WorkspaceID: "small", Kind: "execution", Quantity: 50_000})
	if dec != Deny {
		t.Errorf("small dec = %v, want Deny", dec)
	}
}

func TestUnlimited_AlwaysAllows(t *testing.T) {
	u := NewUnlimited()
	dec, _ := u.Check(context.Background(), Request{WorkspaceID: "x", Kind: "execution", Quantity: 999_999_999})
	if dec != Allow {
		t.Errorf("unlimited denied")
	}
}
