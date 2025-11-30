package scenes

import (
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/modules"
	"github.com/decker502/pvz/pkg/systems"
)

// initPlantCardSystems initializes the plant selection module.
// Story 3.1 架构优化：使用 PlantSelectionModule 统一管理所有选卡功能
// Story 8.3: 使用 PlantUnlockManager 统一管理植物可用性
//
// 重构说明：
//   - 旧方式：直接在 GameScene 中创建卡片实体和系统（分散）
//   - 新方式：使用 PlantSelectionModule 统一封装（内聚）
//
// 优点：
//   - 高内聚：所有选卡功能封装在单一模块中
//   - 低耦合：GameScene 只通过模块接口交互
//   - 可复用：模块可在不同场景（游戏中、选卡界面）使用
func (s *GameScene) initPlantCardSystems(rm *game.ResourceManager) {
	// Story 8.3: 获取当前关卡配置
	// 注意：CurrentLevel 从 GameState.LoadLevel() 加载
	levelConfig := s.gameState.CurrentLevel
	if levelConfig == nil {
		log.Printf("[GameScene] Warning: No level config found, using default plant cards")
		levelConfig = &config.LevelConfig{
			AvailablePlants: []string{"sunflower", "peashooter", "wallnut", "cherrybomb"},
		}
	}

	// 创建植物选择栏模块
	var err error
	s.plantSelectionModule, err = modules.NewPlantSelectionModule(
		s.entityManager,
		s.gameState,
		rm,
		s.reanimSystem,
		levelConfig,
		s.plantCardFont,
		config.SeedBankX,
		config.SeedBankY,
	)
	if err != nil {
		log.Printf("[GameScene] Error: Failed to initialize plant selection module: %v", err)
		// 游戏可在没有卡片的情况下运行（用于测试环境）
		return
	}

	log.Printf("[GameScene] Plant selection module initialized successfully")
}

// initMenuButton 初始化菜单按钮（ECS 架构）
// 创建可复用的三段式按钮实体，文字自动居中
func (s *GameScene) initMenuButton(rm *game.ResourceManager) {
	// 计算菜单按钮位置（右上角）
	buttonX := float64(WindowWidth) - config.MenuButtonOffsetFromRight
	buttonY := config.MenuButtonOffsetFromTop

	// 按钮中间部分宽度（根据文字长度）
	middleWidth := config.MenuButtonTextWidth + config.MenuButtonTextPadding*2

	// 创建菜单按钮实体
	var err error
	s.menuButtonEntity, err = entities.NewMenuButton(
		s.entityManager,
		rm,
		buttonX,
		buttonY,
		"菜单",                     // 按钮文字
		20.0,                     // 字体大小
		[4]uint8{0, 200, 0, 255}, // 绿色文字
		middleWidth,              // 中间宽度
		func() {
			// Story 10.1: 点击菜单按钮打开暂停菜单
			log.Printf("[GameScene] Menu button clicked! Opening pause menu...")
			if s.pauseMenuModule != nil {
				s.pauseMenuModule.Show()
			}
		},
	)

	if err != nil {
		log.Printf("[GameScene] Warning: Failed to create menu button: %v", err)
	} else {
		log.Printf("[GameScene] Menu button created successfully (Entity ID: %d)", s.menuButtonEntity)
	}
}

// initPauseMenuModule 初始化暂停菜单（ECS 架构）
// Story 10.1: 创建暂停菜单实体和三个按钮
// Story 18.2: 添加战斗存档保存回调
func (s *GameScene) initPauseMenuModule(rm *game.ResourceManager) {
	var err error
	s.pauseMenuModule, err = modules.NewPauseMenuModule(
		s.entityManager,
		s.gameState,
		rm,
		s.buttonSystem,
		s.buttonRenderSystem,
		WindowWidth,
		WindowHeight,
		modules.PauseMenuCallbacks{
			OnContinue: func() {
				s.gameState.SetPaused(false) // 恢复游戏
			},
			OnRestart: func() {
				// Story 18.2: 重新开始时删除战斗存档
				s.deleteBattleSave()
				// 重新加载当前关卡（使用当前关卡ID）
				currentLevelID := "1-1" // 默认值
				if s.gameState.CurrentLevel != nil {
					currentLevelID = s.gameState.CurrentLevel.ID
				}
				s.sceneManager.SwitchTo(NewGameScene(s.resourceManager, s.sceneManager, currentLevelID))
			},
			OnMainMenu: func() {
				// 返回主菜单
				s.sceneManager.SwitchTo(NewMainMenuScene(s.resourceManager, s.sceneManager))
			},
			OnPauseMusic: func() {
				// TODO: 暂停 BGM（当BGM系统实现后）
				// Story 17.6: 暂停波次计时
				if s.levelSystem != nil {
					s.levelSystem.PauseWaveTiming()
				}
			},
			OnResumeMusic: func() {
				// TODO: 恢复 BGM（当BGM系统实现后）
				// Story 17.6: 恢复波次计时
				if s.levelSystem != nil {
					s.levelSystem.ResumeWaveTiming()
				}
			},
			// Story 18.2: 保存战斗状态回调
			OnSaveBattle: func() {
				s.saveBattleState()
			},
		},
	)
	if err != nil {
		log.Printf("[GameScene] Warning: Failed to initialize pause menu module: %v", err)
	} else {
		log.Printf("[GameScene] Pause menu module initialized")
	}
}

