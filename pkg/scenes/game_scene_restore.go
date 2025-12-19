package scenes

import (
	"log"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/config"
	"github.com/gonewx/pvz/pkg/ecs"
	"github.com/gonewx/pvz/pkg/entities"
	"github.com/gonewx/pvz/pkg/game"
	"github.com/gonewx/pvz/pkg/types"
	"github.com/gonewx/pvz/pkg/utils"
)

// ============================================================================
// 状态恢复相关方法
// Story 15.5: 从 game_scene_init.go 拆分出存档恢复逻辑（约900行）
// ============================================================================

func (s *GameScene) restoreBattleState() {
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot restore battle state")
		return
	}

	// 检查是否有存档
	if !saveManager.HasBattleSave(currentUser) {
		log.Printf("[GameScene] No battle save found for user: %s", currentUser)
		return
	}

	// 获取 gdata Manager
	gdataManager := s.gameState.GetGdataManager()
	if gdataManager == nil {
		log.Printf("[GameScene] Warning: gdata Manager not available, cannot restore battle state")
		return
	}

	log.Printf("[GameScene] 开始从战斗存档恢复用户: %s", currentUser)

	// 创建序列化器并加载
	serializer := game.NewBattleSerializer(gdataManager)
	saveData, err := serializer.LoadBattle(currentUser)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to load battle data: %v", err)
		log.Printf("[GameScene] 继续正常游戏...")
		return
	}

	// 验证存档的 LevelID 是否与当前关卡匹配
	currentLevelID := ""
	if s.gameState.CurrentLevel != nil {
		currentLevelID = s.gameState.CurrentLevel.ID
	}
	if saveData.LevelID != "" && saveData.LevelID != currentLevelID {
		log.Printf("[GameScene] 存档关卡不匹配: %s vs %s，删除不匹配存档",
			saveData.LevelID, currentLevelID)
		if err := saveManager.DeleteBattleSave(currentUser); err != nil {
			log.Printf("[GameScene] Warning: Failed to delete mismatched battle save: %v", err)
		}
		return
	}

	// 恢复游戏状态
	s.gameState.Sun = saveData.Sun
	s.gameState.LevelTime = saveData.LevelTime
	s.gameState.CurrentWaveIndex = saveData.CurrentWaveIndex
	if len(saveData.SpawnedWaves) > 0 {
		s.gameState.SpawnedWaves = make([]bool, len(saveData.SpawnedWaves))
		copy(s.gameState.SpawnedWaves, saveData.SpawnedWaves)
	}
	s.gameState.TotalZombiesSpawned = saveData.TotalZombiesSpawned
	s.gameState.ZombiesKilled = saveData.ZombiesKilled

	// Story 8.12: 恢复选卡系统选择的植物列表
	if len(saveData.SelectedPlants) > 0 {
		s.gameState.SetSelectedPlants(saveData.SelectedPlants)
		log.Printf("[GameScene] 恢复选卡植物列表: %v", saveData.SelectedPlants)
	}

	log.Printf("[GameScene] 状态恢复: Sun=%d, Wave=%d, Time=%.1f, Killed=%d/%d",
		s.gameState.Sun, s.gameState.CurrentWaveIndex, s.gameState.LevelTime,
		s.gameState.ZombiesKilled, s.gameState.TotalZombiesInLevel)

	s.restorePlants(saveData.Plants)
	s.restoreZombies(saveData.Zombies)
	s.restoreProjectiles(saveData.Projectiles)
	s.restoreSuns(saveData.Suns)
	s.restoreLawnmowers(saveData.Lawnmowers)
	s.restorePlantCards(saveData.PlantCards)

	// v7: 恢复已用除草车行号状态（解决存档恢复后僵尸胜利条件失效问题）
	s.restoreUsedLawnmowerLanes(saveData.UsedLawnmowerLanes)

	log.Printf("[GameScene] 实体恢复: Plants=%d, Zombies=%d, Suns=%d",
		len(saveData.Plants), len(saveData.Zombies), len(saveData.Suns))

	s.isIntroAnimPlaying = false
	s.cameraX = config.GameCameraX
	if s.cameraSystem != nil {
		s.gameState.CameraX = config.GameCameraX
	}

	s.soddingAnimStarted = true
	s.seedBankSlideInStarted = true
	s.seedBankSlideInProgress = 1.0
	s.seedBankSlideInCompleted = true

	s.restoreProgressBar(saveData)

	if saveData.Tutorial != nil {
		s.restoreTutorialState(saveData.Tutorial)
	}

	// 恢复波次计时系统状态
	if s.levelSystem != nil {
		waveTimingSystem := s.levelSystem.GetWaveTimingSystem()
		if waveTimingSystem != nil {
			waveTimingSystem.RestoreState(saveData.CurrentWaveIndex, saveData.LevelTime)

			if saveData.WaveSystemState != nil {
				log.Printf("[GameScene] 波次系统状态: IsPaused=%v, ControlledBy=%s",
					saveData.WaveSystemState.IsPaused, saveData.WaveSystemState.ControlledBy)

				// 教学控制的波次不恢复 CountdownCs
				if saveData.WaveSystemState.ControlledBy != "tutorial" {
					waveTimingSystem.RestoreTimerState(
						saveData.WaveSystemState.CountdownCs,
						saveData.WaveSystemState.WaveElapsedCs,
						saveData.WaveSystemState.IsFlagWaveApproaching,
						saveData.WaveSystemState.IsFinalWave,
					)
				}

				// 只有非 timer 控制的波次系统才恢复暂停状态
				if saveData.WaveSystemState.IsPaused && saveData.WaveSystemState.ControlledBy != "timer" {
					waveTimingSystem.Pause()
				}
			} else {
				// 旧存档兼容
				shouldPause := s.inferWaveSystemPauseState(saveData)
				if shouldPause {
					waveTimingSystem.Pause()
				} else {
					zombiesOnField := s.gameState.TotalZombiesSpawned - s.gameState.ZombiesKilled
					hasMoreWaves := saveData.CurrentWaveIndex < len(s.gameState.SpawnedWaves)
					if zombiesOnField == 0 && hasMoreWaves {
						waveTimingSystem.TriggerNextWaveImmediately()
					}
				}
			}
		}
	}

	s.restoreBowlingNuts(saveData.BowlingNuts)
	s.restoreConveyorBelt(saveData.ConveyorBelt)
	s.restoreLevelPhase(saveData.LevelPhase)
	s.restoreDaveDialogue(saveData.DaveDialogue)
	s.restoreGuidedTutorial(saveData.GuidedTutorial)

	log.Printf("[GameScene] 战斗状态恢复完成")
}

