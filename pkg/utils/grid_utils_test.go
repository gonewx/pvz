package utils

import (
	"testing"

	"github.com/decker502/pvz/pkg/config"
)

// 测试用的摄像机位置常量
const (
	testCameraX = 215.0 // 游戏默认摄像机位置
)

// TestMouseToGridCoords 测试鼠标坐标到网格坐标的转换
func TestMouseToGridCoords(t *testing.T) {
	tests := []struct {
		name      string
		mouseX    int
		mouseY    int
		cameraX   float64
		wantCol   int
		wantRow   int
		wantValid bool
	}{
		{
			name:      "左上角第一个格子",
			mouseX:    36, // 屏幕坐标
			mouseY:    72, // 屏幕坐标
			cameraX:   testCameraX,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
		{
			name:      "右下角最后一个格子",
			mouseX:    716, // 屏幕坐标：36 + 8*80 + 40（格子中心）
			mouseY:    522, // 屏幕坐标：72 + 4*100 + 50（格子中心）
			cameraX:   testCameraX,
			wantCol:   8,
			wantRow:   4,
			wantValid: true,
		},
		{
			name:      "中间格子 (col=4, row=2)",
			mouseX:    356, // 屏幕坐标：36 + 4*80
			mouseY:    272, // 屏幕坐标：72 + 2*100
			cameraX:   testCameraX,
			wantCol:   4,
			wantRow:   2,
			wantValid: true,
		},
		{
			name:      "摄像机位置变化测试 - cameraX = 0",
			mouseX:    251, // 世界坐标 = 屏幕坐标（当cameraX=0时）
			mouseY:    72,
			cameraX:   0,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
		{
			name:      "摄像机位置变化测试 - cameraX = 300",
			mouseX:    -49, // 屏幕坐标：251 - 300 = -49
			mouseY:    72,
			cameraX:   300,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCol, gotRow, gotValid := MouseToGridCoords(
				tt.mouseX, tt.mouseY,
				tt.cameraX,
				config.GridWorldStartX, config.GridWorldStartY,
				config.GridColumns, config.GridRows,
				config.CellWidth, config.CellHeight,
			)
			if gotCol != tt.wantCol || gotRow != tt.wantRow || gotValid != tt.wantValid {
				t.Errorf("MouseToGridCoords(%d, %d, cameraX=%.1f) = (%d, %d, %v), want (%d, %d, %v)",
					tt.mouseX, tt.mouseY, tt.cameraX, gotCol, gotRow, gotValid,
					tt.wantCol, tt.wantRow, tt.wantValid)
			}
		})
	}
}

// TestMouseOutOfBounds 测试边界外的坐标
func TestMouseOutOfBounds(t *testing.T) {
	tests := []struct {
		name    string
		mouseX  int
		mouseY  int
		cameraX float64
	}{
		{
			name:    "网格左边界外",
			mouseX:  0,
			mouseY:  100,
			cameraX: testCameraX,
		},
		{
			name:    "网格上边界外",
			mouseX:  100,
			mouseY:  0,
			cameraX: testCameraX,
		},
		{
			name:    "网格右边界外",
			mouseX:  800,
			mouseY:  200,
			cameraX: testCameraX,
		},
		{
			name:    "网格下边界外",
			mouseX:  100,
			mouseY:  600,
			cameraX: testCameraX,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, gotValid := MouseToGridCoords(
				tt.mouseX, tt.mouseY,
				tt.cameraX,
				config.GridWorldStartX, config.GridWorldStartY,
				config.GridColumns, config.GridRows,
				config.CellWidth, config.CellHeight,
			)
			if gotValid {
				t.Errorf("MouseToGridCoords(%d, %d, cameraX=%.1f) should be invalid (out of bounds), but got valid=true",
					tt.mouseX, tt.mouseY, tt.cameraX)
			}
		})
	}
}

// TestGridToScreenCoords 测试网格坐标到屏幕坐标的转换
func TestGridToScreenCoords(t *testing.T) {
	tests := []struct {
		name      string
		col       int
		row       int
		cameraX   float64
		wantCentX float64
		wantCentY float64
	}{
		{
			name:      "第一个格子 (0,0) - 默认摄像机位置",
			col:       0,
			row:       0,
			cameraX:   testCameraX,
			wantCentX: 36.0 + 40.0, // (251 - 215) + CellWidth/2 = 36 + 40
			wantCentY: 72.0 + 50.0, // GridWorldStartY + CellHeight/2
		},
		{
			name:      "最后一个格子 (8,4) - 默认摄像机位置",
			col:       8,
			row:       4,
			cameraX:   testCameraX,
			wantCentX: 36.0 + 8*80.0 + 40.0,
			wantCentY: 72.0 + 4*100.0 + 50.0,
		},
		{
			name:      "中间格子 (4,2) - 默认摄像机位置",
			col:       4,
			row:       2,
			cameraX:   testCameraX,
			wantCentX: 36.0 + 4*80.0 + 40.0,
			wantCentY: 72.0 + 2*100.0 + 50.0,
		},
		{
			name:      "第一个格子 (0,0) - cameraX = 0",
			col:       0,
			row:       0,
			cameraX:   0,
			wantCentX: 251.0 + 40.0, // GridWorldStartX + CellWidth/2
			wantCentY: 72.0 + 50.0,
		},
		{
			name:      "第一个格子 (0,0) - cameraX = 300",
			col:       0,
			row:       0,
			cameraX:   300,
			wantCentX: -49.0 + 40.0, // (251 - 300) + CellWidth/2
			wantCentY: 72.0 + 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCentX, gotCentY := GridToScreenCoords(
				tt.col, tt.row,
				tt.cameraX,
				config.GridWorldStartX, config.GridWorldStartY,
				config.CellWidth, config.CellHeight,
			)
			if gotCentX != tt.wantCentX || gotCentY != tt.wantCentY {
				t.Errorf("GridToScreenCoords(%d, %d, cameraX=%.1f) = (%.1f, %.1f), want (%.1f, %.1f)",
					tt.col, tt.row, tt.cameraX, gotCentX, gotCentY,
					tt.wantCentX, tt.wantCentY)
			}
		})
	}
}

// TestRoundTripConversion 测试坐标转换的往返一致性
func TestRoundTripConversion(t *testing.T) {
	// 测试不同的摄像机位置
	testCameraPositions := []float64{0, 100, 215, 300, 500}

	for _, cameraX := range testCameraPositions {
		t.Run("CameraX="+string(rune(int(cameraX))), func(t *testing.T) {
			// 对于网格中心的坐标，进行往返转换应该得到相同的网格索引
			for row := 0; row < config.GridRows; row++ {
				for col := 0; col < config.GridColumns; col++ {
					centerX, centerY := GridToScreenCoords(
						col, row,
						cameraX,
						config.GridWorldStartX, config.GridWorldStartY,
						config.CellWidth, config.CellHeight,
					)
					gotCol, gotRow, gotValid := MouseToGridCoords(
						int(centerX), int(centerY),
						cameraX,
						config.GridWorldStartX, config.GridWorldStartY,
						config.GridColumns, config.GridRows,
						config.CellWidth, config.CellHeight,
					)

					if !gotValid {
						t.Errorf("[cameraX=%.1f] Round trip conversion for grid (%d, %d) resulted in invalid", cameraX, col, row)
					}
					if gotCol != col || gotRow != row {
						t.Errorf("[cameraX=%.1f] Round trip conversion for grid (%d, %d) got (%d, %d)",
							cameraX, col, row, gotCol, gotRow)
					}
				}
			}
		})
	}
}
