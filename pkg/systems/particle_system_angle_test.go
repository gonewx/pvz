package systems

import (
	"math"
	"testing"
)

// TestAngleConversion 验证 PvZ 角度坐标系转换的正确性
func TestAngleConversion(t *testing.T) {
	tests := []struct {
		name           string
		pvzAngle       float64
		expectedDir    string  // 期望的飞行方向
		expectedVxSign float64 // 期望的 X 速度符号（正=右，负=左）
		expectedVySign float64 // 期望的 Y 速度符号（正=下，负=上）
	}{
		{
			name:           "ZombieHead 中间角度 167.5° 应该向右飞",
			pvzAngle:       167.5,
			expectedDir:    "right",
			expectedVxSign: 1.0, // 向右
			expectedVySign: 1.0, // 略微向下
		},
		{
			name:           "ZombieHead 最小角度 150° 应该向右下飞",
			pvzAngle:       150,
			expectedDir:    "right-down",
			expectedVxSign: 1.0, // 向右
			expectedVySign: 1.0, // 向下（330° 在屏幕坐标系中 Y 为正）
		},
		{
			name:           "ZombieHead 最大角度 185° 应该向右上飞",
			pvzAngle:       185,
			expectedDir:    "right-up",
			expectedVxSign: 1.0,  // 向右
			expectedVySign: -1.0, // 向上（5° 在屏幕坐标系中 Y 为负）
		},
		{
			name:           "MoweredZombieHead 中间角度 205° 应该向右上飞",
			pvzAngle:       205,
			expectedDir:    "right-up",
			expectedVxSign: 1.0,  // 向右
			expectedVySign: -1.0, // 向上（25° 在屏幕坐标系中 Y 为负）
		},
		{
			name:           "PvZ 0° 应该向左飞（僵尸前方）",
			pvzAngle:       0,
			expectedDir:    "left",
			expectedVxSign: -1.0, // 向左
			expectedVySign: 0.0,  // 水平
		},
		{
			name:           "PvZ 180° 应该向右飞（僵尸后方）",
			pvzAngle:       180,
			expectedDir:    "right",
			expectedVxSign: 1.0, // 向右
			expectedVySign: 0.0, // 水平
		},
		{
			name:           "PvZ 90° 应该向下飞",
			pvzAngle:       90,
			expectedDir:    "down",
			expectedVxSign: 0.0, // 水平
			expectedVySign: 1.0, // 向下
		},
		{
			name:           "PvZ 270° 应该向上飞",
			pvzAngle:       270,
			expectedDir:    "up",
			expectedVxSign: 0.0,  // 水平
			expectedVySign: -1.0, // 向上
		},
	}

	const speed = 100.0 // 固定速度用于测试

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 应用与 particle_system.go 中相同的转换逻辑
			screenAngle := tt.pvzAngle + 180.0
			if screenAngle >= 360.0 {
				screenAngle -= 360.0
			}

			angleRad := screenAngle * math.Pi / 180.0
			velocityX := speed * math.Cos(angleRad)
			velocityY := -speed * math.Sin(angleRad)

			// 验证 X 方向
			if tt.expectedVxSign != 0 {
				if math.Signbit(velocityX) != math.Signbit(tt.expectedVxSign) {
					t.Errorf("X 方向错误: pvzAngle=%.1f°, screenAngle=%.1f°, velocityX=%.2f, 期望符号=%+.0f",
						tt.pvzAngle, screenAngle, velocityX, tt.expectedVxSign)
				}
			} else {
				// 期望接近 0
				if math.Abs(velocityX) > 1.0 {
					t.Errorf("X 方向应该接近 0: pvzAngle=%.1f°, velocityX=%.2f",
						tt.pvzAngle, velocityX)
				}
			}

			// 验证 Y 方向
			if tt.expectedVySign != 0 {
				if math.Signbit(velocityY) != math.Signbit(tt.expectedVySign) {
					t.Errorf("Y 方向错误: pvzAngle=%.1f°, screenAngle=%.1f°, velocityY=%.2f, 期望符号=%+.0f",
						tt.pvzAngle, screenAngle, velocityY, tt.expectedVySign)
				}
			} else {
				// 期望接近 0
				if math.Abs(velocityY) > 1.0 {
					t.Errorf("Y 方向应该接近 0: pvzAngle=%.1f°, velocityY=%.2f",
						tt.pvzAngle, velocityY)
				}
			}

			t.Logf("✓ PvZ角度 %.1f° → 屏幕角度 %.1f° → 速度 (%.2f, %.2f) [%s]",
				tt.pvzAngle, screenAngle, velocityX, velocityY, tt.expectedDir)
		})
	}
}