// inferWaveSystemPauseState 推断波次系统是否应该暂停（旧存档兼容）
func (s *GameScene) inferWaveSystemPauseState(saveData *game.BattleSaveData) bool {
	if s.gameState.CurrentLevel == nil {
		return false
	}

	openingType := s.gameState.CurrentLevel.OpeningType

	if openingType == "tutorial" {
		if saveData.Tutorial != nil && saveData.Tutorial.IsActive {
			return true
		}
	}

	if openingType == "special" {
		if saveData.LevelPhase != nil {
			if saveData.LevelPhase.CurrentPhase < 2 ||
				saveData.LevelPhase.PhaseState == components.PhaseStateTransitioning {
				return true
			}
		}
		if saveData.LevelPhase == nil {
			return true
		}
	}

	return false
}

// restorePlants 恢复植物实体
func (s *GameScene) restorePlants(plants []game.PlantData) {
	deps := entities.PlantFactoryDeps{
		EntityManager:  s.entityManager,
		ResourceLoader: s.resourceManager,
		GameState:      s.gameState,
		ReanimSystem:   s.reanimSystem,
	}

	for _, plantData := range plants {
		// 跳过正在爆炸的土豆地雷
		// 爆炸已触发、伤害已造成、网格已释放，不需要恢复这个即将消失的实体
		if plantData.PlantType == types.PlantPotatoMine.String() &&
			plantData.PotatoMinePhase == int(components.PotatoMineExploding) {
			log.Printf("[GameScene] Skipping exploding potato mine at (%d,%d), phase=%d",
				plantData.GridRow, plantData.GridCol, plantData.PotatoMinePhase)
			continue
		}
		var entityID ecs.EntityID
		var err error

		factory, found := entities.GetPlantFactory(plantData.PlantType)
		if !found {
			log.Printf("[GameScene] Unknown plant type '%s', using default factory", plantData.PlantType)
			factory = entities.GetDefaultPlantFactory()
		}
		entityID, err = factory(deps, plantData.GridCol, plantData.GridRow)

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore plant %s at (%d,%d): %v",
				plantData.PlantType, plantData.GridRow, plantData.GridCol, err)
			continue
		}

		// 恢复生命值
		if healthComp, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			healthComp.CurrentHealth = plantData.Health
			healthComp.MaxHealth = plantData.MaxHealth
		}

		// 恢复计时器状态
		if timerComp, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID); ok {
			if plantData.TimerTargetTime > 0 {
				timerComp.TargetTime = plantData.TimerTargetTime
			}
			timerComp.CurrentTime = timerComp.TargetTime - plantData.AttackCooldown
			if timerComp.CurrentTime < 0 {
				timerComp.CurrentTime = 0
			}
			timerComp.IsReady = plantData.AttackCooldown <= 0
		}

		if plantData.PlantType == types.PlantPotatoMine.String() {
			if plantComp, ok := ecs.GetComponent[*components.PlantComponent](s.entityManager, entityID); ok {
				plantComp.PotatoMinePhase = components.PotatoMinePhase(plantData.PotatoMinePhase)
				plantComp.ArmingTimer = plantData.ArmingTimer

				// 根据阶段触发正确的动画
				switch plantComp.PotatoMinePhase {
				case components.PotatoMineArming:
					// 武装阶段：播放 anim_idle（埋在地下）
					ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
						UnitID:        "potatomine",
						AnimationName: "anim_idle",
						Processed:     false,
					})
				case components.PotatoMineRising:
					// 升起阶段：播放 anim_rise
					ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
						UnitID:        "potatomine",
						AnimationName: "anim_rise",
						Processed:     false,
					})
				case components.PotatoMineArmed:
					ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
						UnitID:    "potatomine",
						ComboName: "idle",
						Processed: false,
					})
				case components.PotatoMineExploding:
					ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
						UnitID:        "potatomine",
						AnimationName: "anim_mashed",
						Processed:     false,
					})
				}

				log.Printf("[GameScene] Restored potato mine at (%d,%d): Phase=%d, ArmingTimer=%.2f",
					plantData.GridRow, plantData.GridCol, plantData.PotatoMinePhase, plantData.ArmingTimer)
			}
		}

		// 更新草坪网格占用状态
		if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
			if err := s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, plantData.GridCol, plantData.GridRow, entityID); err != nil {
				log.Printf("[GameScene] Warning: Failed to occupy grid cell (%d,%d): %v",
					plantData.GridCol, plantData.GridRow, err)
			}
		}

	}
}

