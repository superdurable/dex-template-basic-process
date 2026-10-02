// Copyright (c) 2026 Super Durable
// SPDX-License-Identifier: MIT

package process

import (
	"time"

	"github.com/superdurable/dex/sdk-go/dex"
)

// ExampleFlow is the initial design. Replace it with the requested business Flow.
type ExampleFlow struct{ dex.FlowDefaults }

func (ExampleFlow) GetSteps() []dex.StepDef {
	return []dex.StepDef{dex.DefineStartStep(ExampleStep{})}
}

func (ExampleFlow) GetPersistenceSchema() dex.PersistenceSchema {
	return dex.PersistenceSchema{}
}

func (flow ExampleFlow) GetRPCs() []dex.RPCDef {
	return []dex.RPCDef{
		dex.DefineRPC(flow.GetDexSummary, nil),
		dex.DefineRPC(flow.GetDexDisplay, nil),
	}
}

func (ExampleFlow) GetDexSummary(_ dex.Context, _ dex.None) (*dex.RPCResult[map[string]any], error) {
	return &dex.RPCResult[map[string]any]{Output: map[string]any{}}, nil
}

func (ExampleFlow) GetDexDisplay(_ dex.Context, _ dex.None) (*dex.RPCResult[map[string]any], error) {
	return &dex.RPCResult[map[string]any]{Output: map[string]any{}}, nil
}

// dex:group group-id:example group-label:"Example"
// dex:explanation text:"Complete this example. Replace it with your business steps."
type ExampleStep struct {
	dex.StepDefaultsNoWaitFor[dex.None]
}

func (ExampleStep) GetStepOptions() *dex.StepOptions {
	return &dex.StepOptions{
		ExecuteMethodTimeout: 10 * time.Second,
		ExecuteRetry:         &dex.RetryPolicy{MaximumAttempts: 1},
		ExecuteDurability:    dex.StepDurabilitySync,
	}
}

func (ExampleStep) Execute(_ dex.Context, _ dex.None) (*dex.StepDecision, error) {
	return dex.GracefulComplete(nil), nil
}
