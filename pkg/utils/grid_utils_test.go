package utils

import "testing"

// TestMouseToGridCoords 测试鼠标坐标到网格坐标的转换
func TestMouseToGridCoords(t *testing.T) {
	tests := []struct {
		name      string
		mouseX    int
		mouseY    int
		wantCol   int
		wantRow   int
		wantValid bool
	}{
		{
			name:      "左上角第一个格子",
			mouseX:    260,
			mouseY:    100,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
		{
			name:      "右下角最后一个格子",
			mouseX:    960,
			mouseY:    580,
			wantCol:   8,
			wantRow:   4,
			wantValid: true,
		},
		{
			name:      "中间格子 (col=4, row=2)",
			mouseX:    570,
			mouseY:    290,
			wantCol:   4,
			wantRow:   2,
			wantValid: true,
		},
		{
			name:      "格子边界测试 - 第一列最左边",
			mouseX:    250,
			mouseY:    100,
			wantCol:   0,
			wantRow:   0,
			wantValid: true,
		},
		{
			name:      "格子边界测试 - 第二列开始",
			mouseX:    330,
			mouseY:    100,
			wantCol:   1,
			wantRow:   0,
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCol, gotRow, gotValid := MouseToGridCoords(tt.mouseX, tt.mouseY)
			if gotCol != tt.wantCol || gotRow != tt.wantRow || gotValid != tt.wantValid {
				t.Errorf("MouseToGridCoords(%d, %d) = (%d, %d, %v), want (%d, %d, %v)",
					tt.mouseX, tt.mouseY, gotCol, gotRow, gotValid,
					tt.wantCol, tt.wantRow, tt.wantValid)
			}
		})
	}
}

// TestMouseOutOfBounds 测试边界外的坐标
func TestMouseOutOfBounds(t *testing.T) {
	tests := []struct {
		name   string
		mouseX int
		mouseY int
	}{
		{
			name:   "网格左边界外",
			mouseX: 200,
			mouseY: 100,
		},
		{
			name:   "网格上边界外",
			mouseX: 300,
			mouseY: 50,
		},
		{
			name:   "网格右边界外",
			mouseX: 1000,
			mouseY: 200,
		},
		{
			name:   "网格下边界外",
			mouseX: 300,
			mouseY: 600,
		},
		{
			name:   "完全在左上角外",
			mouseX: 0,
			mouseY: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, gotValid := MouseToGridCoords(tt.mouseX, tt.mouseY)
			if gotValid {
				t.Errorf("MouseToGridCoords(%d, %d) should be invalid (out of bounds), but got valid=true",
					tt.mouseX, tt.mouseY)
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
		wantCentX float64
		wantCentY float64
	}{
		{
			name:      "第一个格子 (0,0)",
			col:       0,
			row:       0,
			wantCentX: 250.0 + 40.0, // GridStartX + CellWidth/2
			wantCentY: 90.0 + 50.0,  // GridStartY + CellHeight/2
		},
		{
			name:      "最后一个格子 (8,4)",
			col:       8,
			row:       4,
			wantCentX: 250.0 + 8*80.0 + 40.0,
			wantCentY: 90.0 + 4*100.0 + 50.0,
		},
		{
			name:      "中间格子 (4,2)",
			col:       4,
			row:       2,
			wantCentX: 250.0 + 4*80.0 + 40.0,
			wantCentY: 90.0 + 2*100.0 + 50.0,
		},
		{
			name:      "第二列第一行 (1,0)",
			col:       1,
			row:       0,
			wantCentX: 250.0 + 1*80.0 + 40.0,
			wantCentY: 90.0 + 50.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCentX, gotCentY := GridToScreenCoords(tt.col, tt.row)
			if gotCentX != tt.wantCentX || gotCentY != tt.wantCentY {
				t.Errorf("GridToScreenCoords(%d, %d) = (%.1f, %.1f), want (%.1f, %.1f)",
					tt.col, tt.row, gotCentX, gotCentY,
					tt.wantCentX, tt.wantCentY)
			}
		})
	}
}

// TestRoundTripConversion 测试坐标转换的往返一致性
func TestRoundTripConversion(t *testing.T) {
	// 对于网格中心的坐标，进行往返转换应该得到相同的网格索引
	for row := 0; row < GridRows; row++ {
		for col := 0; col < GridColumns; col++ {
			centerX, centerY := GridToScreenCoords(col, row)
			gotCol, gotRow, gotValid := MouseToGridCoords(int(centerX), int(centerY))

			if !gotValid {
				t.Errorf("Round trip conversion for grid (%d, %d) resulted in invalid", col, row)
			}
			if gotCol != col || gotRow != row {
				t.Errorf("Round trip conversion for grid (%d, %d) got (%d, %d)",
					col, row, gotCol, gotRow)
			}
		}
	}
}