// restoreZombies 恢复僵尸实体
func (s *GameScene) restoreZombies(zombies []game.ZombieData) {
	for _, zombieData := range zombies {
		// 跳过正在死亡的僵尸
		if zombieData.BehaviorType == "dying" || zombieData.BehaviorType == "dying_explosion" {
			log.Printf("[GameScene] Skipping dying zombie at (%.1f, %.1f), incrementing ZombiesKilled", zombieData.X, zombieData.Y)
			s.gameState.ZombiesKilled++
			continue
		}

		// 计算行号（从 Y 坐标推算，如果 Lane 未设置）
		lane := zombieData.Lane
		if lane == 0 {
			// 从 Y 坐标推算行号
			lane = int((zombieData.Y-config.GridWorldStartY)/config.CellHeight) + 1
			if lane < 1 {
				lane = 1
			}
			if lane > 5 {
				lane = 5
			}
		}

		var entityID ecs.EntityID
		var err error

		factory, found := entities.GetZombieFactory(zombieData.ZombieType)
		if !found {
			log.Printf("[GameScene] Unknown zombie type %s, using default factory", zombieData.ZombieType)
			factory = entities.GetDefaultZombieFactory()
		}
		entityID, err = factory(s.entityManager, s.resourceManager, lane-1, zombieData.X)

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore zombie %s at (%.1f, %.1f): %v",
				zombieData.ZombieType, zombieData.X, zombieData.Y, err)
			continue
		}

		// 恢复位置（Y 坐标可能需要调整）
		if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
			posComp.X = zombieData.X
			posComp.Y = zombieData.Y
		}

		// 恢复生命值
		if healthComp, ok := ecs.GetComponent[*components.HealthComponent](s.entityManager, entityID); ok {
			healthComp.CurrentHealth = zombieData.Health
			healthComp.MaxHealth = zombieData.MaxHealth
		}

		// 恢复护甲值（如果有）
		if zombieData.ArmorHealth > 0 {
			if armorComp, ok := ecs.GetComponent[*components.ArmorComponent](s.entityManager, entityID); ok {
				armorComp.CurrentArmor = zombieData.ArmorHealth
				armorComp.MaxArmor = zombieData.ArmorMax
			}
		}

		if zombieData.ZombieType == types.UnitIDZombiePolevaulter {
			if poleVault, ok := ecs.GetComponent[*components.PoleVaultComponent](s.entityManager, entityID); ok {
				poleVault.HasPole = zombieData.HasPole
				poleVault.IsJumping = zombieData.IsJumping
			}
		}

		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
			if zombieData.VelocityX != 0 {
				velComp.VX = zombieData.VelocityX
			} else {
				if zombieData.ZombieType == types.UnitIDZombiePolevaulter && zombieData.HasPole {
					velComp.VX = config.PolevaulterZombieRunSpeed
				} else {
					velComp.VX = config.ZombieWalkSpeed
				}
			}
		}

		if behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
			if zombieData.IsEating {
				behaviorComp.ZombieAnimState = components.ZombieAnimEating
			} else {
				behaviorComp.ZombieAnimState = components.ZombieAnimWalking
			}
		}

		unitID := zombieData.ZombieType
		comboName := "walk"

		// 撑杆僵尸持杆时使用 run 动画
		if zombieData.ZombieType == types.UnitIDZombiePolevaulter && zombieData.HasPole {
			comboName = "run"
		}

		if zombieData.IsEating {
			comboName = "eat"
		}

		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    unitID,
			ComboName: comboName,
			Processed: false,
		})

		ecs.AddComponent(s.entityManager, entityID, &components.ZombieTargetLaneComponent{
			TargetRow: lane - 1,
		})
	}
}

