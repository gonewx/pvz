package managers

import (
	"testing"

	"github.com/gonewx/pvz/pkg/ecs"
)

// TestDefaultSystemOptions verifies that DefaultSystemOptions returns
// expected default values for standard gameplay.
func TestDefaultSystemOptions(t *testing.T) {
	opts := DefaultSystemOptions()

	// Core systems should be enabled
	if !opts.EnableCamera {
		t.Error("EnableCamera should be true by default")
	}
	if !opts.EnableZombieGroan {
		t.Error("EnableZombieGroan should be true by default")
	}
	if !opts.EnableInput {
		t.Error("EnableInput should be true by default")
	}
	if !opts.EnableButton {
		t.Error("EnableButton should be true by default")
	}
	if !opts.EnablePlantPreview {
		t.Error("EnablePlantPreview should be true by default")
	}

	// Level-specific systems should be disabled by default
	if opts.EnableTutorial {
		t.Error("EnableTutorial should be false by default")
	}
	if opts.EnableConveyorBelt {
		t.Error("EnableConveyorBelt should be false by default")
	}
	if opts.EnableBowlingNut {
		t.Error("EnableBowlingNut should be false by default")
	}
}

// TestVerifyGameplayOptions verifies that VerifyGameplayOptions returns
// minimal system options for the verify_gameplay tool.
func TestVerifyGameplayOptions(t *testing.T) {
	opts := VerifyGameplayOptions()

	// Most systems should be disabled for verify_gameplay
	if opts.EnableCamera {
		t.Error("EnableCamera should be false for verify_gameplay")
	}
	if opts.EnableZombieGroan {
		t.Error("EnableZombieGroan should be false for verify_gameplay")
	}
	// EnableInput is true for verify_gameplay to support planting logic
	if !opts.EnableInput {
		t.Error("EnableInput should be true for verify_gameplay (needed for planting logic)")
	}
	if opts.EnableButton {
		t.Error("EnableButton should be false for verify_gameplay")
	}

	// PlantPreview should be enabled for testing
	if !opts.EnablePlantPreview {
		t.Error("EnablePlantPreview should be true for verify_gameplay")
	}
}

// TestSystemDependencies verifies that SystemDependencies can be created
// with minimal required fields.
func TestSystemDependencies(t *testing.T) {
	em := ecs.NewEntityManager()

	deps := SystemDependencies{
		EntityManager: em,
		EnabledLanes:  []int{1, 2, 3, 4, 5},
	}

	if deps.EntityManager == nil {
		t.Error("EntityManager should not be nil")
	}
	if len(deps.EnabledLanes) != 5 {
		t.Errorf("Expected 5 enabled lanes, got %d", len(deps.EnabledLanes))
	}
}

// TestSystemManagerGetters verifies that all getter methods return
// the expected system instances (nil or non-nil based on options).
func TestSystemManagerGetters(t *testing.T) {
	// Skip this test in environments without resources
	// The full integration test would require loading game assets
	t.Skip("Full SystemManager test requires game assets - covered by verify_gameplay")
}