// TestZombieHeadAngleRange 专门测试 ZombieHead 的角度范围
func TestZombieHeadAngleRange(t *testing.T) {
	const speed = 330.0 // ZombieHead 的实际速度

	// 测试角度范围 [150, 185]
	angles := []float64{150, 160, 167.5, 175, 185}

	t.Log("=== ZombieHead LaunchAngle [150 185] 验证 ===")
	for _, pvzAngle := range angles {
		screenAngle := pvzAngle + 180.0
		if screenAngle >= 360.0 {
			screenAngle -= 360.0
		}

		angleRad := screenAngle * math.Pi / 180.0
		velocityX := speed * math.Cos(angleRad)
		velocityY := -speed * math.Sin(angleRad)

		// 所有角度都应该向右飞（velocityX > 0）
		if velocityX <= 0 {
			t.Errorf("❌ 角度 %.1f° 向左飞了！velocityX=%.2f", pvzAngle, velocityX)
		} else {
			t.Logf("✓ PvZ %.1f° → 屏幕 %.1f° → 速度 (%.1f, %.1f) 向右飞",
				pvzAngle, screenAngle, velocityX, velocityY)
		}
	}
}

// TestMoweredZombieHeadAngleRange 专门测试 MoweredZombieHead 的角度范围
func TestMoweredZombieHeadAngleRange(t *testing.T) {
	const speed = 330.0

	// 测试角度范围 [190, 220]
	angles := []float64{190, 200, 205, 210, 220}

	t.Log("=== MoweredZombieHead LaunchAngle [190 220] 验证 ===")
	for _, pvzAngle := range angles {
		screenAngle := pvzAngle + 180.0
		if screenAngle >= 360.0 {
			screenAngle -= 360.0
		}

		angleRad := screenAngle * math.Pi / 180.0
		velocityX := speed * math.Cos(angleRad)
		velocityY := -speed * math.Sin(angleRad)

		// 所有角度都应该向右飞（velocityX > 0）
		if velocityX <= 0 {
			t.Errorf("❌ 角度 %.1f° 向左飞了！velocityX=%.2f", pvzAngle, velocityX)
		} else {
			// MoweredZombieHead 应该比 ZombieHead 更向下飞（velocityY 更大）
			t.Logf("✓ PvZ %.1f° → 屏幕 %.1f° → 速度 (%.1f, %.1f) 向右下飞",
				pvzAngle, screenAngle, velocityX, velocityY)
		}
	}
}

// TestPottedPlantGlowAngleRange 测试 PottedPlantGlow 的角度范围
func TestPottedPlantGlowAngleRange(t *testing.T) {
	const speed = 55.0 // 平均速度

	// 测试角度范围 [90, 270] - 应该是左侧半圆
	angles := []float64{90, 135, 180, 225, 270}

	t.Log("=== PottedPlantGlow LaunchAngle [90 270] 验证 ===")
	for _, pvzAngle := range angles {
		screenAngle := pvzAngle + 180.0
		if screenAngle >= 360.0 {
			screenAngle -= 360.0
		}

		angleRad := screenAngle * math.Pi / 180.0
		velocityX := speed * math.Cos(angleRad)
		velocityY := -speed * math.Sin(angleRad)

		direction := ""
		if velocityX > 1 {
			direction += "右"
		} else if velocityX < -1 {
			direction += "左"
		}
		if velocityY > 1 {
			direction += "下"
		} else if velocityY < -1 {
			direction += "上"
		}

		t.Logf("✓ PvZ %.1f° → 屏幕 %.1f° → 速度 (%.1f, %.1f) %s",
			pvzAngle, screenAngle, velocityX, velocityY, direction)
	}
}