// restoreProjectiles 恢复子弹实体
func (s *GameScene) restoreProjectiles(projectiles []game.ProjectileData) {
	for _, projData := range projectiles {
		// 目前只支持豌豆子弹
		if projData.Type != "pea" {
			log.Printf("[GameScene] Warning: Unsupported projectile type '%s', skipping", projData.Type)
			continue
		}

		laneIndex := utils.GetEntityRow(projData.Y, config.GridWorldStartY, config.CellHeight)
		entityID, err := entities.NewPeaProjectile(s.entityManager, s.resourceManager, projData.X, projData.Y, laneIndex)
		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore projectile at (%.1f, %.1f): %v", projData.X, projData.Y, err)
			continue
		}

		// 恢复速度（如果保存了不同的速度）
		if projData.VelocityX != 0 {
			if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
				velComp.VX = projData.VelocityX
			}
		}

		log.Printf("[GameScene] Restored projectile at (%.1f, %.1f)", projData.X, projData.Y)
	}
}

// restoreSuns 恢复阳光实体
func (s *GameScene) restoreSuns(suns []game.SunData) {
	for _, sunData := range suns {
		if sunData.IsCollecting {
			log.Printf("[GameScene] Skipping collecting sun at (%.1f, %.1f)", sunData.X, sunData.Y)
			continue
		}

		// 创建静态阳光实体（已着陆状态）
		entityID := entities.NewSunEntityStatic(s.entityManager, s.resourceManager, sunData.X, sunData.Y)

		// 添加动画命令组件，让 ReanimSystem 初始化阳光动画
		ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
			UnitID:    "sun",
			ComboName: "idle",
			Processed: false,
		})

		// 恢复剩余生命周期
		if lifetimeComp, ok := ecs.GetComponent[*components.LifetimeComponent](s.entityManager, entityID); ok {
			if sunData.Lifetime > 0 {
				// 设置当前生命周期为 (最大 - 剩余)
				lifetimeComp.CurrentLifetime = lifetimeComp.MaxLifetime - sunData.Lifetime
				if lifetimeComp.CurrentLifetime < 0 {
					lifetimeComp.CurrentLifetime = 0
				}
			}
		}

		if sunComp, ok := ecs.GetComponent[*components.SunComponent](s.entityManager, entityID); ok {
			sunComp.State = components.SunLanded
		}

		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
			velComp.VY = 0
		}

		log.Printf("[GameScene] Restored sun at (%.1f, %.1f), lifetime=%.1f", sunData.X, sunData.Y, sunData.Lifetime)
	}
}

// restoreLawnmowers 恢复除草车实体
func (s *GameScene) restoreLawnmowers(lawnmowers []game.LawnmowerData) {
	// 首先检查是否已经通过 initLawnmowers 创建了除草车
	// 如果存档中有除草车数据，我们需要先清理默认创建的除草车
	existingLawnmowers := ecs.GetEntitiesWith1[*components.LawnmowerComponent](s.entityManager)
	for _, entityID := range existingLawnmowers {
		s.entityManager.DestroyEntity(entityID)
	}

	for _, lmData := range lawnmowers {
		// 跳过已触发且激活的除草车（已经移出屏幕）
		if lmData.Triggered && lmData.Active && lmData.X > float64(WindowWidth)+100 {
			log.Printf("[GameScene] Skipping lawnmower on lane %d (already moved off-screen)", lmData.Lane)
			// 标记该 lane 为已使用（修复：存档时割草机正在移动但还未被系统删除的情况）
			s.markLawnmowerLaneAsUsed(lmData.Lane)
			continue
		}

		// 创建除草车实体
		entityID, err := entities.NewLawnmowerEntity(s.entityManager, s.resourceManager, lmData.Lane)
		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore lawnmower on lane %d: %v", lmData.Lane, err)
			continue
		}

		// 恢复位置
		if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
			posComp.X = lmData.X
		}

		if lmComp, ok := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, entityID); ok {
			lmComp.IsTriggered = lmData.Triggered
			lmComp.IsMoving = lmData.Active
			lmComp.IsEntering = false
		}

		if reanimComp, ok := ecs.GetComponent[*components.ReanimComponent](s.entityManager, entityID); ok {
			reanimComp.IsPaused = !lmData.Active
		}

		if lmData.Active {
			if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
				velComp.VX = config.LawnmowerSpeed
			} else {
				s.entityManager.AddComponent(entityID, &components.VelocityComponent{
					VX: config.LawnmowerSpeed,
					VY: 0,
				})
			}
		}

		log.Printf("[GameScene] Restored lawnmower on lane %d at X=%.1f, triggered=%v, active=%v",
			lmData.Lane, lmData.X, lmData.Triggered, lmData.Active)
	}
}

