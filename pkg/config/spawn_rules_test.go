package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSpawnRules(t *testing.T) {
	tests := []struct {
		name        string
		yamlContent string
		wantErr     bool
		errContains string
		validate    func(*testing.T, *SpawnRulesConfig)
	}{
		{
			name: "valid config",
			yamlContent: `
zombieTiers:
  basic: 1
  buckethead: 2
  gargantuar: 4

tierWaveRestrictions:
  1: 1
  2: 3
  3: 8
  4: 15

redEyeRules:
  startRound: 5
  capacityPerRound: 1

sceneTypeRestrictions:
  waterZombies:
    - snorkel
    - dolphinrider
  dancingRestrictions:
    prohibitedScenes:
      - roof
    requiresAdjacentLanes: true
  waterLaneConfig:
    pool: [3, 4]
    fog: [3, 4]
`,
			wantErr: false,
			validate: func(t *testing.T, cfg *SpawnRulesConfig) {
				if cfg.ZombieTiers["basic"] != 1 {
					t.Errorf("expected basic tier = 1, got %d", cfg.ZombieTiers["basic"])
				}
				if cfg.ZombieTiers["buckethead"] != 2 {
					t.Errorf("expected buckethead tier = 2, got %d", cfg.ZombieTiers["buckethead"])
				}
				if cfg.TierWaveRestrictions[4] != 15 {
					t.Errorf("expected tier 4 min wave = 15, got %d", cfg.TierWaveRestrictions[4])
				}
				if cfg.RedEyeRules.StartRound != 5 {
					t.Errorf("expected red eye start round = 5, got %d", cfg.RedEyeRules.StartRound)
				}
				if len(cfg.SceneTypeRestrictions.WaterZombies) != 2 {
					t.Errorf("expected 2 water zombies, got %d", len(cfg.SceneTypeRestrictions.WaterZombies))
				}
			},
		},
		{
			name: "empty zombie tiers",
			yamlContent: `
zombieTiers: {}
tierWaveRestrictions:
  1: 1
redEyeRules:
  startRound: 5
  capacityPerRound: 1
`,
			wantErr:     true,
			errContains: "zombieTiers cannot be empty",
		},
		{
			name: "invalid tier number",
			yamlContent: `
zombieTiers:
  basic: 5
tierWaveRestrictions:
  1: 1
redEyeRules:
  startRound: 5
  capacityPerRound: 1
`,
			wantErr:     true,
			errContains: "zombie tier must be between 1 and 4",
		},
		{
			name: "invalid tier wave restriction",
			yamlContent: `
zombieTiers:
  basic: 1
tierWaveRestrictions:
  5: 1
redEyeRules:
  startRound: 5
  capacityPerRound: 1
`,
			wantErr:     true,
			errContains: "tier in tierWaveRestrictions must be between 1 and 4",
		},
		{
			name: "negative min wave",
			yamlContent: `
zombieTiers:
  basic: 1
tierWaveRestrictions:
  1: -1
redEyeRules:
  startRound: 5
  capacityPerRound: 1
`,
			wantErr:     true,
			errContains: "minimum wave for tier 1 must be >= 1",
		},
		{
			name: "negative red eye start round",
			yamlContent: `
zombieTiers:
  basic: 1
tierWaveRestrictions:
  1: 1
redEyeRules:
  startRound: -1
  capacityPerRound: 1
`,
			wantErr:     true,
			errContains: "redEyeRules.startRound must be >= 0",
		},
		{
			name: "negative red eye capacity per round",
			yamlContent: `
zombieTiers:
  basic: 1
tierWaveRestrictions:
  1: 1
redEyeRules:
  startRound: 5
  capacityPerRound: -1
`,
			wantErr:     true,
			errContains: "redEyeRules.capacityPerRound must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建临时 YAML 文件
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "spawn_rules.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("failed to create temp file: %v", err)
			}

			// 加载配置
			cfg, err := LoadSpawnRules(tmpFile)

			// 检查错误
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
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

func TestLoadSpawnRules_FileNotFound(t *testing.T) {
	_, err := LoadSpawnRules("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
	if !contains(err.Error(), "failed to read spawn rules file") {
		t.Errorf("expected error about reading file, got: %v", err)
	}
}

func TestLoadSpawnRules_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(tmpFile, []byte("invalid: yaml: content:"), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	_, err := LoadSpawnRules(tmpFile)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
	if !contains(err.Error(), "failed to parse spawn rules YAML") {
		t.Errorf("expected YAML parse error, got: %v", err)
	}
}

func TestValidateSpawnRules(t *testing.T) {
	tests := []struct {
		name        string
		config      *SpawnRulesConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &SpawnRulesConfig{
				ZombieTiers: map[string]int{
					"basic":      1,
					"gargantuar": 4,
				},
				TierWaveRestrictions: map[int]int{
					1: 1,
					4: 15,
				},
				RedEyeRules: RedEyeRulesConfig{
					StartRound:       5,
					CapacityPerRound: 1,
				},
			},
			wantErr: false,
		},
		{
			name: "empty zombie type",
			config: &SpawnRulesConfig{
				ZombieTiers: map[string]int{
					"": 1,
				},
				TierWaveRestrictions: map[int]int{1: 1},
				RedEyeRules: RedEyeRulesConfig{
					StartRound:       5,
					CapacityPerRound: 1,
				},
			},
			wantErr:     true,
			errContains: "zombie type cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSpawnRules(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errContains)
				} else if tt.errContains != "" && !contains(err.Error(), tt.errContains) {
					t.Errorf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
