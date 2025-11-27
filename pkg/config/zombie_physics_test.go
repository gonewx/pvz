package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadZombiePhysicsConfig(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *ZombiePhysicsConfig)
	}{
		{
			name: "valid config",
			yamlContent: `
spawnX:
  normal:
    min: 780
    max: 819
  flagWave:
    min: 820
    max: 859
  flagZombie: 800
  zomboni:
    min: 800
    max: 809
  gargantuar:
    min: 845
    max: 854

defeatBoundary:
  default: -100
  basic: -100
  football: -175
  zomboni: -175
  gargantuar: -150
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *ZombiePhysicsConfig) {
				// 验证普通波出生点
				if cfg.SpawnX.Normal.Min != 780 {
					t.Errorf("expected normal min = 780, got %f", cfg.SpawnX.Normal.Min)
				}
				if cfg.SpawnX.Normal.Max != 819 {
					t.Errorf("expected normal max = 819, got %f", cfg.SpawnX.Normal.Max)
				}
				// 验证旗帜波出生点
				if cfg.SpawnX.FlagWave.Min != 820 {
					t.Errorf("expected flagWave min = 820, got %f", cfg.SpawnX.FlagWave.Min)
				}
				// 验证进家边界
				if cfg.DefeatBoundary["basic"] != -100 {
					t.Errorf("expected basic boundary = -100, got %f", cfg.DefeatBoundary["basic"])
				}
				if cfg.DefeatBoundary["football"] != -175 {
					t.Errorf("expected football boundary = -175, got %f", cfg.DefeatBoundary["football"])
				}
			},
		},
		{
			name: "invalid normal spawn range",
			yamlContent: `
spawnX:
  normal:
    min: 850
    max: 800
  flagWave:
    min: 820
    max: 859
  flagZombie: 800
  zomboni:
    min: 800
    max: 809
  gargantuar:
    min: 845
    max: 854

defeatBoundary:
  default: -100
`,
			wantErr:     true,
			errContains: "normal spawn range invalid",
		},
		{
			name: "invalid flagWave spawn range",
			yamlContent: `
spawnX:
  normal:
    min: 780
    max: 819
  flagWave:
    min: 900
    max: 850
  flagZombie: 800
  zomboni:
    min: 800
    max: 809
  gargantuar:
    min: 845
    max: 854

defeatBoundary:
  default: -100
`,
			wantErr:     true,
			errContains: "flagWave spawn range invalid",
		},
		{
			name: "invalid zomboni spawn range",
			yamlContent: `
spawnX:
  normal:
    min: 780
    max: 819
  flagWave:
    min: 820
    max: 859
  flagZombie: 800
  zomboni:
    min: 900
    max: 800
  gargantuar:
    min: 845
    max: 854

defeatBoundary:
  default: -100
`,
			wantErr:     true,
			errContains: "zomboni spawn range invalid",
		},
		{
			name: "invalid gargantuar spawn range",
			yamlContent: `
spawnX:
  normal:
    min: 780
    max: 819
  flagWave:
    min: 820
    max: 859
  flagZombie: 800
  zomboni:
    min: 800
    max: 809
  gargantuar:
    min: 900
    max: 800

defeatBoundary:
  default: -100
`,
			wantErr:     true,
			errContains: "gargantuar spawn range invalid",
		},
		{
			name: "invalid defeat boundary positive value",
			yamlContent: `
spawnX:
  normal:
    min: 780
    max: 819
  flagWave:
    min: 820
    max: 859
  flagZombie: 800
  zomboni:
    min: 800
    max: 809
  gargantuar:
    min: 845
    max: 854

defeatBoundary:
  default: 50
`,
			wantErr:     true,
			errContains: "defeat boundary for 'default' should be <= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时 YAML 文件
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "zombie_physics.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// 加载配置
			cfg, err := LoadZombiePhysicsConfig(tmpFile)

			// 检查错误
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !strContains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				} else if tt.validate != nil {
					tt.validate(t, cfg)
				}
			}
		})
	}
}

func TestLoadZombiePhysicsConfig_FileNotFound(t *testing.T) {
	_, err := LoadZombiePhysicsConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !strContains(err.Error(), "failed to read zombie physics config") {
		t.Errorf("expected error about reading file, got: %v", err)
	}
}

func TestLoadZombiePhysicsConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := LoadZombiePhysicsConfig(tmpFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if !strContains(err.Error(), "failed to parse zombie physics config") {
		t.Errorf("expected YAML parse error, got: %v", err)
	}
}

func TestGetDefeatBoundary(t *testing.T) {
	cfg := &ZombiePhysicsConfig{
		DefeatBoundary: map[string]float64{
			"default":    -100,
			"basic":      -100,
			"football":   -175,
			"zomboni":    -175,
			"gargantuar": -150,
		},
	}

	tests := []struct {
		name         string
		zombieType   string
		wantBoundary float64
	}{
		{"basic zombie", "basic", -100},
		{"football zombie", "football", -175},
		{"zomboni", "zomboni", -175},
		{"gargantuar", "gargantuar", -150},
		{"unknown type uses default", "unknown_zombie", -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetDefeatBoundary(tt.zombieType)
			if got != tt.wantBoundary {
				t.Errorf("GetDefeatBoundary(%s) = %f, want %f", tt.zombieType, got, tt.wantBoundary)
			}
		})
	}
}

func TestGetDefeatBoundary_NoDefaultFallback(t *testing.T) {
	// 测试没有 default 配置时的兜底值
	cfg := &ZombiePhysicsConfig{
		DefeatBoundary: map[string]float64{
			"basic": -100,
		},
	}

	got := cfg.GetDefeatBoundary("unknown_type")
	if got != -100 {
		t.Errorf("GetDefeatBoundary for unknown type without default = %f, want -100", got)
	}
}

func TestGetSpawnXRange(t *testing.T) {
	cfg := &ZombiePhysicsConfig{
		SpawnX: SpawnXConfig{
			Normal:     SpawnRange{Min: 780, Max: 819},
			FlagWave:   SpawnRange{Min: 820, Max: 859},
			FlagZombie: 800,
			Zomboni:    SpawnRange{Min: 800, Max: 809},
			Gargantuar: SpawnRange{Min: 845, Max: 854},
		},
	}

	tests := []struct {
		name         string
		zombieType   string
		isFlagWave   bool
		isFlagZombie bool
		wantMin      float64
		wantMax      float64
	}{
		{
			name:       "normal wave basic zombie",
			zombieType: "basic",
			isFlagWave: false,
			wantMin:    780,
			wantMax:    819,
		},
		{
			name:       "flag wave basic zombie",
			zombieType: "basic",
			isFlagWave: true,
			wantMin:    820,
			wantMax:    859,
		},
		{
			name:         "flag zombie",
			zombieType:   "flag_zombie",
			isFlagWave:   true,
			isFlagZombie: true,
			wantMin:      800,
			wantMax:      800,
		},
		{
			name:       "zomboni",
			zombieType: "zomboni",
			isFlagWave: false,
			wantMin:    800,
			wantMax:    809,
		},
		{
			name:       "gargantuar",
			zombieType: "gargantuar",
			isFlagWave: false,
			wantMin:    845,
			wantMax:    854,
		},
		{
			name:       "gargantuar_redeye",
			zombieType: "gargantuar_redeye",
			isFlagWave: false,
			wantMin:    845,
			wantMax:    854,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMin, gotMax := cfg.GetSpawnXRange(tt.zombieType, tt.isFlagWave, tt.isFlagZombie)
			if gotMin != tt.wantMin || gotMax != tt.wantMax {
				t.Errorf("GetSpawnXRange(%s, %v, %v) = (%f, %f), want (%f, %f)",
					tt.zombieType, tt.isFlagWave, tt.isFlagZombie, gotMin, gotMax, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// ========================================
// 坐标转换测试
// ========================================

func TestGridToWorldX(t *testing.T) {
	tests := []struct {
		name           string
		gridRelativeX  float64
		expectedWorldX float64
	}{
		{"normal spawn min", 780, GridWorldStartX + 780},
		{"normal spawn max", 819, GridWorldStartX + 819},
		{"flag spawn min", 820, GridWorldStartX + 820},
		{"negative boundary", -100, GridWorldStartX - 100},
		{"zero", 0, GridWorldStartX},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GridToWorldX(tt.gridRelativeX)
			if got != tt.expectedWorldX {
				t.Errorf("GridToWorldX(%f) = %f, want %f", tt.gridRelativeX, got, tt.expectedWorldX)
			}
		})
	}
}

func TestWorldToGridX(t *testing.T) {
	tests := []struct {
		name         string
		worldX       float64
		expectedGrid float64
	}{
		{"spawn point", GridWorldStartX + 780, 780},
		{"boundary", GridWorldStartX - 100, -100},
		{"grid origin", GridWorldStartX, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorldToGridX(tt.worldX)
			if got != tt.expectedGrid {
				t.Errorf("WorldToGridX(%f) = %f, want %f", tt.worldX, got, tt.expectedGrid)
			}
		})
	}
}

func TestGridWorldRoundTrip(t *testing.T) {
	// 测试坐标转换往返正确性
	testValues := []float64{0, 100, -50, 780, -100, 500.5}

	for _, gridX := range testValues {
		worldX := GridToWorldX(gridX)
		roundTrip := WorldToGridX(worldX)
		if roundTrip != gridX {
			t.Errorf("GridToWorldX/WorldToGridX round trip failed: %f -> %f -> %f", gridX, worldX, roundTrip)
		}
	}
}

func TestGridToWorldY(t *testing.T) {
	tests := []struct {
		name           string
		gridRelativeY  float64
		expectedWorldY float64
	}{
		{"row 0 top", 0, GridWorldStartY},
		{"row center", CellHeight / 2, GridWorldStartY + CellHeight/2},
		{"negative", -50, GridWorldStartY - 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GridToWorldY(tt.gridRelativeY)
			if got != tt.expectedWorldY {
				t.Errorf("GridToWorldY(%f) = %f, want %f", tt.gridRelativeY, got, tt.expectedWorldY)
			}
		})
	}
}

func TestWorldToGridY(t *testing.T) {
	tests := []struct {
		name         string
		worldY       float64
		expectedGrid float64
	}{
		{"grid origin", GridWorldStartY, 0},
		{"row center", GridWorldStartY + CellHeight/2, CellHeight / 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorldToGridY(tt.worldY)
			if got != tt.expectedGrid {
				t.Errorf("WorldToGridY(%f) = %f, want %f", tt.worldY, got, tt.expectedGrid)
			}
		})
	}
}

func TestGetRowCenterY(t *testing.T) {
	tests := []struct {
		name     string
		row      int
		expected float64
	}{
		{"row 0", 0, GridWorldStartY + CellHeight/2},
		{"row 1", 1, GridWorldStartY + CellHeight + CellHeight/2},
		{"row 2", 2, GridWorldStartY + 2*CellHeight + CellHeight/2},
		{"row 4", 4, GridWorldStartY + 4*CellHeight + CellHeight/2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetRowCenterY(tt.row)
			if got != tt.expected {
				t.Errorf("GetRowCenterY(%d) = %f, want %f", tt.row, got, tt.expected)
			}
		})
	}
}

func TestGetDefeatBoundaryWorldX(t *testing.T) {
	cfg := &ZombiePhysicsConfig{
		DefeatBoundary: map[string]float64{
			"default":  -100,
			"basic":    -100,
			"football": -175,
		},
	}

	tests := []struct {
		name         string
		cfg          *ZombiePhysicsConfig
		zombieType   string
		expectedX    float64
	}{
		{
			name:       "basic zombie with config",
			cfg:        cfg,
			zombieType: "basic",
			expectedX:  GridWorldStartX - 100,
		},
		{
			name:       "football zombie with config",
			cfg:        cfg,
			zombieType: "football",
			expectedX:  GridWorldStartX - 175,
		},
		{
			name:       "nil config uses default",
			cfg:        nil,
			zombieType: "basic",
			expectedX:  GridWorldStartX - 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetDefeatBoundaryWorldX(tt.cfg, tt.zombieType)
			if got != tt.expectedX {
				t.Errorf("GetDefeatBoundaryWorldX(%v, %s) = %f, want %f", tt.cfg != nil, tt.zombieType, got, tt.expectedX)
			}
		})
	}
}

// strContains 检查字符串是否包含子串
func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