// initProgressBar 初始化关卡进度条（Story 11.2）
// 创建进度条实体和渲染系统，关联到 LevelSystem
func (s *GameScene) initProgressBar(rm *game.ResourceManager) {
	// 加载字体（用于关卡文本）
	font, err := rm.LoadFont("assets/fonts/SimHei.ttf", config.LevelTextFontSize)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to load progress bar font: %v", err)
		return
	}
	log.Printf("[GameScene] Loaded progress bar font: SimHei.ttf (%.0fpx)", config.LevelTextFontSize)

	// 创建进度条渲染系统
	s.levelProgressBarRenderSystem = systems.NewLevelProgressBarRenderSystem(s.entityManager, font)
	log.Printf("[GameScene] Created progress bar render system")

	// 创建进度条实体（位置会在渲染时根据右对齐动态计算）
	progressBarEntity, err := entities.NewLevelProgressBarEntity(
		s.entityManager,
		rm,
	)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to create progress bar entity: %v", err)
		return
	}

	s.levelProgressBarEntity = progressBarEntity

	// 关联到 LevelSystem（让 LevelSystem 初始化进度条数据）
	s.levelSystem.SetProgressBarEntity(progressBarEntity)

	log.Printf("[GameScene] Level progress bar initialized (Entity ID: %d)", progressBarEntity)
}

// initLawnmowers 初始化除草车实体
// Story 10.2: 在每个启用的行上创建一台除草车
//
// 除草车是每行的最后防线：
// - 僵尸到达左侧边界时自动触发
// - 沿该行向右快速移动，消灭路径上的所有僵尸
// - 每行只有一台除草车，使用后不可恢复
func (s *GameScene) initLawnmowers() {
	if s.gameState.CurrentLevel == nil {
		log.Printf("[GameScene] Warning: No current level, skipping lawnmower initialization")
		return
	}

	// 获取关卡启用的行
	enabledLanes := s.gameState.CurrentLevel.EnabledLanes
	if len(enabledLanes) == 0 {
		// 如果未配置EnabledLanes，默认启用所有5行
		enabledLanes = []int{1, 2, 3, 4, 5}
	}

	// 为每个启用的行创建除草车
	for _, lane := range enabledLanes {
		lawnmowerID, err := entities.NewLawnmowerEntity(
			s.entityManager,
			s.resourceManager,
			lane,
		)

		if err != nil {
			log.Printf("[GameScene] ERROR: Failed to create lawnmower for lane %d: %v", lane, err)
			continue
		}

		log.Printf("[GameScene] Created lawnmower for lane %d (Entity ID: %d)", lane, lawnmowerID)
	}

	log.Printf("[GameScene] Initialized %d lawnmowers for enabled lanes: %v", len(enabledLanes), enabledLanes)
}

// saveBattleState 保存当前战斗状态
//
// Story 18.2: 战斗存档保存触发
//
// 调用时机：
//   - 玩家点击暂停菜单的"返回主菜单"按钮
//
// 保存内容：
//   - 关卡ID、时间、阳光
//   - 波次进度
//   - 所有实体状态（植物、僵尸、子弹、阳光、除草车）
func (s *GameScene) saveBattleState() {
	// 获取当前用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot save battle state")
		return
	}

	// 获取存档文件路径
	battleSavePath := saveManager.GetBattleSavePath(currentUser)

	// 创建序列化器并保存
	serializer := game.NewBattleSerializer()
	if err := serializer.SaveBattle(s.entityManager, s.gameState, battleSavePath); err != nil {
		log.Printf("[GameScene] ERROR: Failed to save battle state: %v", err)
		return
	}

	log.Printf("[GameScene] Battle state saved successfully to %s", battleSavePath)
}

