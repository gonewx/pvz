package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

var (
	levelID = flag.String("level", "1-2", "Level ID to test (e.g., 1-2)")
	verbose = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// 设置日志输出
	if !*verbose {
		log.SetOutput(os.Stdout)
	}

	log.Printf("=== Verifying Reward System for Level %s ===\n", *levelID)

	// 初始化 ECS
	em := ecs.NewEntityManager()
	gs := game.GetGameState()

	// 加载关卡配置
	log.Printf("Loading level configuration...")
	levelPath := fmt.Sprintf("data/levels/level-%s.yaml", *levelID)
	levelConfig, err := config.LoadLevelConfig(levelPath)
	if err != nil {
		log.Fatalf("❌ FATAL: Failed to load level: %v", err)
	}
	gs.LoadLevel(levelConfig)

	log.Printf("✅ Level loaded: %s", levelConfig.Name)
	log.Printf("   - Reward plant: %s", levelConfig.RewardPlant)
	log.Printf("   - Unlock tools: %v", levelConfig.UnlockTools)
	log.Printf("   - Total waves: %d", len(levelConfig.Waves))

	// 创建资源管理器（最小化初始化）
	log.Printf("\nInitializing resource manager...")
	audioContext := audio.NewContext(48000)
	rm := game.NewResourceManager(audioContext)

	// 加载资源配置文件（CRITICAL：必须在加载任何资源前调用）
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		log.Fatalf("❌ FATAL: Failed to load resource config: %v", err)
	}
	log.Printf("✅ Resource config loaded")

	// 加载 Reanim 资源（奖励动画需要植物 Reanim）
	if err := rm.LoadReanimResources(); err != nil {
		log.Printf("⚠️  WARNING: Failed to load Reanim resources: %v", err)
	} else {
		log.Printf("✅ Reanim resources loaded")
	}

	// 加载必要资源（奖励系统需要）
	if err := rm.LoadResourceGroup("init"); err != nil {
		log.Printf("⚠️  WARNING: Failed to load init resources: %v", err)
	}
	if err := rm.LoadResourceGroup("loadingimages"); err != nil {
		log.Printf("⚠️  WARNING: Failed to load loadingimages: %v", err)
	}

	// 创建核心系统
	log.Printf("Creating core systems...")
	reanimSys := systems.NewReanimSystem(em)
	particleSys := systems.NewParticleSystem(em, rm)
	renderSys := systems.NewRenderSystem(em)
	rewardSys := systems.NewRewardAnimationSystem(em, gs, rm, nil, reanimSys, particleSys, renderSys)

	// 创建 LevelSystem（用于验证系统集成）
	_ = systems.NewLevelSystem(em, gs, nil, rm, reanimSys, rewardSys, nil)

	log.Printf("✅ Systems initialized\n")

	// 模拟胜利条件
	log.Printf("=== Simulating Victory Condition ===")
	log.Printf("Setting up victory state...")

	// 计算总僵尸数
	totalZombies := 0
	for _, wave := range levelConfig.Waves {
		for _, zombieGroup := range wave.Zombies {
			totalZombies += zombieGroup.Count
		}
	}

	gs.ZombiesKilled = totalZombies
	gs.TotalZombiesInLevel = totalZombies // 设置为关卡配置总数
	gs.TotalZombiesSpawned = totalZombies // 假设所有僵尸都已激活

	// 标记所有波次已生成
	for i := range levelConfig.Waves {
		gs.MarkWaveSpawned(i)
	}

	log.Printf("   - Zombies spawned: %d", gs.TotalZombiesSpawned)
	log.Printf("   - Zombies killed: %d", gs.ZombiesKilled)
	log.Printf("   - All waves marked as spawned: %d", len(levelConfig.Waves))

	// 手动调用胜利检查
	log.Printf("\n=== Triggering Victory Check ===")

	// 使用反射调用私有方法 checkVictoryCondition
	// 注意：这是测试代码，生产代码不应使用反射调用私有方法
	if gs.CheckVictory() {
		log.Printf("✅ Victory condition met (CheckVictory returned true)")

		// 手动设置游戏结果
		gs.SetGameResult("win")
		log.Printf("✅ Game result set to 'win'")

		// 手动触发奖励流程（模拟 LevelSystem.checkVictoryCondition）
		if levelConfig.RewardPlant != "" {
			log.Printf("✅ Calling CompleteLevel with rewardPlant: %s", levelConfig.RewardPlant)
			if err := gs.CompleteLevel(*levelID, levelConfig.RewardPlant, levelConfig.UnlockTools); err != nil {
				log.Printf("⚠️  WARNING: CompleteLevel failed: %v", err)
			}
		} else {
			log.Printf("⚠️  WARNING: No rewardPlant configured, calling CompleteLevel with empty plant")
			if err := gs.CompleteLevel(*levelID, "", levelConfig.UnlockTools); err != nil {
				log.Printf("⚠️  WARNING: CompleteLevel failed: %v", err)
			}
		}

		// 检查最后解锁的植物
		lastUnlocked := gs.GetPlantUnlockManager().GetLastUnlocked()
		log.Printf("\n=== Checking Unlock Status ===")
		log.Printf("   - Last unlocked plant: '%s'", lastUnlocked)

		// 手动触发奖励动画（模拟 LevelSystem.triggerRewardIfNeeded）
		if lastUnlocked != "" {
			log.Printf("✅ Triggering reward animation for: %s", lastUnlocked)
			rewardSys.TriggerReward(lastUnlocked)

			// 清除标记
			gs.GetPlantUnlockManager().ClearLastUnlocked()
		} else {
			log.Printf("⚠️  WARNING: No plant unlocked, reward animation NOT triggered")
		}
	} else {
		log.Printf("❌ FAILURE: Victory condition NOT met")
	}

	// 验证结果
	log.Printf("\n=== Verification Results ===")

	success := true

	// 检查1: 配置有效性
	if levelConfig.RewardPlant != "" {
		log.Printf("✅ Level has rewardPlant configured: %s", levelConfig.RewardPlant)
	} else {
		log.Printf("⚠️  WARNING: Level has NO rewardPlant (only unlockTools: %v)", levelConfig.UnlockTools)
		if len(levelConfig.UnlockTools) == 0 {
			log.Printf("❌ FAILURE: Level has neither rewardPlant nor unlockTools!")
			success = false
		}
	}

	// 检查2: 奖励动画系统状态
	if rewardSys.IsActive() {
		log.Printf("✅ Reward animation system is ACTIVE")
	} else {
		log.Printf("❌ FAILURE: Reward animation system is NOT active")
		success = false
	}

	// 检查3: 植物解锁状态
	if levelConfig.RewardPlant != "" {
		if gs.GetPlantUnlockManager().IsUnlocked(levelConfig.RewardPlant) {
			log.Printf("✅ Plant '%s' is unlocked in PlantUnlockManager", levelConfig.RewardPlant)
		} else {
			log.Printf("❌ FAILURE: Plant '%s' is NOT unlocked", levelConfig.RewardPlant)
			success = false
		}
	}

	// 最终结果
	log.Printf("\n=== Final Result ===")
	if success {
		log.Printf("✅ SUCCESS: All checks passed for level %s", *levelID)
		os.Exit(0)
	} else {
		log.Printf("❌ FAILURE: Some checks failed for level %s", *levelID)
		os.Exit(1)
	}
}
