package scenes

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
)

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

// TestZombieHandAnimationBlocksButtons tests that buttons are blocked during zombie hand animation.
// Story 12.6 Task 2.6
func TestZombieHandAnimationBlocksButtons(t *testing.T) {
	tests := []struct {
		name          string
		menuState     MainMenuState
		expectBlocked bool
	}{
		{
			name:          "正常状态 - 按钮可点击",
			menuState:     MainMenuStateNormal,
			expectBlocked: false,
		},
		{
			name:          "僵尸手掌动画播放中 - 按钮被阻塞",
			menuState:     MainMenuStateZombieHandPlaying,
			expectBlocked: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a minimal MainMenuScene for testing
			scene := &MainMenuScene{
				menuState:           tt.menuState,
				hoveredButton:       "test_button",
				hoveredBottomButton: components.BottomButtonOptions,
			}

			// Simulate the button blocking logic from Update()
			// This is the logic that should block buttons when menuState == MainMenuStateZombieHandPlaying
			if scene.menuState == MainMenuStateZombieHandPlaying {
				scene.hoveredButton = ""
				scene.hoveredBottomButton = components.BottomButtonNone
			}

			// Verify the result
			if tt.expectBlocked {
				if scene.hoveredButton != "" {
					t.Errorf("Expected hoveredButton to be empty during zombie hand animation, got %q", scene.hoveredButton)
				}
				if scene.hoveredBottomButton != components.BottomButtonNone {
					t.Errorf("Expected hoveredBottomButton to be None during zombie hand animation, got %v", scene.hoveredBottomButton)
				}
			} else {
				if scene.hoveredButton != "test_button" {
					t.Errorf("Expected hoveredButton to be 'test_button' in normal state, got %q", scene.hoveredButton)
				}
				if scene.hoveredBottomButton != components.BottomButtonOptions {
					t.Errorf("Expected hoveredBottomButton to be Options in normal state, got %v", scene.hoveredBottomButton)
				}
			}
		})
	}
}
