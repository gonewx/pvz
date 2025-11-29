package game

import (
	"encoding/gob"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
)

// BattleSerializer 战斗状态序列化器
//
// Story 18.1: 战斗状态序列化系统
//
// 负责将游戏战斗状态序列化为二进制文件，以及从文件反序列化恢复游戏状态。
// 使用 gob 二进制格式，具有紧凑、类型安全、防作弊等优点。
//
// 架构说明：
//   - 这是一个工具类，不是 ECS 系统
//   - 可以访问 EntityManager 收集实体数据
//   - 不直接修改游戏状态，仅负责序列化/反序列化
type BattleSerializer struct{}

// NewBattleSerializer 创建战斗序列化器实例
func NewBattleSerializer() *BattleSerializer {
	return &BattleSerializer{}
}

// SaveBattle 保存战斗状态到文件
//
// 从 EntityManager 收集所有实体数据，从 GameState 收集关卡状态，
// 序列化为 gob 二进制格式并写入文件。
//
// 参数：
//   - em: EntityManager 实例，用于收集实体数据
//   - gs: GameState 实例，用于收集关卡状态
//   - filePath: 保存文件路径
//
// 返回：
//   - error: 如果保存失败返回错误
func (s *BattleSerializer) SaveBattle(em *ecs.EntityManager, gs *GameState, filePath string) error {
	if em == nil {
		return fmt.Errorf("EntityManager is nil")
	}
	if gs == nil {
		return fmt.Errorf("GameState is nil")
	}

	// 创建存档数据
	saveData := NewBattleSaveData()
	saveData.SaveTime = time.Now()

	// 收集关卡状态
	s.collectLevelState(gs, saveData)

	// 收集实体数据
	saveData.Plants = s.collectPlantData(em)
	saveData.Zombies = s.collectZombieData(em)
	saveData.Projectiles = s.collectProjectileData(em)
	saveData.Suns = s.collectSunData(em)
	saveData.Lawnmowers = s.collectLawnmowerData(em)

	// 创建文件
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create save file: %w", err)
	}
	defer file.Close()

	// 使用 gob 编码
	encoder := gob.NewEncoder(file)
	if err := encoder.Encode(saveData); err != nil {
		return fmt.Errorf("failed to encode save data: %w", err)
	}

	log.Printf("[BattleSerializer] Saved battle to %s: Level=%s, Sun=%d, Wave=%d/%d, Plants=%d, Zombies=%d",
		filePath, saveData.LevelID, saveData.Sun, saveData.CurrentWaveIndex,
		len(saveData.SpawnedWaves), len(saveData.Plants), len(saveData.Zombies))

	return nil
}

// LoadBattle 从文件加载战斗状态
//
// 从 gob 二进制文件反序列化战斗数据。
// 会进行版本兼容性检查，如果版本不匹配返回错误。
//
// 参数：
//   - filePath: 存档文件路径
//
// 返回：
//   - *BattleSaveData: 战斗存档数据
//   - error: 如果加载失败返回错误
func (s *BattleSerializer) LoadBattle(filePath string) (*BattleSaveData, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open save file: %w", err)
	}
	defer file.Close()

	// 使用 gob 解码
	var saveData BattleSaveData
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&saveData); err != nil {
		return nil, fmt.Errorf("failed to decode save data: %w", err)
	}

	// 版本兼容性检查
	if saveData.Version != BattleSaveVersion {
		return nil, fmt.Errorf("incompatible save version: %d (expected %d)",
			saveData.Version, BattleSaveVersion)
	}

	log.Printf("[BattleSerializer] Loaded battle from %s: Level=%s, Sun=%d, Wave=%d/%d, Plants=%d, Zombies=%d",
		filePath, saveData.LevelID, saveData.Sun, saveData.CurrentWaveIndex,
		len(saveData.SpawnedWaves), len(saveData.Plants), len(saveData.Zombies))

	return &saveData, nil
}

// collectLevelState 从 GameState 收集关卡状态
func (s *BattleSerializer) collectLevelState(gs *GameState, saveData *BattleSaveData) {
	// 关卡基本信息
	if gs.CurrentLevel != nil {
		saveData.LevelID = gs.CurrentLevel.ID
	}
	saveData.LevelTime = gs.LevelTime
	saveData.CurrentWaveIndex = gs.CurrentWaveIndex
	saveData.Sun = gs.Sun

	// 波次状态
	saveData.SpawnedWaves = make([]bool, len(gs.SpawnedWaves))
	copy(saveData.SpawnedWaves, gs.SpawnedWaves)

	// 僵尸计数
	saveData.TotalZombiesSpawned = gs.TotalZombiesSpawned
	saveData.ZombiesKilled = gs.ZombiesKilled
}

