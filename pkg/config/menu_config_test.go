package config

import (
	"testing"
)

// TestCompareLevels tests the level comparison function.
func TestCompareLevels(t *testing.T) {
	tests := []struct {
		name     string
		level1   string
		level2   string
		expected int
	}{
		// Equal levels
		{"Equal levels", "1-1", "1-1", 0},
		{"Equal levels (3-5)", "3-5", "3-5", 0},
		{"Equal levels (5-10)", "5-10", "5-10", 0},

		// Different chapters
		{"Chapter 1 < Chapter 2", "1-1", "2-1", -1},
		{"Chapter 2 > Chapter 1", "2-1", "1-1", 1},
		{"Chapter 3 < Chapter 5", "3-9", "5-1", -1},
		{"Chapter 5 > Chapter 3", "5-1", "3-9", 1},

		// Same chapter, different stages
		{"Stage 1 < Stage 2", "3-1", "3-2", -1},
		{"Stage 2 > Stage 1", "3-2", "3-1", 1},
		{"Stage 9 < Stage 10", "5-9", "5-10", -1},
		{"Stage 10 > Stage 9", "5-10", "5-9", 1},

		// Invalid formats (should return -1)
		{"Invalid level1", "invalid", "3-2", -1},
		{"Invalid level2", "3-2", "invalid", -1},
		{"Missing hyphen", "12", "3-2", -1},
		{"Too many parts", "1-2-3", "3-2", -1},
		{"Empty string", "", "3-2", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareLevels(tt.level1, tt.level2)
			if result != tt.expected {
				t.Errorf("compareLevels(%q, %q) = %d, expected %d",
					tt.level1, tt.level2, result, tt.expected)
			}
		})
	}
}

// TestIsMenuModeUnlocked_Adventure tests Adventure mode unlock logic.
func TestIsMenuModeUnlocked_Adventure(t *testing.T) {
	tests := []struct {
		name         string
		highestLevel string
		expected     bool
	}{
		{"New player (1-1)", "1-1", true},
		{"Mid-game player (3-5)", "3-5", true},
		{"Completed game (5-10)", "5-10", true},
		{"Invalid level", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMenuModeUnlocked(MenuButtonAdventure, tt.highestLevel)
			if result != tt.expected {
				t.Errorf("IsMenuModeUnlocked(Adventure, %q) = %v, expected %v",
					tt.highestLevel, result, tt.expected)
			}
		})
	}
}

// TestIsMenuModeUnlocked_Challenges tests Challenges mode unlock logic.
func TestIsMenuModeUnlocked_Challenges(t *testing.T) {
	tests := []struct {
		name         string
		highestLevel string
		expected     bool
	}{
		{"Before unlock (1-1)", "1-1", false},
		{"Before unlock (3-1)", "3-1", false},
		{"Exactly at unlock (3-2)", "3-2", true},
		{"After unlock (3-3)", "3-3", true},
		{"After unlock (4-1)", "4-1", true},
		{"After unlock (5-10)", "5-10", true},
		{"Invalid level", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMenuModeUnlocked(MenuButtonChallenges, tt.highestLevel)
			if result != tt.expected {
				t.Errorf("IsMenuModeUnlocked(Challenges, %q) = %v, expected %v",
					tt.highestLevel, result, tt.expected)
			}
		})
	}
}

// TestIsMenuModeUnlocked_Vasebreaker tests Vasebreaker mode unlock logic.
func TestIsMenuModeUnlocked_Vasebreaker(t *testing.T) {
	tests := []struct {
		name         string
		highestLevel string
		expected     bool
	}{
		{"Before unlock (1-1)", "1-1", false},
		{"Before unlock (3-2)", "3-2", false},
		{"Before unlock (5-9)", "5-9", false},
		{"Exactly at unlock (5-10)", "5-10", true},
		{"After unlock (5-11)", "5-11", true},
		{"Invalid level", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMenuModeUnlocked(MenuButtonVasebreaker, tt.highestLevel)
			if result != tt.expected {
				t.Errorf("IsMenuModeUnlocked(Vasebreaker, %q) = %v, expected %v",
					tt.highestLevel, result, tt.expected)
			}
		})
	}
}

// TestIsMenuModeUnlocked_Survival tests Survival mode unlock logic.
func TestIsMenuModeUnlocked_Survival(t *testing.T) {
	tests := []struct {
		name         string
		highestLevel string
		expected     bool
	}{
		{"Before unlock (1-1)", "1-1", false},
		{"Before unlock (3-2)", "3-2", false},
		{"Before unlock (5-9)", "5-9", false},
		{"Exactly at unlock (5-10)", "5-10", true},
		{"After unlock (5-11)", "5-11", true},
		{"Invalid level", "invalid", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsMenuModeUnlocked(MenuButtonSurvival, tt.highestLevel)
			if result != tt.expected {
				t.Errorf("IsMenuModeUnlocked(Survival, %q) = %v, expected %v",
					tt.highestLevel, result, tt.expected)
			}
		})
	}
}

// TestMenuButtonHitboxes verifies that the hitbox configuration is valid.
func TestMenuButtonHitboxes(t *testing.T) {
	// Check that we have exactly 5 button hitboxes (including both Adventure and StartAdventure variants)
	if len(MenuButtonHitboxes) != 5 {
		t.Errorf("Expected 5 button hitboxes, got %d", len(MenuButtonHitboxes))
	}

	// Check that all button types are present
	buttonTypes := make(map[MenuButtonType]bool)
	for _, hitbox := range MenuButtonHitboxes {
		buttonTypes[hitbox.ButtonType] = true

		// Verify dimensions are positive
		if hitbox.Width <= 0 {
			t.Errorf("Button %s has invalid width: %f", hitbox.TrackName, hitbox.Width)
		}
		if hitbox.Height <= 0 {
			t.Errorf("Button %s has invalid height: %f", hitbox.TrackName, hitbox.Height)
		}

		// Verify track name is not empty
		if hitbox.TrackName == "" {
			t.Errorf("Button has empty track name")
		}
	}

	// Check all button types are present
	expectedTypes := []MenuButtonType{
		MenuButtonAdventure,
		MenuButtonChallenges,
		MenuButtonVasebreaker,
		MenuButtonSurvival,
	}
	for _, expectedType := range expectedTypes {
		if !buttonTypes[expectedType] {
			t.Errorf("Missing button type: %v", expectedType)
		}
	}
}
