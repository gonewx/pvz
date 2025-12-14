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
				if cfg.RedEyeRules.StartRound != 5 {
					t.Errorf("expected red eye start round = 5, got %d", cfg.RedEyeRules.StartRound)
				}
				if len(cfg.SceneTypeRestrictions.WaterZombies) != 2 {
					t.Errorf("expected 2 water zombies, got %d", len(cfg.SceneTypeRestrictions.WaterZombies))
				}
			},
		},
		{
			name: "negative red eye start round",
			yamlContent: `
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
				RedEyeRules: RedEyeRulesConfig{
					StartRound:       5,
					CapacityPerRound: 1,
				},
			},
			wantErr: false,
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