// collectPlantData 从 EntityManager 收集所有植物实体数据
func (s *BattleSerializer) collectPlantData(em *ecs.EntityManager) []PlantData {
	var plants []PlantData

	// 查询所有拥有 PlantComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.PlantComponent,
		*components.PositionComponent,
	](em)

	for _, entity := range entities {
		plantComp, ok := ecs.GetComponent[*components.PlantComponent](em, entity)
		if !ok {
			continue
		}

		// 获取生命值组件
		var health, maxHealth int
		if healthComp, ok := ecs.GetComponent[*components.HealthComponent](em, entity); ok {
			health = healthComp.CurrentHealth
			maxHealth = healthComp.MaxHealth
		}

		// 获取计时器组件（攻击冷却）
		var attackCooldown float64
		if timerComp, ok := ecs.GetComponent[*components.TimerComponent](em, entity); ok {
			// 剩余冷却时间 = 目标时间 - 当前时间
			attackCooldown = timerComp.TargetTime - timerComp.CurrentTime
			if attackCooldown < 0 {
				attackCooldown = 0
			}
		}

		plants = append(plants, PlantData{
			PlantType:       plantComp.PlantType.String(),
			GridRow:         plantComp.GridRow,
			GridCol:         plantComp.GridCol,
			Health:          health,
			MaxHealth:       maxHealth,
			AttackCooldown:  attackCooldown,
			BlinkTimer:      plantComp.BlinkTimer,
			AttackAnimState: int(plantComp.AttackAnimState),
		})
	}

	return plants
}

// collectZombieData 从 EntityManager 收集所有僵尸实体数据
func (s *BattleSerializer) collectZombieData(em *ecs.EntityManager) []ZombieData {
	var zombies []ZombieData

	// 查询所有拥有 BehaviorComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](em)

	for _, entity := range entities {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](em, entity)
		if !ok {
			continue
		}

		// 判断是否是僵尸（检查行为类型）
		if !isZombieBehavior(behaviorComp.Type) {
			continue
		}

		posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entity)
		if !ok {
			continue
		}

		// 获取速度组件
		var velocityX float64
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, entity); ok {
			velocityX = velComp.VX
		}

		// 获取生命值组件
		var health, maxHealth int
		if healthComp, ok := ecs.GetComponent[*components.HealthComponent](em, entity); ok {
			health = healthComp.CurrentHealth
			maxHealth = healthComp.MaxHealth
		}

		// 获取护甲组件
		var armorHealth, armorMax int
		if armorComp, ok := ecs.GetComponent[*components.ArmorComponent](em, entity); ok {
			armorHealth = armorComp.CurrentArmor
			armorMax = armorComp.MaxArmor
		}

		// 获取行号
		var lane int
		if collComp, ok := ecs.GetComponent[*components.CollisionComponent](em, entity); ok {
			// 从碰撞组件或位置推算行号
			_ = collComp
		}
		// 尝试从 ZombieTargetLaneComponent 获取行号
		if laneComp, ok := ecs.GetComponent[*components.ZombieTargetLaneComponent](em, entity); ok {
			lane = laneComp.TargetRow + 1 // TargetRow 是 0-based，转换为 1-based
		}

		zombies = append(zombies, ZombieData{
			ZombieType:   behaviorTypeToZombieType(behaviorComp.Type),
			X:            posComp.X,
			Y:            posComp.Y,
			VelocityX:    velocityX,
			Health:       health,
			MaxHealth:    maxHealth,
			ArmorHealth:  armorHealth,
			ArmorMax:     armorMax,
			Lane:         lane,
			BehaviorType: behaviorTypeToString(behaviorComp.Type),
			IsEating:     behaviorComp.Type == components.BehaviorZombieEating,
		})
	}

	return zombies
}

// collectProjectileData 从 EntityManager 收集所有子弹实体数据
func (s *BattleSerializer) collectProjectileData(em *ecs.EntityManager) []ProjectileData {
	var projectiles []ProjectileData

	// 查询所有拥有 BehaviorComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.BehaviorComponent,
		*components.PositionComponent,
	](em)

	for _, entity := range entities {
		behaviorComp, ok := ecs.GetComponent[*components.BehaviorComponent](em, entity)
		if !ok {
			continue
		}

		// 判断是否是子弹
		if behaviorComp.Type != components.BehaviorPeaProjectile {
			continue
		}

		posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entity)
		if !ok {
			continue
		}

		// 获取速度组件
		var velocityX float64
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, entity); ok {
			velocityX = velComp.VX
		}

		// 获取碰撞组件（用于获取行号）
		var lane int
		if collComp, ok := ecs.GetComponent[*components.CollisionComponent](em, entity); ok {
			_ = collComp
			// 从位置推算行号（后续可以优化）
		}

		projectiles = append(projectiles, ProjectileData{
			Type:      "pea", // 目前只有豌豆子弹
			X:         posComp.X,
			Y:         posComp.Y,
			VelocityX: velocityX,
			Damage:    20, // 默认伤害值
			Lane:      lane,
		})
	}

	return projectiles
}

