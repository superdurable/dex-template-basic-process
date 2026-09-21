package process

import (
	"testing"

	"github.com/superdurable/dex/sdk-go/dex"
)

func TestBasicProcessDefinitionRegistersEveryDurablePrimitive(t *testing.T) {
	registry, err := dex.NewRegistry([]dex.Flow{BasicProcess})
	if err != nil {
		t.Fatalf("register Basic Process Flow: %v", err)
	}
	if registry == nil {
		t.Fatal("registry is nil")
	}
	if got := len(BasicProcess.GetSteps()); got != 6 {
		t.Fatalf("step count = %d, want 6", got)
	}
	if got := len(BasicProcess.GetPersistenceSchema().Attributes); got != 3 {
		t.Fatalf("attribute count = %d, want 3", got)
	}
	if got := len(BasicProcess.GetPersistenceSchema().Channels); got != 1 {
		t.Fatalf("channel count = %d, want 1", got)
	}
	if got := len(BasicProcess.GetRPCs()); got != 4 {
		t.Fatalf("RPC count = %d, want 4", got)
	}
}