// deleteBattleSave 删除当前用户的战斗存档
//
// Story 18.2: 重新开始时删除存档
//
// 调用时机：
//   - 玩家点击暂停菜单的"重新开始"按钮
//   - 游戏胜利后（进入下一关）
//   - 从存档恢复后
func (s *GameScene) deleteBattleSave() {
	// 获取当前用户
	saveManager := s.gameState.GetSaveManager()
	currentUser := saveManager.GetCurrentUser()
	if currentUser == "" {
		log.Printf("[GameScene] Warning: No current user, cannot delete battle save")
		return
	}

	// 删除存档
	if err := saveManager.DeleteBattleSave(currentUser); err != nil {
		log.Printf("[GameScene] ERROR: Failed to delete battle save: %v", err)
		return
	}

	log.Printf("[GameScene] Battle save deleted for user: %s", currentUser)
}

// restoreBattleState 从战斗存档恢复战斗状态
//
// Story 18.3: 继续游戏对话框与场景恢复
//
// 恢复流程：
//  1. 获取当前用户的存档路径
//  2. 使用 BattleSerializer.LoadBattle() 加载存档数据
//  3. 恢复游戏状态（阳光、波次进度）
//  4. 恢复所有实体（植物、僵尸、子弹、阳光、除草车）
//  5. 恢复草坪网格占用状态
//  6. 成功后删除存档（避免重复加载）
//  7. 失败时记录日志，继续正常游戏
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

	// 获取存档路径
	battleSavePath := saveManager.GetBattleSavePath(currentUser)
	log.Printf("[GameScene] 开始从战斗存档恢复: %s", battleSavePath)

	// 创建序列化器并加载
	serializer := game.NewBattleSerializer()
	saveData, err := serializer.LoadBattle(battleSavePath)
	if err != nil {
		log.Printf("[GameScene] ERROR: Failed to load battle data: %v", err)
		log.Printf("[GameScene] 继续正常游戏...")
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

	log.Printf("[GameScene] 游戏状态已恢复: Sun=%d, Wave=%d, Time=%.1f",
		s.gameState.Sun, s.gameState.CurrentWaveIndex, s.gameState.LevelTime)

	// Story 18.3: 恢复所有实体
	s.restorePlants(saveData.Plants)
	s.restoreZombies(saveData.Zombies)
	s.restoreProjectiles(saveData.Projectiles)
	s.restoreSuns(saveData.Suns)
	s.restoreLawnmowers(saveData.Lawnmowers)

	log.Printf("[GameScene] 实体恢复完成: Plants=%d, Zombies=%d, Projectiles=%d, Suns=%d, Lawnmowers=%d",
		len(saveData.Plants), len(saveData.Zombies), len(saveData.Projectiles),
		len(saveData.Suns), len(saveData.Lawnmowers))

	// 跳过开场动画
	s.isIntroAnimPlaying = false
	s.cameraX = config.GameCameraX
	if s.cameraSystem != nil {
		s.gameState.CameraX = config.GameCameraX
	}

	// 跳过铺草皮动画
	s.soddingAnimStarted = true

	// 恢复后删除存档
	if err := saveManager.DeleteBattleSave(currentUser); err != nil {
		log.Printf("[GameScene] Warning: Failed to delete battle save after restore: %v", err)
	} else {
		log.Printf("[GameScene] 存档已删除（恢复成功后）")
	}

	log.Printf("[GameScene] 战斗状态恢复完成!")
}

