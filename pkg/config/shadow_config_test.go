package config

import (
	"testing"
)

func TestGetShadowSize_ValidTypes(t *testing.T) {
	tests := []struct {
		name       string
		entityType string
		wantWidth  float64
		wantHeight float64
	}{
		{"豌豆射手", "peashooter", 50, 25},
		{"向日葵", "sunflower", 55, 28},
		{"坚果墙", "wallnut", 70, 35},
		{"樱桃炸弹", "cherrybomb", 60, 30},
		{"普通僵尸", "zombie", 60, 30},
		{"路障僵尸", "zombie_cone", 60, 30},
		{"铁桶僵尸", "zombie_bucket", 60, 30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size := GetShadowSize(tt.entityType)

			if size.Width != tt.wantWidth {
				t.Errorf("GetShadowSize(%q).Width = %v, want %v", tt.entityType, size.Width, tt.wantWidth)
			}
			if size.Height != tt.wantHeight {
				t.Errorf("GetShadowSize(%q).Height = %v, want %v", tt.entityType, size.Height, tt.wantHeight)
			}
		})
	}
}

func TestGetShadowSize_UnknownType(t *testing.T) {
	unknownTypes := []string{
		"unknown_plant",
		"nonexistent_zombie",
		"",
		"InvalidType",
	}

	for _, entityType := range unknownTypes {
		t.Run("未知类型_"+entityType, func(t *testing.T) {
			size := GetShadowSize(entityType)

			// 应该返回默认值
			if size.Width != DefaultShadowSize.Width {
				t.Errorf("GetShadowSize(%q).Width = %v, want default %v", entityType, size.Width, DefaultShadowSize.Width)
			}
			if size.Height != DefaultShadowSize.Height {
				t.Errorf("GetShadowSize(%q).Height = %v, want default %v", entityType, size.Height, DefaultShadowSize.Height)
			}
		})
	}
}

func TestDefaultShadowSize(t *testing.T) {
	if DefaultShadowSize.Width != 55 {
		t.Errorf("DefaultShadowSize.Width = %v, want 55", DefaultShadowSize.Width)
	}
	if DefaultShadowSize.Height != 28 {
		t.Errorf("DefaultShadowSize.Height = %v, want 28", DefaultShadowSize.Height)
	}
}

func TestDefaultShadowAlpha(t *testing.T) {
	if DefaultShadowAlpha != 0.65 {
		t.Errorf("DefaultShadowAlpha = %v, want 0.65", DefaultShadowAlpha)
	}
}

func TestShadowSizes_AllValidEntries(t *testing.T) {
	// 验证所有条目都有有效的宽度和高度
	for entityType, size := range ShadowSizes {
		if size.Width <= 0 {
			t.Errorf("ShadowSizes[%q].Width = %v, must be > 0", entityType, size.Width)
		}
		if size.Height <= 0 {
			t.Errorf("ShadowSizes[%q].Height = %v, must be > 0", entityType, size.Height)
		}

		// 检查合理的宽高比 (应该在 1.5:1 到 3:1 之间)
		ratio := size.Width / size.Height
		if ratio < 1.5 || ratio > 3.0 {
			t.Logf("警告: ShadowSizes[%q] 宽高比 = %.2f 可能不合理", entityType, ratio)
		}
	}
}

func TestShadowSizes_CoveragePlants(t *testing.T) {
	// 确保主要植物类型都有阴影配置
	requiredPlants := []string{
		"peashooter",
		"sunflower",
		"wallnut",
		"cherrybomb",
	}

	for _, plant := range requiredPlants {
		if _, exists := ShadowSizes[plant]; !exists {
			t.Errorf("缺少必需的植物阴影配置: %q", plant)
		}
	}
}

func TestShadowSizes_CoverageZombies(t *testing.T) {
	// 确保主要僵尸类型都有阴影配置
	requiredZombies := []string{
		"zombie",
		"zombie_cone",
		"zombie_bucket",
	}

	for _, zombie := range requiredZombies {
		if _, exists := ShadowSizes[zombie]; !exists {
			t.Errorf("缺少必需的僵尸阴影配置: %q", zombie)
		}
	}
}
