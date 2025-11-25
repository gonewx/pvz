package components

import (
	"testing"
)

func TestShadowComponent_BasicFields(t *testing.T) {
	tests := []struct {
		name    string
		shadow  ShadowComponent
		wantW   float64
		wantH   float64
		wantA   float32
		wantOffY float64
	}{
		{
			name: "标准阴影",
			shadow: ShadowComponent{
				Width:   30.0,
				Height:  15.0,
				Alpha:   0.65,
				OffsetY: 0,
			},
			wantW:    30.0,
			wantH:    15.0,
			wantA:    0.65,
			wantOffY: 0,
		},
		{
			name: "带偏移的阴影",
			shadow: ShadowComponent{
				Width:   50.0,
				Height:  25.0,
				Alpha:   0.7,
				OffsetY: -10.0,
			},
			wantW:    50.0,
			wantH:    25.0,
			wantA:    0.7,
			wantOffY: -10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shadow.Width != tt.wantW {
				t.Errorf("Width = %v, want %v", tt.shadow.Width, tt.wantW)
			}
			if tt.shadow.Height != tt.wantH {
				t.Errorf("Height = %v, want %v", tt.shadow.Height, tt.wantH)
			}
			if tt.shadow.Alpha != tt.wantA {
				t.Errorf("Alpha = %v, want %v", tt.shadow.Alpha, tt.wantA)
			}
			if tt.shadow.OffsetY != tt.wantOffY {
				t.Errorf("OffsetY = %v, want %v", tt.shadow.OffsetY, tt.wantOffY)
			}
		})
	}
}

func TestShadowComponent_ZeroValues(t *testing.T) {
	shadow := ShadowComponent{}

	if shadow.Width != 0 {
		t.Errorf("Zero value Width = %v, want 0", shadow.Width)
	}
	if shadow.Height != 0 {
		t.Errorf("Zero value Height = %v, want 0", shadow.Height)
	}
	if shadow.Alpha != 0 {
		t.Errorf("Zero value Alpha = %v, want 0", shadow.Alpha)
	}
	if shadow.OffsetY != 0 {
		t.Errorf("Zero value OffsetY = %v, want 0", shadow.OffsetY)
	}
}

func TestShadowComponent_BoundaryValues(t *testing.T) {
	tests := []struct {
		name  string
		alpha float32
	}{
		{"Alpha最小值", 0.0},
		{"Alpha最大值", 1.0},
		{"Alpha超出范围(负数)", -0.1}, // 虽然无效,但组件应能存储
		{"Alpha超出范围(>1)", 1.5},    // 虽然无效,但组件应能存储
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shadow := ShadowComponent{
				Width:   30.0,
				Height:  15.0,
				Alpha:   tt.alpha,
				OffsetY: 0,
			}

			if shadow.Alpha != tt.alpha {
				t.Errorf("Alpha = %v, want %v", shadow.Alpha, tt.alpha)
			}
		})
	}
}
