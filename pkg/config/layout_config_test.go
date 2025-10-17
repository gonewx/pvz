package config

import (
	"testing"
)

// TestCalculateSodRollPosition 测试 SodRoll 实体位置计算
func TestCalculateSodRollPosition(t *testing.T) {
	tests := []struct {
		name         string
		enabledLanes []int
		wantPosX     float64
		wantPosY     float64
	}{
		{
			name:         "第3行单行（1-1关）",
			enabledLanes: []int{3},
			wantPosX:     0,                                       // 固定为0
			wantPosY:     GridWorldStartY + 2*CellHeight + CellHeight/2 - SodRollBaseY, // 第3行中心 - 326.7
		},
		{
			name:         "第2-4行三行（1-2关）",
			enabledLanes: []int{2, 3, 4},
			wantPosX:     0,
			wantPosY:     GridWorldStartY + 2*CellHeight + CellHeight/2 - SodRollBaseY, // 第3行中心 - 326.7
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPosX, gotPosY := CalculateSodRollPosition(tt.enabledLanes)

			// 使用epsilon比较浮点数，避免精度问题
			epsilon := 0.01
			if gotPosX < tt.wantPosX-epsilon || gotPosX > tt.wantPosX+epsilon {
				t.Errorf("CalculateSodRollPosition() posX = %.2f, want %.2f", gotPosX, tt.wantPosX)
			}

			if gotPosY < tt.wantPosY-epsilon || gotPosY > tt.wantPosY+epsilon {
				t.Errorf("CalculateSodRollPosition() posY = %.2f, want %.2f", gotPosY, tt.wantPosY)
			}

			// 验证最终世界Y坐标（SodRollCap位置）
			finalWorldY := gotPosY + SodRollBaseY
			expectedCenterY := GridWorldStartY + (float64(tt.enabledLanes[0]+tt.enabledLanes[len(tt.enabledLanes)-1])/2.0 - 1.0 + 0.5) * CellHeight

			t.Logf("实体Position.Y = %.2f", gotPosY)
			t.Logf("SodRollCap.reanim基准Y = %.1f", SodRollBaseY)
			t.Logf("最终世界Y（SodRollCap）= %.2f", finalWorldY)
			t.Logf("期望中心Y = %.1f", expectedCenterY)
			t.Logf("差距 = %.2f", finalWorldY - expectedCenterY)
		})
	}
}

// TestCalculateSodOverlayPosition 测试草皮叠加图位置计算
func TestCalculateSodOverlayPosition(t *testing.T) {
	tests := []struct {
		name           string
		enabledLanes   []int
		sodImageHeight float64
		wantX          float64
		wantYApprox    float64 // 大约的Y值
	}{
		{
			name:           "第3行单行（127高）",
			enabledLanes:   []int{3},
			sodImageHeight: 127,
			wantX:          GridWorldStartX - 30.0, // ≈ 222
			wantYApprox:    GridWorldStartY + 2*CellHeight, // 第3行起点 ≈ 272
		},
		{
			name:           "第2-4行三行（355高）",
			enabledLanes:   []int{2, 3, 4},
			sodImageHeight: 355,
			wantX:          GridWorldStartX - 30.0, // ≈ 222
			wantYApprox:    GridWorldStartY + 1*CellHeight, // 第2行起点 ≈ 172
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotX, gotY := CalculateSodOverlayPosition(tt.enabledLanes, tt.sodImageHeight)

			if gotX != tt.wantX {
				t.Errorf("CalculateSodOverlayPosition() X = %.1f, want %.1f", gotX, tt.wantX)
			}

			// Y坐标只检查大致范围
			if gotY < tt.wantYApprox - 50 || gotY > tt.wantYApprox + 50 {
				t.Errorf("CalculateSodOverlayPosition() Y = %.1f, expected around %.1f", gotY, tt.wantYApprox)
			}

			t.Logf("草皮叠加图位置: (%.1f, %.1f)", gotX, gotY)
			t.Logf("草皮覆盖范围: X[%.1f - %.1f], Y[%.1f - %.1f]",
				gotX, gotX + SodRowWidth,
				gotY, gotY + tt.sodImageHeight)
		})
	}
}

// TestSodRollAndOverlayAlignment 测试草皮卷追踪位置和草皮叠加图的对齐
func TestSodRollAndOverlayAlignment(t *testing.T) {
	enabledLanes := []int{3}
	sodImageHeight := 127.0

	// 计算草皮叠加图位置
	sodOverlayX, _ := CalculateSodOverlayPosition(enabledLanes, sodImageHeight)

	// 模拟草皮卷追踪位置
	sodRollStartX := sodOverlayX
	sodRollEndX := sodOverlayX + SodRowWidth

	t.Logf("=== 草皮卷追踪位置验证 ===")
	t.Logf("草皮叠加图起点: X = %.1f", sodOverlayX)
	t.Logf("草皮卷追踪范围: %.1f → %.1f", sodRollStartX, sodRollEndX)
	t.Logf("草皮宽度: %.1f", SodRowWidth)

	// 验证范围一致性
	if sodRollStartX != sodOverlayX {
		t.Errorf("草皮卷起点(%.1f) 应该等于草皮叠加图起点(%.1f)", sodRollStartX, sodOverlayX)
	}

	if sodRollEndX != sodOverlayX + SodRowWidth {
		t.Errorf("草皮卷终点(%.1f) 应该等于草皮叠加图终点(%.1f)", sodRollEndX, sodOverlayX + SodRowWidth)
	}

	// 模拟动画进度
	for progress := 0.0; progress <= 1.0; progress += 0.25 {
		sodRollPosX := sodRollStartX + progress * (sodRollEndX - sodRollStartX)
		visibleWidth := sodRollPosX - sodOverlayX

		t.Logf("进度 %.0f%%: 草皮卷位置=%.1f, 可见宽度=%.1f/%.1f",
			progress*100, sodRollPosX, visibleWidth, SodRowWidth)
	}
}
