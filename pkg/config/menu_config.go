package config

import (
	"strconv"
	"strings"
)

// MenuButtonType represents the type of main menu button.
// These correspond to the four main game modes visible on the tombstone menu.
//
// Story 12.1: Main Menu Tombstone System Enhancement
type MenuButtonType int

const (
	// MenuButtonAdventure is the Adventure mode button (always unlocked).
	MenuButtonAdventure MenuButtonType = iota
	// MenuButtonChallenges is the Mini-Games/Challenges mode button (unlocked at level 3-2).
	MenuButtonChallenges
	// MenuButtonVasebreaker is the Vasebreaker/Puzzle mode button (unlocked at level 5-10).
	MenuButtonVasebreaker
	// MenuButtonSurvival is the Survival mode button (unlocked at level 5-10).
	MenuButtonSurvival
)

// MenuButtonHitbox defines the clickable region for a menu button.
//
// IMPORTANT: The track names in SelectorScreen.reanim do NOT match the actual game modes!
// See docs/stories/12.1.story-reanim-design.md Section 2.1.5 for the correct mapping:
//   - SelectorScreen_Adventure_button → Adventure mode ✅
//   - SelectorScreen_Survival_button → Challenges mode ⚠️ (名称错位)
//   - SelectorScreen_Challenges_button → Vasebreaker mode ⚠️ (名称错位)
//   - SelectorScreen_ZenGarden_button → Survival mode ⚠️ (名称错位)
//
// Story 12.1: Main Menu Tombstone System Enhancement
type MenuButtonHitbox struct {
	// TrackName is the Reanim track name from SelectorScreen.reanim
	TrackName string
	// ButtonType is the actual game mode this button represents
	ButtonType MenuButtonType
	// X, Y are the top-left corner coordinates of the clickable region
	X, Y float64
	// Width, Height are the dimensions of the clickable region
	Width, Height float64
}

// MenuButtonHitboxes defines the click regions for all four main menu buttons.
//
// Coordinates extracted from SelectorScreen.reanim (final frame positions).
// Dimensions are based on button image sizes (330x120 approximate).
//
// Story 12.1: Main Menu Tombstone System Enhancement
var MenuButtonHitboxes = []MenuButtonHitbox{
	{
		TrackName:  "SelectorScreen_Adventure_button",
		ButtonType: MenuButtonAdventure,
		X:          405,
		Y:          79.7,
		Width:      330,
		Height:     120,
	},
	{
		TrackName:  "SelectorScreen_StartAdventure_button", // 新用户版本的冒险按钮
		ButtonType: MenuButtonAdventure,
		X:          405,
		Y:          79.7,
		Width:      330,
		Height:     120,
	},
	{
		TrackName:  "SelectorScreen_Survival_button",
		ButtonType: MenuButtonChallenges, // 注意：轨道名称是 Survival，但实际对应玩玩小游戏
		X:          406,
		Y:          173.1,
		Width:      330,
		Height:     120,
	},
	{
		TrackName:  "SelectorScreen_Challenges_button",
		ButtonType: MenuButtonVasebreaker, // 注意：轨道名称是 Challenges，但实际对应解谜模式
		X:          410,
		Y:          257.5,
		Width:      20,
		Height:     120,
	},
	{
		TrackName:  "SelectorScreen_ZenGarden_button",
		ButtonType: MenuButtonSurvival, // 注意：轨道名称是 ZenGarden，但实际对应生存模式
		X:          413,
		Y:          328.0,
		Width:      330,
		Height:     120,
	},
}

// IsMenuModeUnlocked checks if a menu mode is unlocked based on the player's highest level.
//
// Unlock rules:
//   - Adventure mode: Always unlocked
//   - Challenges mode (Mini-Games): Unlocked at level 3-2 or higher
//   - Vasebreaker mode (Puzzle): Unlocked at level 5-10 or higher
//   - Survival mode: Unlocked at level 5-10 or higher
//
// Parameters:
//   - modeType: The menu button type to check
//   - highestLevel: The player's highest level (format: "X-Y", e.g., "3-2")
//
// Returns:
//   - true if the mode is unlocked, false otherwise
//
// Story 12.1: Main Menu Tombstone System Enhancement
func IsMenuModeUnlocked(modeType MenuButtonType, highestLevel string) bool {
	switch modeType {
	case MenuButtonAdventure:
		return true // Adventure mode is always unlocked
	case MenuButtonChallenges:
		// Challenges (Mini-Games) unlocks at level 3-2
		return compareLevels(highestLevel, "3-2") >= 0
	case MenuButtonVasebreaker, MenuButtonSurvival:
		// Vasebreaker and Survival unlock at level 5-10 (after completing Adventure mode)
		return compareLevels(highestLevel, "5-10") >= 0
	default:
		return false
	}
}

// compareLevels compares two level strings (format: "X-Y").
//
// Returns:
//   - -1 if level1 < level2
//   - 0 if level1 == level2
//   - 1 if level1 > level2
//
// If parsing fails, returns -1 (treats invalid levels as lower).
//
// Story 12.1: Main Menu Tombstone System Enhancement
func compareLevels(level1, level2 string) int {
	// Parse level1
	parts1 := strings.Split(level1, "-")
	if len(parts1) != 2 {
		return -1 // Invalid format, treat as lower
	}
	chapter1, err1 := strconv.Atoi(parts1[0])
	stage1, err2 := strconv.Atoi(parts1[1])
	if err1 != nil || err2 != nil {
		return -1 // Invalid format, treat as lower
	}

	// Parse level2
	parts2 := strings.Split(level2, "-")
	if len(parts2) != 2 {
		return -1 // Invalid format, treat as lower
	}
	chapter2, err3 := strconv.Atoi(parts2[0])
	stage2, err4 := strconv.Atoi(parts2[1])
	if err3 != nil || err4 != nil {
		return -1 // Invalid format, treat as lower
	}

	// Compare chapters first
	if chapter1 < chapter2 {
		return -1
	}
	if chapter1 > chapter2 {
		return 1
	}

	// Chapters are equal, compare stages
	if stage1 < stage2 {
		return -1
	}
	if stage1 > stage2 {
		return 1
	}

	// Both chapters and stages are equal
	return 0
}