// restoreUsedLawnmowerLanes 恢复已用除草车行号状态
//
// v7 新增：解决存档恢复后僵尸胜利条件失效问题
//
// 当除草车被触发并离开屏幕后，实体被删除，但 UsedLanes 状态需要被保存和恢复。
// 这个状态用于失败条件判断：如果某行的除草车已经用过，僵尸再次进入该行会导致游戏失败。
func (s *GameScene) restoreUsedLawnmowerLanes(usedLanes []int) {
	if len(usedLanes) == 0 {
		return
	}

	// 查找 LawnmowerStateComponent 实体
	stateEntities := ecs.GetEntitiesWith1[*components.LawnmowerStateComponent](s.entityManager)
	if len(stateEntities) == 0 {
		log.Printf("[GameScene] Warning: No LawnmowerStateComponent found, cannot restore used lanes")
		return
	}

	// 取第一个状态实体（由 LawnmowerSystem 创建）
	stateEntity := stateEntities[0]
	stateComp, ok := ecs.GetComponent[*components.LawnmowerStateComponent](s.entityManager, stateEntity)
	if !ok {
		log.Printf("[GameScene] Warning: Failed to get LawnmowerStateComponent")
		return
	}

	// 恢复 UsedLanes
	if stateComp.UsedLanes == nil {
		stateComp.UsedLanes = make(map[int]bool)
	}

	for _, lane := range usedLanes {
		stateComp.UsedLanes[lane] = true
	}

	log.Printf("[GameScene] Restored used lawnmower lanes: %v", usedLanes)
}

// markLawnmowerLaneAsUsed 标记指定行的除草车为已使用
//
// 用于处理存档恢复时割草机已跑出屏幕的情况：
// 存档时割草机可能正在移动但还未被系统删除，此时 UsedLanes 中还没有该 lane。
// 恢复时跳过创建实体后，需要手动标记该 lane 为已使用，否则僵尸到达边界时
// 会尝试触发不存在的割草机，导致每帧重复打印日志。
func (s *GameScene) markLawnmowerLaneAsUsed(lane int) {
	// 查找 LawnmowerStateComponent 实体
	stateEntities := ecs.GetEntitiesWith1[*components.LawnmowerStateComponent](s.entityManager)
	if len(stateEntities) == 0 {
		log.Printf("[GameScene] Warning: No LawnmowerStateComponent found, cannot mark lane %d as used", lane)
		return
	}

	stateEntity := stateEntities[0]
	stateComp, ok := ecs.GetComponent[*components.LawnmowerStateComponent](s.entityManager, stateEntity)
	if !ok {
		log.Printf("[GameScene] Warning: Failed to get LawnmowerStateComponent")
		return
	}

	if stateComp.UsedLanes == nil {
		stateComp.UsedLanes = make(map[int]bool)
	}

	stateComp.UsedLanes[lane] = true
	log.Printf("[GameScene] Marked lawnmower lane %d as used (off-screen at restore time)", lane)
}

// restoreProgressBar 恢复进度条数据
func (s *GameScene) restoreProgressBar(saveData *game.BattleSaveData) {
	if s.levelProgressBarEntity == 0 {
		log.Printf("[GameScene] Warning: No progress bar entity, cannot restore progress bar data")
		return
	}

	progressBar, ok := ecs.GetComponent[*components.LevelProgressBarComponent](s.entityManager, s.levelProgressBarEntity)
	if !ok {
		log.Printf("[GameScene] Warning: LevelProgressBarComponent not found for entity %d", s.levelProgressBarEntity)
		return
	}

	// 恢复已击杀僵尸数
	progressBar.KilledZombies = saveData.ZombiesKilled

	// 恢复当前波次号（存档中是索引，从0开始；波次号从1开始）
	progressBar.CurrentWaveNum = saveData.CurrentWaveIndex + 1

	// 计算进度百分比
	if progressBar.TotalZombies > 0 {
		progressBar.ProgressPercent = float64(saveData.ZombiesKilled) / float64(progressBar.TotalZombies)
		progressBar.VirtualProgress = progressBar.ProgressPercent
		progressBar.RealProgress = progressBar.ProgressPercent
		progressBar.GameTickCS = int(saveData.LevelTime * 100)
		progressBar.LastTrackUpdateCS = progressBar.GameTickCS
	}

	if saveData.CurrentWaveIndex > 0 || saveData.ZombiesKilled > 0 || len(saveData.Zombies) > 0 {
		progressBar.ShowLevelTextOnly = false
	}

	log.Printf("[GameScene] 进度条恢复: %d/%d zombies, wave %d",
		progressBar.KilledZombies, progressBar.TotalZombies, progressBar.CurrentWaveNum)
}

