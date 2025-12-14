package systems

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
)

func TestCheckZombieTypeAllowed(t *testing.T) {
	tests := []struct {
		name          string
		zombieType    string
		allowedTypes  []string
		expectedValid bool
	}{
		{
			name:          "allowed zombie type",
			zombieType:    "basic",
			allowedTypes:  []string{"basic", "conehead", "buckethead"},
			expectedValid: true,
		},
		{
			name:          "not allowed zombie type",
			zombieType:    "gargantuar",
			allowedTypes:  []string{"basic", "conehead"},
			expectedValid: false,
		},
		{
			name:          "empty allowed list allows all",
			zombieType:    "basic",
			allowedTypes:  []string{},
			expectedValid: true,
		},
		{
			name:          "nil allowed list allows all",
			zombieType:    "basic",
			allowedTypes:  nil,
			expectedValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CheckZombieTypeAllowed(tt.zombieType, tt.allowedTypes)
			if result != tt.expectedValid {
				t.Errorf("CheckZombieTypeAllowed() = %v, want %v", result, tt.expectedValid)
			}
		})
	}
}

func TestCalculateRedEyeCapacity(t *testing.T) {
	spawnRules := &config.SpawnRulesConfig{
		RedEyeRules: config.RedEyeRulesConfig{
			StartRound:       5,
			CapacityPerRound: 1,
		},
	}

	tests := []struct {
		name             string
		roundNumber      int
		expectedCapacity int
	}{
		{
			name:             "round 0 (below start round)",
			roundNumber:      0,
			expectedCapacity: 0,
		},
		{
			name:             "round 4 (below start round)",
			roundNumber:      4,
			expectedCapacity: 0,
		},
		{
			name:             "round 5 (start round)",
			roundNumber:      5,
			expectedCapacity: 1,
		},
		{
			name:             "round 6",
			roundNumber:      6,
			expectedCapacity: 2,
		},
		{
			name:             "round 10",
			roundNumber:      10,
			expectedCapacity: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity := CalculateRedEyeCapacity(tt.roundNumber, spawnRules)
			if capacity != tt.expectedCapacity {
				t.Errorf("CalculateRedEyeCapacity() = %d, want %d", capacity, tt.expectedCapacity)
			}
		})
	}
}

func TestCheckRedEyeLimit(t *testing.T) {
	spawnRules := &config.SpawnRulesConfig{
		RedEyeRules: config.RedEyeRulesConfig{
			StartRound:       5,
			CapacityPerRound: 1,
		},
	}

	tests := []struct {
		name          string
		zombieType    string
		redEyeCount   int
		roundNumber   int
		expectedValid bool
		expectError   bool
	}{
		{
			name:          "non-red-eye zombie always passes",
			zombieType:    "basic",
			redEyeCount:   10,
			roundNumber:   0,
			expectedValid: true,
			expectError:   false,
		},
		{
			name:          "red eye at round 5 with count 0",
			zombieType:    "gargantuar_redeye",
			redEyeCount:   0,
			roundNumber:   5,
			expectedValid: true,
			expectError:   false,
		},
		{
			name:          "red eye at round 5 with count 1 (at limit)",
			zombieType:    "gargantuar_redeye",
			redEyeCount:   1,
			roundNumber:   5,
			expectedValid: false,
			expectError:   true,
		},
		{
			name:          "red eye at round 4 (below start round)",
			zombieType:    "gargantuar_redeye",
			redEyeCount:   0,
			roundNumber:   4,
			expectedValid: false,
			expectError:   true,
		},
		{
			name:          "red eye at round 6 with count 1",
			zombieType:    "gargantuar_redeye",
			redEyeCount:   1,
			roundNumber:   6,
			expectedValid: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := CheckRedEyeLimit(tt.zombieType, tt.redEyeCount, tt.roundNumber, spawnRules)

			if tt.expectError {
				if err == nil {
					t.Errorf("CheckRedEyeLimit() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CheckRedEyeLimit() unexpected error: %v", err)
				}
			}

			if valid != tt.expectedValid {
				t.Errorf("CheckRedEyeLimit() = %v, want %v (error: %v)", valid, tt.expectedValid, err)
			}
		})
	}
}

