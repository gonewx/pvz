package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// getProjectRoot returns the project root directory.
// It uses runtime.Caller to find the source file location and navigates up to project root.
func getProjectRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	// This file is at pkg/config/unit_config_test.go
	// Navigate up two directories to reach project root
	dir := filepath.Dir(filename)
	return filepath.Join(dir, "..", "..")
}

// checkSoundFileExists checks if a sound file exists relative to project root.
func checkSoundFileExists(path string) bool {
	projectRoot := getProjectRoot()
	fullPath := filepath.Join(projectRoot, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// TestGameInterfaceSoundPaths tests that game interface sound paths are valid.
// Story 10.9: 音效系统集成 - Task 6.5
func TestGameInterfaceSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"PauseSoundPath", PauseSoundPath},
		{"SeedLiftSoundPath", SeedLiftSoundPath},
		{"CoinSoundPath", CoinSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestLevelResultSoundPaths tests that level result sound paths are valid.
// Story 10.9: 音效系统集成 - Task 2
func TestLevelResultSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"LevelWinMusicPath", LevelWinMusicPath},
		{"LevelLoseMusicPath", LevelLoseMusicPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestWaveWarningSoundPaths tests that wave warning sound paths are valid.
// Story 10.9: 音效系统集成 - Task 3
func TestWaveWarningSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"FinalWaveSoundPath", FinalWaveSoundPath},
		{"HugeWaveSoundPath", HugeWaveSoundPath},
		{"SirenSoundPath", SirenSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestLawnmowerSoundPath tests that lawnmower sound path is valid.
// Story 10.9: 音效系统集成 - Task 4
func TestLawnmowerSoundPath(t *testing.T) {
	if LawnmowerSoundPath == "" {
		t.Error("LawnmowerSoundPath is empty")
		return
	}

	if !checkSoundFileExists(LawnmowerSoundPath) {
		t.Errorf("LawnmowerSoundPath: file does not exist: %s", LawnmowerSoundPath)
	}
}

// TestZombieSoundPaths tests that zombie sound paths are valid.
// Story 10.9: 音效系统集成 - Task 5
func TestZombieSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"ZombieHitSoundPath", ZombieHitSoundPath},
		{"ZombieEatingSoundPath", ZombieEatingSoundPath},
		{"ZombieLimbsPopSoundPath", ZombieLimbsPopSoundPath},
		{"ZombieChompAltSoundPath", ZombieChompAltSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestPlantSoundPaths tests that plant sound paths are valid.
// Story 10.9: 音效系统集成 - Task 7
func TestPlantSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"PotatoMineExplodeSoundPath", PotatoMineExplodeSoundPath},
		{"PuffShroomAttackSoundPath", PuffShroomAttackSoundPath},
		{"FumeShroomAttackSoundPath", FumeShroomAttackSoundPath},
		{"SnowPeaAttackSoundPath", SnowPeaAttackSoundPath},
		{"FrozenEffectSoundPath", FrozenEffectSoundPath},
		{"PlantGrowSoundPath", PlantGrowSoundPath},
		{"CherryBombExplodeSoundPath", CherryBombExplodeSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestArmorHitSoundPaths tests that armor hit sound paths are valid.
// Story 10.9: 音效系统集成 - Task 8
func TestArmorHitSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"ArmorBreakSoundPath", ArmorBreakSoundPath},
		{"ShieldHit2SoundPath", ShieldHit2SoundPath},
		{"BonkSoundPath", BonkSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestEnvironmentSoundPaths tests that environment sound paths are valid.
// Story 10.9: 音效系统集成 - Task 9
func TestEnvironmentSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"GravestoneRumbleSoundPath", GravestoneRumbleSoundPath},
		{"DirtRiseSoundPath", DirtRiseSoundPath},
		{"ThunderSoundPath", ThunderSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestBowlingSoundPaths tests that bowling sound paths are valid.
// Story 19.6-19.8: 保龄球关卡音效
func TestBowlingSoundPaths(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		{"BowlingRollSoundPath", BowlingRollSoundPath},
		{"BowlingImpactSoundPath", BowlingImpactSoundPath},
		{"BowlingImpact2SoundPath", BowlingImpact2SoundPath},
		{"ExplosiveNutExplosionSoundPath", ExplosiveNutExplosionSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name, func(t *testing.T) {
			if sp.path == "" {
				t.Errorf("%s is empty", sp.name)
				return
			}

			if !checkSoundFileExists(sp.path) {
				t.Errorf("%s: file does not exist: %s", sp.name, sp.path)
			}
		})
	}
}

// TestAllSoundPathsFormat tests that all sound paths have correct format.
// Story 10.9: 音效系统集成 - 验证路径格式
func TestAllSoundPathsFormat(t *testing.T) {
	soundPaths := []struct {
		name string
		path string
	}{
		// 游戏界面音效
		{"PauseSoundPath", PauseSoundPath},
		{"SeedLiftSoundPath", SeedLiftSoundPath},
		{"CoinSoundPath", CoinSoundPath},
		// 关卡结算音效
		{"LevelWinMusicPath", LevelWinMusicPath},
		{"LevelLoseMusicPath", LevelLoseMusicPath},
		// 波次警告音效
		{"FinalWaveSoundPath", FinalWaveSoundPath},
		{"HugeWaveSoundPath", HugeWaveSoundPath},
		{"SirenSoundPath", SirenSoundPath},
		// 割草机音效
		{"LawnmowerSoundPath", LawnmowerSoundPath},
		// 僵尸音效
		{"ZombieHitSoundPath", ZombieHitSoundPath},
		{"ZombieEatingSoundPath", ZombieEatingSoundPath},
		{"ZombieLimbsPopSoundPath", ZombieLimbsPopSoundPath},
		{"ZombieChompAltSoundPath", ZombieChompAltSoundPath},
		// 植物音效
		{"PotatoMineExplodeSoundPath", PotatoMineExplodeSoundPath},
		{"PuffShroomAttackSoundPath", PuffShroomAttackSoundPath},
		{"FumeShroomAttackSoundPath", FumeShroomAttackSoundPath},
		{"SnowPeaAttackSoundPath", SnowPeaAttackSoundPath},
		{"FrozenEffectSoundPath", FrozenEffectSoundPath},
		{"PlantGrowSoundPath", PlantGrowSoundPath},
		{"CherryBombExplodeSoundPath", CherryBombExplodeSoundPath},
		// 护甲/碰撞音效
		{"ArmorBreakSoundPath", ArmorBreakSoundPath},
		{"ShieldHit2SoundPath", ShieldHit2SoundPath},
		{"BonkSoundPath", BonkSoundPath},
		// 环境音效
		{"GravestoneRumbleSoundPath", GravestoneRumbleSoundPath},
		{"DirtRiseSoundPath", DirtRiseSoundPath},
		{"ThunderSoundPath", ThunderSoundPath},
	}

	for _, sp := range soundPaths {
		t.Run(sp.name+"_format", func(t *testing.T) {
			// Skip empty paths (some sounds are intentionally disabled)
			if sp.path == "" {
				return
			}

			// Check path starts with assets/sounds/
			expectedPrefix := "assets/sounds/"
			if len(sp.path) < len(expectedPrefix) || sp.path[:len(expectedPrefix)] != expectedPrefix {
				t.Errorf("%s has wrong prefix: got %s, want prefix %s", sp.name, sp.path, expectedPrefix)
			}

			// Check path ends with .ogg
			expectedSuffix := ".ogg"
			if len(sp.path) < len(expectedSuffix) || sp.path[len(sp.path)-len(expectedSuffix):] != expectedSuffix {
				t.Errorf("%s has wrong suffix: got %s, want suffix %s", sp.name, sp.path, expectedSuffix)
			}
		})
	}
}
