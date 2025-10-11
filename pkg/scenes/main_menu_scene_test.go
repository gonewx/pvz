package scenes

import "testing"

// TestIsPointInRect tests the point-in-rectangle collision detection function.
func TestIsPointInRect(t *testing.T) {
	tests := []struct {
		name   string
		px, py float64
		x, y   float64
		w, h   float64
		want   bool
	}{
		{
			name: "点在矩形内部",
			px:   50, py: 50,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "点在矩形外部（右侧）",
			px:   150, py: 50,
			x: 0, y: 0, w: 100, h: 100,
			want: false,
		},
		{
			name: "点在矩形外部（下方）",
			px:   50, py: 150,
			x: 0, y: 0, w: 100, h: 100,
			want: false,
		},
		{
			name: "点在矩形外部（左侧）",
			px:   -10, py: 50,
			x: 0, y: 0, w: 100, h: 100,
			want: false,
		},
		{
			name: "点在矩形外部（上方）",
			px:   50, py: -10,
			x: 0, y: 0, w: 100, h: 100,
			want: false,
		},
		{
			name: "点在左上角（边界）",
			px:   0, py: 0,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "点在右下角（边界）",
			px:   100, py: 100,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "点在右上角（边界）",
			px:   100, py: 0,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "点在左下角（边界）",
			px:   0, py: 100,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "点在顶边中点（边界）",
			px:   50, py: 0,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "点在右边中点（边界）",
			px:   100, py: 50,
			x: 0, y: 0, w: 100, h: 100,
			want: true,
		},
		{
			name: "非零起点矩形-点在内部",
			px:   150, py: 150,
			x: 100, y: 100, w: 100, h: 100,
			want: true,
		},
		{
			name: "非零起点矩形-点在外部",
			px:   50, py: 50,
			x: 100, y: 100, w: 100, h: 100,
			want: false,
		},
		{
			name: "浮点数坐标-点在内部",
			px:   10.5, py: 20.7,
			x: 10.0, y: 20.0, w: 5.0, h: 5.0,
			want: true,
		},
		{
			name: "浮点数坐标-点在外部",
			px:   10.5, py: 19.9,
			x: 10.0, y: 20.0, w: 5.0, h: 5.0,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isPointInRect(tt.px, tt.py, tt.x, tt.y, tt.w, tt.h)
			if got != tt.want {
				t.Errorf("isPointInRect(%v, %v, %v, %v, %v, %v) = %v, want %v",
					tt.px, tt.py, tt.x, tt.y, tt.w, tt.h, got, tt.want)
			}
		})
	}
}
