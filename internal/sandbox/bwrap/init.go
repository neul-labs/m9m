package bwrap

import (
	"github.com/neul-labs/m9m/internal/sandbox"
)

func init() {
	// Register the bubblewrap sandbox with the factory
	sandbox.RegisterSandbox(sandbox.SandboxTypeBubblewrap, func() (sandbox.Sandbox, error) {
		return New()
	})
}
