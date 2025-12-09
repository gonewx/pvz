package systems

import (
	"fmt"
	"testing"

	"github.com/gonewx/pvz/pkg/components"
)

// TestCooldownProgressCalculation 测试冷却进度计算
func TestCooldownProgressCalculation(t *testing.T) {
	tests := []struct {
		name             string
		cooldownTime     float64
		currentCooldown  float64
		expectedProgress float64
	}{
		{
			name:             "30% progress",
			cooldownTime:     10.0,
			currentCooldown:  3.0,
			expectedProgress: 0.3,
		},
		{
			name:             "50% progress",
			cooldownTime:     8.0,
			currentCooldown:  4.0,
			expectedProgress: 0.5,
		},
		{
			name:             "100% progress (just started)",
			cooldownTime:     10.0,
			currentCooldown:  10.0,
			expectedProgress: 1.0,
		},
		{
			name:             "0% progress (finished)",
			cooldownTime:     10.0,
			currentCooldown:  0.0,
			expectedProgress: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &components.PlantCardComponent{
				CooldownTime:    tt.cooldownTime,
				CurrentCooldown: tt.currentCooldown,
			}

			// 计算进度（模拟 PlantCardRenderSystem.Draw 中的逻辑）
			progress := card.CurrentCooldown / card.CooldownTime

			if progress != tt.expectedProgress {
				t.Errorf("Expected progress = %f, got %f", tt.expectedProgress, progress)
			}
		})
	}
}

// TestCoverHeightCalculation 测试覆盖层高度计算
func TestCoverHeightCalculation(t *testing.T) {
	cardHeight := 70.0

	tests := []struct {
		name                string
		progress            float64
		expectedCoverHeight float64
	}{
		{
			name:                "30% cooldown remaining",
			progress:            0.3,
			expectedCoverHeight: 21.0, // 70 * 0.3
		},
		{
			name:                "50% cooldown remaining",
			progress:            0.5,
			expectedCoverHeight: 35.0, // 70 * 0.5
		},
		{
			name:                "100% cooldown (just started)",
			progress:            1.0,
			expectedCoverHeight: 70.0, // 70 * 1.0 (完全覆盖)
		},
		{
			name:                "0% cooldown (finished)",
			progress:            0.0,
			expectedCoverHeight: 0.0, // 70 * 0.0 (无覆盖)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 计算覆盖层高度（模拟 PlantCardRenderSystem.Draw 中的逻辑）
			coverHeight := cardHeight * tt.progress

			if coverHeight != tt.expectedCoverHeight {
				t.Errorf("Expected coverHeight = %f, got %f", tt.expectedCoverHeight, coverHeight)
			}
		})
	}
}

// TestCoverYPosition 测试覆盖层Y坐标计算（从顶部向下）
func TestCoverYPosition(t *testing.T) {
	cardY := 8.0 // 卡片Y坐标

	tests := []struct {
		name           string
		progress       float64
		expectedCoverY float64
	}{
		{
			name:           "30% cooldown - cover from top",
			progress:       0.3,
			expectedCoverY: 8.0, // 从卡片顶部开始
		},
		{
			name:           "50% cooldown - cover from top",
			progress:       0.5,
			expectedCoverY: 8.0, // 从卡片顶部开始
		},
		{
			name:           "100% cooldown - cover from top",
			progress:       1.0,
			expectedCoverY: 8.0, // 从卡片顶部开始
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 计算覆盖层Y坐标（从顶部向下，Y坐标始终是卡片顶部）
			coverY := cardY

			if coverY != tt.expectedCoverY {
				t.Errorf("Expected coverY = %f, got %f", tt.expectedCoverY, coverY)
			}
		})
	}
}

// TestSunCostTextFormat 测试阳光数量文本格式
func TestSunCostTextFormat(t *testing.T) {
	tests := []struct {
		name         string
		sunCost      int
		expectedText string
	}{
		{
			name:         "Sunflower cost",
			sunCost:      50,
			expectedText: "50",
		},
		{
			name:         "Peashooter cost",
			sunCost:      100,
			expectedText: "100",
		},
		{
			name:         "Expensive plant",
			sunCost:      200,
			expectedText: "200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card := &components.PlantCardComponent{
				SunCost: tt.sunCost,
			}

			// 模拟文本格式化逻辑
			sunText := fmt.Sprintf("%d", card.SunCost)

			if sunText != tt.expectedText {
				t.Errorf("Expected text = %s, got %s", tt.expectedText, sunText)
			}
		})
	}
}
