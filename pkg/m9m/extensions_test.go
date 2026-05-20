package m9m

import (
	"context"
	"testing"

	"github.com/neul-labs/m9m/pkg/billing"
	"github.com/neul-labs/m9m/pkg/blob"
	"github.com/neul-labs/m9m/pkg/observability"
	"github.com/neul-labs/m9m/pkg/quota"
	"github.com/neul-labs/m9m/pkg/snapshot"
)

func TestDefaultExtensions_NonNil(t *testing.T) {
	e := New()
	if e.Billing() == nil {
		t.Error("default Billing is nil")
	}
	if e.Snapshot() == nil {
		t.Error("default Snapshot is nil")
	}
	if e.Blob() == nil {
		t.Error("default Blob is nil")
	}
	if e.Quota() == nil {
		t.Error("default Quota is nil")
	}
	if e.Observability() == nil {
		t.Error("default Observability is nil")
	}
}

func TestDefaultBilling_AllowsEverything(t *testing.T) {
	e := New()
	dec, err := e.Billing().CheckQuota(context.Background(), billing.QuotaRequest{
		WorkspaceID: "ws", Kind: "execution", Quantity: 999_999_999,
	})
	if err != nil {
		t.Fatal(err)
	}
	if dec != billing.QuotaAllow {
		t.Errorf("default billing denied request: %v", dec)
	}
}

func TestWithBilling_Overrides(t *testing.T) {
	custom := billing.NewConfigProvider(map[billing.Tier]map[string]int64{
		billing.TierFree: {"execution": 5},
	})
	e := NewWithOptions(WithBilling(custom))
	if e.Billing() != custom {
		t.Error("WithBilling did not override default provider")
	}
}

func TestWithSnapshot_Overrides(t *testing.T) {
	custom, err := snapshot.NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := NewWithOptions(WithSnapshot(custom))
	if e.Snapshot() != custom {
		t.Error("WithSnapshot did not override default sink")
	}
}

func TestWithBlob_Overrides(t *testing.T) {
	custom, err := blob.NewLocalFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	e := NewWithOptions(WithBlob(custom))
	if e.Blob() != custom {
		t.Error("WithBlob did not override default store")
	}
}

func TestWithQuota_Overrides(t *testing.T) {
	custom := quota.NewConfigEnforcer(nil, map[string]int64{"execution": 10}, 0.8)
	e := NewWithOptions(WithQuota(custom))
	if e.Quota() != custom {
		t.Error("WithQuota did not override default enforcer")
	}
}

func TestWithObservability_Overrides(t *testing.T) {
	custom := observability.NewNullEmitter() // distinct instance
	e := NewWithOptions(WithObservability(custom))
	if e.Observability() != custom {
		t.Error("WithObservability did not override default emitter")
	}
}

func TestWithNil_KeepsDefault(t *testing.T) {
	// Passing nil to a With* option should be a no-op, leaving the default
	// in place. This guards against accidental nil-dereference panics from
	// embedders that haven't constructed their extension yet.
	e := NewWithOptions(
		WithBilling(nil),
		WithSnapshot(nil),
		WithBlob(nil),
		WithQuota(nil),
		WithObservability(nil),
	)
	if e.Billing() == nil || e.Snapshot() == nil || e.Blob() == nil ||
		e.Quota() == nil || e.Observability() == nil {
		t.Error("nil option clobbered default extension")
	}
}
