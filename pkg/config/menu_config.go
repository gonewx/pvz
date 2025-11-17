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

// Point represents a 2D coordinate point.
type Point struct {
	X float64
	Y float64
}

// MenuButtonHitbox defines the clickable region for a menu button using a rotated rectangle (quadrilateral).
//
// The hitbox is defined by four corner points, allowing precise matching of tilted/rotated buttons.
// Points are specified in clockwise order: TopLeft → TopRight → BottomRight → BottomLeft.
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
	// Four corner points defining the quadrilateral hitbox (clockwise order)
	TopLeft     Point
	TopRight    Point
	BottomRight Point
	BottomLeft  Point
}

// IsPointInQuadrilateral checks if a point is inside a quadrilateral defined by four corners.
//
// Uses the cross product method: a point is inside if it's on the same side of all four edges.
// The corners must be in order (clockwise or counter-clockwise).
//
// Parameters:
//   - x, y: The point to test
//   - quad: The quadrilateral hitbox
//
// Returns:
//   - true if the point is inside the quadrilateral, false otherwise
func IsPointInQuadrilateral(x, y float64, quad *MenuButtonHitbox) bool {
	// 使用叉积法判断点是否在四边形内
	// 如果点在所有边的同一侧，则点在四边形内

	// 定义四条边（顺时针）
	edges := []struct{ p1, p2 Point }{
		{quad.TopLeft, quad.TopRight},       // 上边
		{quad.TopRight, quad.BottomRight},   // 右边
		{quad.BottomRight, quad.BottomLeft}, // 下边
		{quad.BottomLeft, quad.TopLeft},     // 左边
	}

	// 检查点是否在所有边的同一侧
	var sign int // 第一条边的符号（正或负）
	for i, edge := range edges {
		// 计算叉积：(p2 - p1) × (point - p1)
		cross := crossProduct(edge.p1, edge.p2, Point{x, y})

		if i == 0 {
			// 记录第一条边的符号
			if cross > 0 {
				sign = 1
			} else if cross < 0 {
				sign = -1
			} else {
				sign = 0
			}
		} else {
			// 检查后续边的符号是否一致
			if cross > 0 && sign < 0 {
				return false
			}
			if cross < 0 && sign > 0 {
				return false
			}
		}
	}

	return true
}

// crossProduct calculates the 2D cross product of vectors (p2-p1) and (p3-p1).
//
// Returns:
//   - > 0: p3 is on the left side of the line p1->p2
//   - < 0: p3 is on the right side of the line p1->p2
//   - = 0: p3 is on the line p1->p2
func crossProduct(p1, p2, p3 Point) float64 {
	return (p2.X-p1.X)*(p3.Y-p1.Y) - (p2.Y-p1.Y)*(p3.X-p1.X)
}

// MenuButtonHitboxes defines the click regions for all four main menu buttons.
//
// Coordinates are automatically calculated from SelectorScreen.reanim using cmd/calculate_hitbox tool.
// Each button is defined as a quadrilateral with four corner points to support rotated/skewed buttons.
//
// To regenerate this configuration:
//   go run cmd/calculate_hitbox/main.go
//
// Story 12.1: Main Menu Tombstone System Enhancement
var MenuButtonHitboxes = []MenuButtonHitbox{
	{
		TrackName:   "SelectorScreen_Adventure_button",
		ButtonType:  MenuButtonAdventure,
		TopLeft:     Point{X: 405.0, Y: 79.7},
		TopRight:    Point{X: 735.0, Y: 79.7},
		BottomRight: Point{X: 735.0, Y: 199.7},
		BottomLeft:  Point{X: 405.0, Y: 199.7},
	},
	{
		TrackName:   "SelectorScreen_StartAdventure_button",
		ButtonType:  MenuButtonAdventure, // 新用户版本的冒险按钮
		TopLeft:     Point{X: 405.0, Y: 65.0},
		TopRight:    Point{X: 735.0, Y: 65.0},
		BottomRight: Point{X: 735.0, Y: 185.0},
		BottomLeft:  Point{X: 405.0, Y: 185.0},
	},
	{
		TrackName:   "SelectorScreen_Survival_button",
		ButtonType:  MenuButtonChallenges, // 注意：轨道名称是 Survival，但实际对应玩玩小游戏
		TopLeft:     Point{X: 406.0, Y: 173.1},
		TopRight:    Point{X: 719.0, Y: 173.1},
		BottomRight: Point{X: 719.0, Y: 306.1},
		BottomLeft:  Point{X: 406.0, Y: 306.1},
	},
	{
		TrackName:   "SelectorScreen_Challenges_button",
		ButtonType:  MenuButtonVasebreaker, // 注意：轨道名称是 Challenges，但实际对应解谜模式
		TopLeft:     Point{X: 410.0, Y: 257.5},
		TopRight:    Point{X: 696.0, Y: 257.5},
		BottomRight: Point{X: 696.0, Y: 379.5},
		BottomLeft:  Point{X: 410.0, Y: 379.5},
	},
	{
		TrackName:   "SelectorScreen_ZenGarden_button",
		ButtonType:  MenuButtonSurvival, // 注意：轨道名称是 ZenGarden，但实际对应生存模式
		TopLeft:     Point{X: 413.0, Y: 328.0},
		TopRight:    Point{X: 679.0, Y: 328.0},
		BottomRight: Point{X: 679.0, Y: 451.0},
		BottomLeft:  Point{X: 413.0, Y: 451.0},
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