// restoreTutorialState 恢复教学状态
func (s *GameScene) restoreTutorialState(tutorialData *game.TutorialSaveData) {
	if tutorialData == nil {
		return
	}

	// 查找教学组件
	tutorialEntities := ecs.GetEntitiesWith1[*components.TutorialComponent](s.entityManager)
	if len(tutorialEntities) == 0 {
		log.Printf("[GameScene] Warning: No tutorial entity found, cannot restore tutorial state")
		return
	}

	// 取第一个教学实体（通常只有一个）
	tutorialEntity := tutorialEntities[0]
	tutorial, ok := ecs.GetComponent[*components.TutorialComponent](s.entityManager, tutorialEntity)
	if !ok {
		log.Printf("[GameScene] Warning: TutorialComponent not found")
		return
	}

	// 恢复教学状态
	tutorial.CurrentStepIndex = tutorialData.CurrentStepIndex
	tutorial.IsActive = tutorialData.IsActive

	// 恢复已完成的步骤
	if tutorialData.CompletedSteps != nil {
		tutorial.CompletedSteps = make(map[string]bool)
		for k, v := range tutorialData.CompletedSteps {
			tutorial.CompletedSteps[k] = v
		}
	}

	if tutorialData.CompletedSteps["plantPlaced"] || tutorialData.PlantCount > 0 {
		if s.lawnGridSystem != nil {
			s.lawnGridSystem.DisableFlash()
		}
	}

	if s.tutorialSystem != nil && tutorialData.PlantCount > 0 {
		s.tutorialSystem.SetPlantCount(tutorialData.PlantCount)
	}

	if s.tutorialSystem != nil && tutorialData.IsActive {
		s.tutorialSystem.ResetTextDisplayTimer()
	}

}

// restoreBowlingNuts 恢复保龄球坚果实体
func (s *GameScene) restoreBowlingNuts(bowlingNuts []game.BowlingNutData) {
	if len(bowlingNuts) == 0 {
		return
	}

	log.Printf("[GameScene] 恢复 %d 个保龄球坚果...", len(bowlingNuts))

	for _, nutData := range bowlingNuts {
		// 使用工厂函数创建（col=0 作为临时值，后续会覆盖位置）
		entityID, err := entities.NewBowlingNutEntity(
			s.entityManager,
			s.resourceManager,
			nutData.Row,
			0, // 临时 col，位置会被覆盖
			nutData.IsExplosive,
		)

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to restore bowling nut at (%.1f, %.1f): %v",
				nutData.X, nutData.Y, err)
			continue
		}

		// 恢复位置（覆盖工厂函数计算的默认位置）
		if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, entityID); ok {
			posComp.X = nutData.X
			posComp.Y = nutData.Y
		}

		// 恢复保龄球坚果组件状态
		if bowlingComp, ok := ecs.GetComponent[*components.BowlingNutComponent](s.entityManager, entityID); ok {
			bowlingComp.VelocityX = nutData.VelocityX
			bowlingComp.VelocityY = nutData.VelocityY
			bowlingComp.Row = nutData.Row
			bowlingComp.IsRolling = nutData.IsRolling
			bowlingComp.IsBouncing = nutData.IsBouncing
			bowlingComp.TargetRow = nutData.TargetRow
			bowlingComp.BounceCount = nutData.BounceCount
			bowlingComp.CollisionCooldown = nutData.CollisionCooldown
			bowlingComp.BounceDirection = nutData.BounceDirection
		}

		log.Printf("[GameScene] Restored bowling nut at (%.1f, %.1f), row=%d, explosive=%v, bouncing=%v",
			nutData.X, nutData.Y, nutData.Row, nutData.IsExplosive, nutData.IsBouncing)
	}
}