// restorePlants 恢复植物实体
//
// Story 18.3: 从存档数据重建植物实体
//
// 恢复内容：
//   - 植物类型和位置（网格行列）
//   - 生命值（当前/最大）
//   - 攻击冷却时间
//   - 草坪网格占用状态
//
// 简化处理：
//   - 动画从 idle 状态开始
//   - 眨眼计时器重置
func (s *GameScene) restorePlants(plants []game.PlantData) {
	for _, plantData := range plants {
		// 将植物类型字符串转换为 PlantType
		plantType := stringToPlantType(plantData.PlantType)
		if plantType == components.PlantUnknown {
			log.Printf("[GameScene] Warning: Unknown plant type '%s', skipping", plantData.PlantType)
			continue
		}

		// 根据植物类型创建实体
		var entityID ecs.EntityID
		var err error

		switch plantType {
		case components.PlantSunflower, components.PlantPeashooter:
			entityID, err = entities.NewPlantEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				plantType,
				plantData.GridCol,
				plantData.GridRow,
			)
		case components.PlantWallnut:
			entityID, err = entities.NewWallnutEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				s.reanimSystem,
				plantData.GridCol,
				plantData.GridRow,
			)
		case components.PlantCherryBomb:
			entityID, err = entities.NewCherryBombEntity(
				s.entityManager,
				s.resourceManager,
				s.gameState,
				plantData.GridCol,
				plantData.GridRow,
			)
		default:
			log.Printf("[GameScene] Warning: Unsupported plant type '%s', skipping", plantData.PlantType)
			continue
		}

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

		// 恢复攻击冷却（通过 TimerComponent）
		if plantData.AttackCooldown > 0 {
			if timerComp, ok := ecs.GetComponent[*components.TimerComponent](s.entityManager, entityID); ok {
				timerComp.CurrentTime = timerComp.TargetTime - plantData.AttackCooldown
				if timerComp.CurrentTime < 0 {
					timerComp.CurrentTime = 0
				}
			}
		}

		// 更新草坪网格占用状态
		if s.lawnGridSystem != nil && s.lawnGridEntityID != 0 {
			if err := s.lawnGridSystem.OccupyCell(s.lawnGridEntityID, plantData.GridCol, plantData.GridRow, entityID); err != nil {
				log.Printf("[GameScene] Warning: Failed to occupy grid cell (%d,%d): %v",
					plantData.GridCol, plantData.GridRow, err)
			}
		}

		log.Printf("[GameScene] Restored plant %s at (%d,%d), health=%d/%d",
			plantData.PlantType, plantData.GridRow, plantData.GridCol, plantData.Health, plantData.MaxHealth)
	}
}

// restoreZombies 恢复僵尸实体
//
// Story 18.3: 从存档数据重建僵尸实体
//
// 恢复内容：
//   - 僵尸类型和位置（X, Y）
//   - 生命值和护甲值
//   - 速度
//   - 行号
//
// 简化处理：
//   - 行为状态简化为 walking（让系统重新判断）
//   - 动画从 walk 状态开始
func (s *GameScene) restoreZombies(zombies []game.ZombieData) {
	for _, zombieData := range zombies {
		// 跳过正在死亡的僵尸
		if zombieData.BehaviorType == "dying" || zombieData.BehaviorType == "dying_explosion" {
			log.Printf("[GameScene] Skipping dying zombie at (%.1f, %.1f)", zombieData.X, zombieData.Y)
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

		// 根据僵尸类型创建实体
		var entityID ecs.EntityID
		var err error

		switch zombieData.ZombieType {
		case "basic":
			entityID, err = entities.NewZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		case "conehead":
			entityID, err = entities.NewConeheadZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		case "buckethead":
			entityID, err = entities.NewBucketheadZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		default:
			// 默认创建普通僵尸
			entityID, err = entities.NewZombieEntity(s.entityManager, s.resourceManager, lane-1, zombieData.X)
		}

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

		// 恢复速度并激活僵尸
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
			if zombieData.VelocityX != 0 {
				velComp.VX = zombieData.VelocityX
			} else {
				// 如果没有保存速度，使用默认速度激活（僵尸标准移动速度）
				velComp.VX = -23.0
			}
		}

		// 设置行为状态为 walking（让 BehaviorSystem 重新判断是否需要切换到 eating）
		if behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](s.entityManager, entityID); ok {
			if zombieData.IsEating {
				behaviorComp.ZombieAnimState = components.ZombieAnimEating
			} else {
				behaviorComp.ZombieAnimState = components.ZombieAnimWalking
			}
		}

		// 添加目标行组件
		ecs.AddComponent(s.entityManager, entityID, &components.ZombieTargetLaneComponent{
			TargetRow: lane - 1,
		})

		log.Printf("[GameScene] Restored zombie %s at (%.1f, %.1f), lane=%d, health=%d/%d",
			zombieData.ZombieType, zombieData.X, zombieData.Y, lane, zombieData.Health, zombieData.MaxHealth)
	}
}

