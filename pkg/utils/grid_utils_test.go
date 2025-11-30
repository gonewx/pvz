package utils

import (
	"testing"

	"github.com/decker502/pvz/pkg/config"
)

// 测试用的摄像机位置常量
// 注意：这应该与 config.GameCameraX (220.0) 匹配才能使屏幕坐标计算正确
const (
	testCameraX = 220.0 // 与 config.GameCameraX 一致
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
			mouseX:    35, // 屏幕坐标：GridScreenStartX = 35
			mouseY:    78, // 屏幕坐标：GridWorldStartY = 78
			cameraX:   testCameraX,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
		{
			name:      "右下角最后一个格子",
			mouseX:    35 + 8*80 + 40, // 屏幕坐标：GridScreenStartX + 8*CellWidth + CellWidth/2
			mouseY:    78 + 4*100 + 50, // 屏幕坐标：GridWorldStartY + 4*CellHeight + CellHeight/2
			cameraX:   testCameraX,
			wantCol:   8,
			wantRow:   4,
			wantValid: true,
		},
		{
			name:      "中间格子 (col=4, row=2)",
			mouseX:    35 + 4*80 + 40, // 屏幕坐标：GridScreenStartX + 4*CellWidth + CellWidth/2
			mouseY:    78 + 2*100 + 50, // 屏幕坐标：GridWorldStartY + 2*CellHeight + CellHeight/2
			cameraX:   testCameraX,
			wantCol:   4,
			wantRow:   2,
			wantValid: true,
		},
		{
			name:      "摄像机位置变化测试 - cameraX = 0",
			mouseX:    255, // 世界坐标 = 屏幕坐标（当cameraX=0时），GridWorldStartX = 255
			mouseY:    78,
			cameraX:   0,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
		{
			name:      "摄像机位置变化测试 - cameraX = 300",
			mouseX:    -45, // 屏幕坐标：255 - 300 = -45 (GridWorldStartX - cameraX)
			mouseY:    78,
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
			cameraX:   testCameraX, // 220
			wantCentX: 35.0 + 40.0, // (255 - 220) + CellWidth/2 = 35 + 40 = 75
			wantCentY: 78.0 + 50.0, // GridWorldStartY + CellHeight/2 = 78 + 50 = 128
		},
		{
			name:      "最后一个格子 (8,4) - 默认摄像机位置",
			col:       8,
			row:       4,
			cameraX:   testCameraX,
			wantCentX: 35.0 + 8*80.0 + 40.0,  // 35 + 640 + 40 = 715
			wantCentY: 78.0 + 4*100.0 + 50.0, // 78 + 400 + 50 = 528
		},
		{
			name:      "中间格子 (4,2) - 默认摄像机位置",
			col:       4,
			row:       2,
			cameraX:   testCameraX,
			wantCentX: 35.0 + 4*80.0 + 40.0,  // 35 + 320 + 40 = 395
			wantCentY: 78.0 + 2*100.0 + 50.0, // 78 + 200 + 50 = 328
		},
		{
			name:      "第一个格子 (0,0) - cameraX = 0",
			col:       0,
			row:       0,
			cameraX:   0,
			wantCentX: 255.0 + 40.0, // GridWorldStartX + CellWidth/2 = 255 + 40 = 295
			wantCentY: 78.0 + 50.0,  // 128
		},
		{
			name:      "第一个格子 (0,0) - cameraX = 300",
			col:       0,
			row:       0,
			cameraX:   300,
			wantCentX: -45.0 + 40.0, // (255 - 300) + CellWidth/2 = -45 + 40 = -5
			wantCentY: 78.0 + 50.0,  // 128
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
	testCameraPositions := []float64{0, 100, 220, 300, 500}

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

// TestGetEntityRow 测试实体行计算函数
func TestGetEntityRow(t *testing.T) {
	tests := []struct {
		name    string
		worldY  float64
		wantRow int
	}{
		{
			name:    "第0行 - 网格起始位置",
			worldY:  config.GridWorldStartY,
			wantRow: 0,
		},
		{
			name:    "第0行 - 格子中间",
			worldY:  config.GridWorldStartY + config.CellHeight/2,
			wantRow: 0,
		},
		{
			name:    "第1行 - 起始位置",
			worldY:  config.GridWorldStartY + config.CellHeight,
			wantRow: 1,
		},
		{
			name:    "第2行 - 中间位置",
			worldY:  config.GridWorldStartY + 2*config.CellHeight + config.CellHeight/2,
			wantRow: 2,
		},
		{
			name:    "第4行 - 最后一行起始",
			worldY:  config.GridWorldStartY + 4*config.CellHeight,
			wantRow: 4,
		},
		{
			name:    "第3行 - 接近行尾",
			worldY:  config.GridWorldStartY + 4*config.CellHeight - 1,
			wantRow: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRow := GetEntityRow(tt.worldY, config.GridWorldStartY, config.CellHeight)
			if gotRow != tt.wantRow {
				t.Errorf("GetEntityRow(worldY=%.1f) = %d, want %d",
					tt.worldY, gotRow, tt.wantRow)
			}
		})
	}
}

// TestGetEntityRow_SameRowDetection 测试同行检测场景
// 验证豌豆射手和僵尸是否在同一行的判断逻辑
func TestGetEntityRow_SameRowDetection(t *testing.T) {
	// 模拟豌豆射手在第2行
	peashooterY := config.GridWorldStartY + 2*config.CellHeight + 30.0 // 第2行，偏移30像素
	peashooterRow := GetEntityRow(peashooterY, config.GridWorldStartY, config.CellHeight)

	tests := []struct {
		name        string
		zombieY     float64
		wantSameRow bool
	}{
		{
			name:        "僵尸在同一行（第2行起始）",
			zombieY:     config.GridWorldStartY + 2*config.CellHeight,
			wantSameRow: true,
		},
		{
			name:        "僵尸在同一行（第2行中间）",
			zombieY:     config.GridWorldStartY + 2*config.CellHeight + 50.0,
			wantSameRow: true,
		},
		{
			name:        "僵尸在同一行（第2行末尾）",
			zombieY:     config.GridWorldStartY + 3*config.CellHeight - 1,
			wantSameRow: true,
		},
		{
			name:        "僵尸在上一行（第1行）",
			zombieY:     config.GridWorldStartY + 1*config.CellHeight + 50.0,
			wantSameRow: false,
		},
		{
			name:        "僵尸在下一行（第3行）",
			zombieY:     config.GridWorldStartY + 3*config.CellHeight + 10.0,
			wantSameRow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zombieRow := GetEntityRow(tt.zombieY, config.GridWorldStartY, config.CellHeight)
			gotSameRow := (zombieRow == peashooterRow)
			if gotSameRow != tt.wantSameRow {
				t.Errorf("Peashooter row=%d, Zombie worldY=%.1f (row=%d), same row=%v, want %v",
					peashooterRow, tt.zombieY, zombieRow, gotSameRow, tt.wantSameRow)
			}
		})
	}
}