// restoreConveyorBelt 恢复传送带状态
func (s *GameScene) restoreConveyorBelt(conveyorData *game.ConveyorBeltData) {
	if conveyorData == nil {
		return
	}

	log.Printf("[GameScene] 恢复传送带状态...")

	// 查找传送带组件
	conveyorEntities := ecs.GetEntitiesWith1[*components.ConveyorBeltComponent](s.entityManager)
	if len(conveyorEntities) == 0 {
		log.Printf("[GameScene] Warning: No conveyor belt entity found, cannot restore")
		return
	}

	conveyorEntity := conveyorEntities[0]
	conveyorComp, ok := ecs.GetComponent[*components.ConveyorBeltComponent](s.entityManager, conveyorEntity)
	if !ok {
		log.Printf("[GameScene] Warning: ConveyorBeltComponent not found")
		return
	}

	// 恢复卡片队列
	conveyorComp.Cards = make([]components.ConveyorCard, len(conveyorData.Cards))
	for i, cardData := range conveyorData.Cards {
		conveyorComp.Cards[i] = components.ConveyorCard{
			CardType:  cardData.CardType,
			PositionX: cardData.PositionX,
			IsStopped: cardData.IsStopped,
		}
	}

	// 恢复状态
	conveyorComp.Capacity = conveyorData.Capacity
	conveyorComp.ScrollOffset = conveyorData.ScrollOffset
	conveyorComp.IsActive = conveyorData.IsActive
	conveyorComp.GenerationTimer = conveyorData.GenerationTimer
	conveyorComp.GenerationInterval = conveyorData.GenerationInterval
	conveyorComp.SelectedCardIndex = conveyorData.SelectedCardIndex
	conveyorComp.FinalWaveTriggered = conveyorData.FinalWaveTriggered

}

// restoreLevelPhase 恢复关卡阶段状态
func (s *GameScene) restoreLevelPhase(phaseData *game.LevelPhaseData) {
	if phaseData == nil {
		return
	}

	log.Printf("[GameScene] 恢复关卡阶段状态...")

	// 查找关卡阶段组件
	phaseEntities := ecs.GetEntitiesWith1[*components.LevelPhaseComponent](s.entityManager)
	if len(phaseEntities) == 0 {
		log.Printf("[GameScene] Warning: No level phase entity found, cannot restore")
		return
	}

	phaseEntity := phaseEntities[0]
	phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, phaseEntity)
	if !ok {
		log.Printf("[GameScene] Warning: LevelPhaseComponent not found")
		return
	}

	// 恢复状态
	phaseComp.CurrentPhase = phaseData.CurrentPhase
	phaseComp.PhaseState = phaseData.PhaseState
	phaseComp.TransitionProgress = phaseData.TransitionProgress
	phaseComp.TransitionStep = phaseData.TransitionStep
	phaseComp.ConveyorBeltY = phaseData.ConveyorBeltY
	phaseComp.ConveyorBeltVisible = phaseData.ConveyorBeltVisible
	phaseComp.ShowRedLine = phaseData.ShowRedLine

	// 跳过转场动画
	if phaseComp.ConveyorBeltVisible {
		phaseComp.ConveyorBeltY = config.ConveyorBeltTargetY
		phaseComp.TransitionProgress = 1.0
		if phaseComp.TransitionStep == components.TransitionStepConveyorSlide {
			phaseComp.TransitionStep = components.TransitionStepShowRedLine
			phaseComp.ShowRedLine = true
		}
	}

	// Phase 2 激活状态下启动相关系统
	if phaseComp.CurrentPhase == 2 && phaseComp.PhaseState == components.PhaseStateActive {
		if s.conveyorBeltSystem != nil {
			s.conveyorBeltSystem.Activate()
		}
		if s.plantPreviewRenderSystem != nil {
			s.plantPreviewRenderSystem.SetRedLineEnabled(true)
		}
	}
}