// restoreProjectiles 恢复子弹实体
//
// Story 18.3: 从存档数据重建子弹实体
//
// 恢复内容：
//   - 子弹类型和位置
//   - 速度
//   - 伤害值
//
// 简化处理：
//   - 只恢复豌豆子弹
func (s *GameScene) restoreProjectiles(projectiles []game.ProjectileData) {
	for _, projData := range projectiles {
		// 目前只支持豌豆子弹
		if projData.Type != "pea" {
			log.Printf("[GameScene] Warning: Unsupported projectile type '%s', skipping", projData.Type)
			continue
		}

		// 计算行号（从 Y 坐标推算）
		row := int((projData.Y - config.GridWorldStartY) / config.CellHeight)
		if row < 0 {
			row = 0
		}
		if row > 4 {
			row = 4
		}

		// 创建子弹实体
		entityID := s.entityManager.CreateEntity()

		// 添加位置组件
		s.entityManager.AddComponent(entityID, &components.PositionComponent{
			X: projData.X,
			Y: projData.Y,
		})

		// 添加速度组件
		velX := projData.VelocityX
		if velX == 0 {
			velX = config.PeaBulletSpeed // 默认速度
		}
		s.entityManager.AddComponent(entityID, &components.VelocityComponent{
			VX: velX,
			VY: 0,
		})

		// 添加行为组件
		s.entityManager.AddComponent(entityID, &components.BehaviorComponent{
			Type: components.BehaviorPeaProjectile,
		})

		// 添加碰撞组件
		s.entityManager.AddComponent(entityID, &components.CollisionComponent{
			Width:  config.PeaBulletWidth,
			Height: config.PeaBulletHeight,
		})

		// 加载豌豆子弹图片
		reanimXML := s.resourceManager.GetReanimXML("ProjectilePea")
		partImages := s.resourceManager.GetReanimPartImages("ProjectilePea")
		if reanimXML != nil && partImages != nil {
			s.entityManager.AddComponent(entityID, &components.ReanimComponent{
				ReanimName: "ProjectilePea",
				ReanimXML:  reanimXML,
				PartImages: partImages,
			})
			// 添加动画命令
			ecs.AddComponent(s.entityManager, entityID, &components.AnimationCommandComponent{
				AnimationName: "anim_idle",
				Processed:     false,
			})
		}

		log.Printf("[GameScene] Restored projectile at (%.1f, %.1f), row=%d", projData.X, projData.Y, row)
	}
}

// restoreSuns 恢复阳光实体
//
// Story 18.3: 从存档数据重建阳光实体
//
// 恢复内容：
//   - 位置
//   - 剩余生命周期
//   - 收集状态
//
// 简化处理：
//   - 收集动画状态重置（正在收集的阳光按已着陆处理）
func (s *GameScene) restoreSuns(suns []game.SunData) {
	for _, sunData := range suns {
		// 跳过正在收集的阳光（简化处理）
		if sunData.IsCollecting {
			log.Printf("[GameScene] Skipping collecting sun at (%.1f, %.1f)", sunData.X, sunData.Y)
			continue
		}

		// 创建静态阳光实体（已着陆状态）
		entityID := entities.NewSunEntityStatic(s.entityManager, s.resourceManager, sunData.X, sunData.Y)

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

		// 确保阳光处于着陆状态
		if sunComp, ok := ecs.GetComponent[*components.SunComponent](s.entityManager, entityID); ok {
			sunComp.State = components.SunLanded
		}

		// 停止下落
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](s.entityManager, entityID); ok {
			velComp.VY = 0
		}

		log.Printf("[GameScene] Restored sun at (%.1f, %.1f), lifetime=%.1f", sunData.X, sunData.Y, sunData.Lifetime)
	}
}

// restoreLawnmowers 恢复除草车实体
//
// Story 18.3: 从存档数据重建除草车实体
//
// 恢复内容：
//   - 行号
//   - 位置
//   - 触发状态
//   - 激活状态
//
// 注意：
//   - 已触发且移出屏幕的除草车不恢复
//   - 正在移动的除草车恢复位置和速度
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

		// 恢复状态
		if lmComp, ok := ecs.GetComponent[*components.LawnmowerComponent](s.entityManager, entityID); ok {
			lmComp.IsTriggered = lmData.Triggered
			lmComp.IsMoving = lmData.Active

			// 如果已触发，跳过入场动画
			if lmData.Triggered || lmData.X >= config.LawnmowerStartX {
				lmComp.IsEntering = false
			}
		}

		// 如果正在移动，设置速度
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

// stringToPlantType 将植物类型字符串转换为 PlantType
func stringToPlantType(s string) components.PlantType {
	switch s {
	case "Sunflower", "sunflower":
		return components.PlantSunflower
	case "Peashooter", "peashooter":
		return components.PlantPeashooter
	case "Wallnut", "wallnut":
		return components.PlantWallnut
	case "CherryBomb", "cherrybomb":
		return components.PlantCherryBomb
	default:
		return components.PlantUnknown
	}
}