func TestCheckSceneTypeRestriction(t *testing.T) {
	spawnRules := &config.SpawnRulesConfig{
		SceneTypeRestrictions: config.SceneRestrictions{
			WaterZombies: []string{"snorkel", "dolphinrider", "ducky"},
			DancingRestrictions: config.DancingRestrictions{
				ProhibitedScenes: []string{"roof"},
			},
			WaterLaneConfig: map[string][]int{
				"pool": {3, 4},
				"fog":  {3, 4},
			},
		},
	}

	tests := []struct {
		name          string
		zombieType    string
		sceneType     string
		lane          int
		expectedValid bool
		expectError   bool
	}{
		{
			name:          "water zombie in water lane (pool scene)",
			zombieType:    "snorkel",
			sceneType:     "pool",
			lane:          3,
			expectedValid: true,
			expectError:   false,
		},
		{
			name:          "water zombie in non-water lane",
			zombieType:    "snorkel",
			sceneType:     "pool",
			lane:          1,
			expectedValid: false,
			expectError:   true,
		},
		{
			name:          "water zombie in scene without water",
			zombieType:    "snorkel",
			sceneType:     "day",
			lane:          1,
			expectedValid: false,
			expectError:   true,
		},
		{
			name:          "non-water zombie in water lane",
			zombieType:    "basic",
			sceneType:     "pool",
			lane:          3,
			expectedValid: false,
			expectError:   true,
		},
		{
			name:          "non-water zombie in non-water lane",
			zombieType:    "basic",
			sceneType:     "pool",
			lane:          1,
			expectedValid: true,
			expectError:   false,
		},
		{
			name:          "dancing zombie in roof scene (prohibited)",
			zombieType:    "dancing",
			sceneType:     "roof",
			lane:          1,
			expectedValid: false,
			expectError:   true,
		},
		{
			name:          "dancing zombie in day scene (allowed)",
			zombieType:    "dancing",
			sceneType:     "day",
			lane:          1,
			expectedValid: true,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := CheckSceneTypeRestriction(tt.zombieType, tt.sceneType, tt.lane, spawnRules)

			if tt.expectError {
				if err == nil {
					t.Errorf("CheckSceneTypeRestriction() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CheckSceneTypeRestriction() unexpected error: %v", err)
				}
			}

			if valid != tt.expectedValid {
				t.Errorf("CheckSceneTypeRestriction() = %v, want %v (error: %v)", valid, tt.expectedValid, err)
			}
		})
	}
}

func TestValidateZombieSpawn(t *testing.T) {
	spawnRules := &config.SpawnRulesConfig{
		RedEyeRules: config.RedEyeRulesConfig{
			StartRound:       5,
			CapacityPerRound: 1,
		},
		SceneTypeRestrictions: config.SceneRestrictions{
			WaterZombies: []string{"snorkel"},
			WaterLaneConfig: map[string][]int{
				"pool": {3, 4},
			},
		},
	}

	tests := []struct {
		name          string
		zombieType    string
		lane          int
		constraint    *components.SpawnConstraintComponent
		roundNumber   int
		expectedValid bool
	}{
		{
			name:       "valid basic zombie at wave 1",
			zombieType: "basic",
			lane:       1,
			constraint: &components.SpawnConstraintComponent{
				RedEyeCount:        0,
				CurrentWaveNum:     1,
				AllowedZombieTypes: []string{"basic", "conehead", "buckethead"},
				SceneType:          "day",
			},
			roundNumber:   0,
			expectedValid: true,
		},
		{
			name:       "zombie not in allowed list",
			zombieType: "gargantuar",
			lane:       1,
			constraint: &components.SpawnConstraintComponent{
				RedEyeCount:        0,
				CurrentWaveNum:     1,
				AllowedZombieTypes: []string{"basic", "conehead"},
				SceneType:          "day",
			},
			roundNumber:   0,
			expectedValid: false,
		},
		{
			name:       "red eye exceeds limit",
			zombieType: "gargantuar_redeye",
			lane:       1,
			constraint: &components.SpawnConstraintComponent{
				RedEyeCount:        1,
				CurrentWaveNum:     15,
				AllowedZombieTypes: []string{"gargantuar_redeye"},
				SceneType:          "day",
			},
			roundNumber:   5,
			expectedValid: false,
		},
		{
			name:       "water zombie in non-water lane",
			zombieType: "snorkel",
			lane:       1,
			constraint: &components.SpawnConstraintComponent{
				RedEyeCount:        0,
				CurrentWaveNum:     8,
				AllowedZombieTypes: []string{"snorkel"},
				SceneType:          "pool",
			},
			roundNumber:   0,
			expectedValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, _ := ValidateZombieSpawn(tt.zombieType, tt.lane, tt.constraint, tt.roundNumber, spawnRules)
			if valid != tt.expectedValid {
				t.Errorf("ValidateZombieSpawn() = %v, want %v", valid, tt.expectedValid)
			}
		})
	}
}