// restoreDaveDialogue 恢复 Dave 对话状态
//
// Story 19.x: 从存档数据恢复 Dave 对话状���
//
// 恢复内容：
//   - 对话进度（当前行索引）
//   - 对话文本
//   - Dave 状态和表情
//   - 位置
//
// 特殊处理：
//   - 先删除初始化时创建的 Dave 实体（避免重复）
//   - 如果 Dave 已经离开（State == Hidden），删除后不重新创建
//   - 如果 Dave 正在对话中，恢复对话进度
func (s *GameScene) restoreDaveDialogue(daveData *game.DaveDialogueData) {
	// 首先删除所有现有的 Dave 对话实体（初始化时可能已创建）
	existingDaves := ecs.GetEntitiesWith1[*components.DaveDialogueComponent](s.entityManager)
	for _, entityID := range existingDaves {
		s.entityManager.DestroyEntity(entityID)
		log.Printf("[GameScene] 删除现有 Dave 实体: %d", entityID)
	}

	if daveData == nil {
		log.Printf("[GameScene] 无 Dave 对话数据，已清理现有实体")
		return
	}

	if daveData.State == int(components.DaveStateHidden) {
		return
	}

	log.Printf("[GameScene] 恢复 Dave 对话状态")

	isTransitionDave := false
	phaseEntities := ecs.GetEntitiesWith1[*components.LevelPhaseComponent](s.entityManager)
	if len(phaseEntities) > 0 {
		if phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, phaseEntities[0]); ok {
			if phaseComp.PhaseState == components.PhaseStateTransitioning &&
				phaseComp.TransitionStep == components.TransitionStepDaveDialogue {
				isTransitionDave = true
			}
		}
	}

	var onComplete func()
	if isTransitionDave {
		onComplete = func() {
			if s.levelPhaseSystem != nil {
				s.levelPhaseSystem.AdvanceToConveyorSlide()
			}
		}
	} else {
		onComplete = func() {
			if s.guidedTutorialSystem != nil {
				s.guidedTutorialSystem.SetActive(true)
			}
		}
	}

	daveEntity, err := entities.NewCrazyDaveEntity(
		s.entityManager,
		s.resourceManager,
		daveData.DialogueKeys,
		onComplete,
	)

	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to restore Dave entity: %v", err)
		return
	}

	if daveComp, ok := ecs.GetComponent[*components.DaveDialogueComponent](s.entityManager, daveEntity); ok {
		daveComp.CurrentLineIndex = daveData.CurrentLineIndex
		daveComp.CurrentText = daveData.CurrentText
		daveComp.IsVisible = daveData.IsVisible
		daveComp.State = components.DaveState(daveData.State)
		daveComp.Expression = daveData.Expression
	}

	if posComp, ok := ecs.GetComponent[*components.PositionComponent](s.entityManager, daveEntity); ok {
		posComp.X = daveData.DaveX
		posComp.Y = daveData.DaveY
	}

	if daveData.State == int(components.DaveStateTalking) {
		if animCmd, ok := ecs.GetComponent[*components.AnimationCommandComponent](s.entityManager, daveEntity); ok {
			animCmd.ComboName = "anim_smalltalk"
			animCmd.Processed = false
		}
	}

	if isTransitionDave && len(phaseEntities) > 0 {
		if phaseComp, ok := ecs.GetComponent[*components.LevelPhaseComponent](s.entityManager, phaseEntities[0]); ok {
			phaseComp.DaveDialogueEntityID = int(daveEntity)
		}
	}
}

// restoreGuidedTutorial 恢复强引导教学状态
func (s *GameScene) restoreGuidedTutorial(guidedData *game.GuidedTutorialData) {
	if guidedData == nil {
		return
	}

	log.Printf("[GameScene] 恢复强引导教学状态...")

	// 查找强引导教学组件
	guidedEntities := ecs.GetEntitiesWith1[*components.GuidedTutorialComponent](s.entityManager)
	if len(guidedEntities) == 0 {
		log.Printf("[GameScene] Warning: No guided tutorial entity found, cannot restore")
		return
	}

	guidedEntity := guidedEntities[0]
	guidedComp, ok := ecs.GetComponent[*components.GuidedTutorialComponent](s.entityManager, guidedEntity)
	if !ok {
		log.Printf("[GameScene] Warning: GuidedTutorialComponent not found")
		return
	}

	// 恢复状态
	guidedComp.IsActive = guidedData.IsActive
	guidedComp.AllowedActions = make([]string, len(guidedData.AllowedActions))
	copy(guidedComp.AllowedActions, guidedData.AllowedActions)
	guidedComp.IdleTimer = guidedData.IdleTimer
	guidedComp.IdleThreshold = guidedData.IdleThreshold
	guidedComp.ShowArrow = guidedData.ShowArrow
	guidedComp.ArrowTarget = guidedData.ArrowTarget
	guidedComp.LastPlantCount = guidedData.LastPlantCount
	guidedComp.TransitionReady = guidedData.TransitionReady
	guidedComp.TutorialTextKey = guidedData.TutorialTextKey
}

// restorePlantCards 恢复植物卡片状态
//
// 恢复卡片的冷却时间和就绪状态，确保存档恢复后卡片可用性正确
func (s *GameScene) restorePlantCards(plantCards []game.PlantCardData) {
	if len(plantCards) == 0 {
		return
	}

	// 创建卡片状态映射
	cardStateMap := make(map[string]game.PlantCardData)
	for _, cardData := range plantCards {
		cardStateMap[cardData.PlantType] = cardData
	}

	cardEntities := ecs.GetEntitiesWith1[*components.PlantCardComponent](s.entityManager)
	for _, entity := range cardEntities {
		cardComp, ok := ecs.GetComponent[*components.PlantCardComponent](s.entityManager, entity)
		if !ok {
			continue
		}

		plantTypeStr := cardComp.PlantType.String()
		if cardState, exists := cardStateMap[plantTypeStr]; exists {
			cardComp.CurrentCooldown = cardState.CurrentCooldown
			// 如果卡片已就绪（冷却完成），暂时标记为可用
			// 实际可用性将由 PlantCardSystem 根据阳光数量重新计算
			if cardState.IsReady {
				cardComp.CurrentCooldown = 0
			}
			log.Printf("[GameScene] 恢复卡片 %s: cooldown=%.2f, isReady=%v",
				plantTypeStr, cardComp.CurrentCooldown, cardState.IsReady)
		}
	}
}