// collectSunData 从 EntityManager 收集所有阳光实体数据
func (s *BattleSerializer) collectSunData(em *ecs.EntityManager) []SunData {
	var suns []SunData

	// 查询所有拥有 SunComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.SunComponent,
		*components.PositionComponent,
	](em)

	for _, entity := range entities {
		sunComp, ok := ecs.GetComponent[*components.SunComponent](em, entity)
		if !ok {
			continue
		}

		posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entity)
		if !ok {
			continue
		}

		// 获取速度组件
		var velocityY float64
		if velComp, ok := ecs.GetComponent[*components.VelocityComponent](em, entity); ok {
			velocityY = velComp.VY
		}

		// 获取生命周期组件
		var lifetime float64
		if lifetimeComp, ok := ecs.GetComponent[*components.LifetimeComponent](em, entity); ok {
			lifetime = lifetimeComp.MaxLifetime - lifetimeComp.CurrentLifetime
			if lifetime < 0 {
				lifetime = 0
			}
		}

		// 获取收集动画组件
		var isCollecting bool
		var targetX, targetY float64
		if collectComp, ok := ecs.GetComponent[*components.SunCollectionAnimationComponent](em, entity); ok {
			isCollecting = true
			targetX = collectComp.TargetX
			targetY = collectComp.TargetY
		}

		suns = append(suns, SunData{
			X:            posComp.X,
			Y:            posComp.Y,
			VelocityY:    velocityY,
			Lifetime:     lifetime,
			Value:        25, // 默认阳光值
			IsCollecting: isCollecting,
			TargetX:      targetX,
			TargetY:      targetY,
		})

		_ = sunComp // 使用 sunComp 避免编译警告
	}

	return suns
}

// collectLawnmowerData 从 EntityManager 收集所有除草车实体数据
func (s *BattleSerializer) collectLawnmowerData(em *ecs.EntityManager) []LawnmowerData {
	var lawnmowers []LawnmowerData

	// 查询所有拥有 LawnmowerComponent 和 PositionComponent 的实体
	entities := ecs.GetEntitiesWith2[
		*components.LawnmowerComponent,
		*components.PositionComponent,
	](em)

	for _, entity := range entities {
		lawnmowerComp, ok := ecs.GetComponent[*components.LawnmowerComponent](em, entity)
		if !ok {
			continue
		}

		posComp, ok := ecs.GetComponent[*components.PositionComponent](em, entity)
		if !ok {
			continue
		}

		lawnmowers = append(lawnmowers, LawnmowerData{
			Lane:      lawnmowerComp.Lane,
			X:         posComp.X,
			Triggered: lawnmowerComp.IsTriggered,
			Active:    lawnmowerComp.IsMoving,
		})
	}

	return lawnmowers
}

// isZombieBehavior 判断行为类型是否是僵尸行为
func isZombieBehavior(behaviorType components.BehaviorType) bool {
	switch behaviorType {
	case components.BehaviorZombieBasic,
		components.BehaviorZombieEating,
		components.BehaviorZombieDying,
		components.BehaviorZombieSquashing,
		components.BehaviorZombieDyingExplosion,
		components.BehaviorZombieConehead,
		components.BehaviorZombieBuckethead,
		components.BehaviorZombiePreview:
		return true
	default:
		return false
	}
}

// behaviorTypeToZombieType 将行为类型转换为僵尸类型字符串
func behaviorTypeToZombieType(behaviorType components.BehaviorType) string {
	switch behaviorType {
	case components.BehaviorZombieBasic, components.BehaviorZombieEating, components.BehaviorZombieDying:
		return "basic"
	case components.BehaviorZombieConehead:
		return "conehead"
	case components.BehaviorZombieBuckethead:
		return "buckethead"
	default:
		return "basic"
	}
}

// behaviorTypeToString 将行为类型转换为字符串
func behaviorTypeToString(behaviorType components.BehaviorType) string {
	switch behaviorType {
	case components.BehaviorZombieBasic:
		return "basic"
	case components.BehaviorZombieEating:
		return "eating"
	case components.BehaviorZombieDying:
		return "dying"
	case components.BehaviorZombieSquashing:
		return "squashing"
	case components.BehaviorZombieDyingExplosion:
		return "dying_explosion"
	case components.BehaviorZombieConehead:
		return "conehead"
	case components.BehaviorZombieBuckethead:
		return "buckethead"
	case components.BehaviorZombiePreview:
		return "preview"
	default:
		return "unknown"
	}
}
