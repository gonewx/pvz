package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/quasilyte/gdata/v2"
)

// createTestGdataManagerForBattle 创建用于测试的 gdata Manager
func createTestGdataManagerForBattle(t *testing.T, testName string) *gdata.Manager {
	appName := fmt.Sprintf("pvz_battle_test_%s_%d", testName, time.Now().UnixNano())
	manager, err := gdata.Open(gdata.Config{
		AppName: appName,
	})
	if err != nil {
		return nil
	}

	// 注册清理函数，测试结束后删除测试目录
	t.Cleanup(func() {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			testDir := filepath.Join(homeDir, ".local", "share", appName)
			os.RemoveAll(testDir)
		}
	})

	return manager
}

// TestBattleSerializer_NewBattleSerializer 测试创建序列化器
func TestBattleSerializer_NewBattleSerializer(t *testing.T) {
	serializer := NewBattleSerializer(nil)
	if serializer == nil {
		t.Fatal("NewBattleSerializer returned nil")
	}
}

// TestBattleSerializer_SaveBattle_NilEntityManager 测试空 EntityManager
func TestBattleSerializer_SaveBattle_NilEntityManager(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "nil_em")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	serializer := NewBattleSerializer(gdataManager)
	gs := &GameState{Sun: 100}

	err := serializer.SaveBattle(nil, gs, "testuser")
	if err == nil {
		t.Error("Expected error when EntityManager is nil")
	}
}

// TestBattleSerializer_SaveBattle_NilGameState 测试空 GameState
func TestBattleSerializer_SaveBattle_NilGameState(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "nil_gs")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	serializer := NewBattleSerializer(gdataManager)
	em := ecs.NewEntityManager()

	err := serializer.SaveBattle(em, nil, "testuser")
	if err == nil {
		t.Error("Expected error when GameState is nil")
	}
}

// TestBattleSerializer_SaveBattle_NilGdataManager 测试空 gdata Manager
func TestBattleSerializer_SaveBattle_NilGdataManager(t *testing.T) {
	serializer := NewBattleSerializer(nil)
	em := ecs.NewEntityManager()
	gs := &GameState{Sun: 100}

	err := serializer.SaveBattle(em, gs, "testuser")
	if err == nil {
		t.Error("Expected error when gdata manager is nil")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_Empty 测试空战斗状态
func TestBattleSerializer_SaveAndLoadBattle_Empty(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "empty")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	// 创建空的 EntityManager 和基本 GameState
	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:              100,
		CurrentWaveIndex: 2,
		SpawnedWaves:     []bool{true, true, false},
		CurrentLevel:     &config.LevelConfig{ID: "1-3"},
	}

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 验证数据存在
	battleKey := "testuser" + BattleSaveKeySuffix
	if !gdataManager.ObjectPropExists(savesObject, battleKey) {
		t.Fatal("Save data was not created in gdata")
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证数据
	if data.Sun != 100 {
		t.Errorf("Expected Sun 100, got %d", data.Sun)
	}
	if data.CurrentWaveIndex != 2 {
		t.Errorf("Expected CurrentWaveIndex 2, got %d", data.CurrentWaveIndex)
	}
	if data.LevelID != "1-3" {
		t.Errorf("Expected LevelID '1-3', got %q", data.LevelID)
	}
	if len(data.SpawnedWaves) != 3 {
		t.Errorf("Expected 3 SpawnedWaves, got %d", len(data.SpawnedWaves))
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithPlants 测试带植物的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithPlants(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "with_plants")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          150,
		SpawnedWaves: []bool{true},
		CurrentLevel: &config.LevelConfig{ID: "1-2"},
	}

	// 创建植物实体
	plant1 := em.CreateEntity()
	ecs.AddComponent(em, plant1, &components.PlantComponent{
		PlantType: components.PlantPeashooter,
		GridRow:   1,
		GridCol:   2,
	})
	ecs.AddComponent(em, plant1, &components.PositionComponent{X: 200, Y: 100})
	ecs.AddComponent(em, plant1, &components.HealthComponent{CurrentHealth: 280, MaxHealth: 300})

	plant2 := em.CreateEntity()
	ecs.AddComponent(em, plant2, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   0,
		GridCol:   0,
	})
	ecs.AddComponent(em, plant2, &components.PositionComponent{X: 100, Y: 50})
	ecs.AddComponent(em, plant2, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证植物数据
	if len(data.Plants) != 2 {
		t.Fatalf("Expected 2 plants, got %d", len(data.Plants))
	}

	// 找到豌豆射手
	var foundPeashooter bool
	for _, p := range data.Plants {
		if p.PlantType == components.PlantPeashooter.String() {
			foundPeashooter = true
			if p.Health != 280 {
				t.Errorf("Peashooter health mismatch: expected 280, got %d", p.Health)
			}
			if p.GridRow != 1 || p.GridCol != 2 {
				t.Errorf("Peashooter grid position mismatch: expected (1, 2), got (%d, %d)", p.GridRow, p.GridCol)
			}
		}
	}
	if !foundPeashooter {
		t.Error("Peashooter not found in saved data")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithZombies 测试带僵尸的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithZombies(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "with_zombies")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          200,
		SpawnedWaves: []bool{true, true},
		CurrentLevel: &config.LevelConfig{ID: "1-4"},
	}

	// 创建僵尸实体
	zombie1 := em.CreateEntity()
	ecs.AddComponent(em, zombie1, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	ecs.AddComponent(em, zombie1, &components.PositionComponent{X: 500, Y: 150})
	ecs.AddComponent(em, zombie1, &components.VelocityComponent{VX: -23.0})
	ecs.AddComponent(em, zombie1, &components.HealthComponent{CurrentHealth: 180, MaxHealth: 200})
	ecs.AddComponent(em, zombie1, &components.ZombieTargetLaneComponent{TargetRow: 1})

	zombie2 := em.CreateEntity()
	ecs.AddComponent(em, zombie2, &components.BehaviorComponent{Type: components.BehaviorZombieConehead})
	ecs.AddComponent(em, zombie2, &components.PositionComponent{X: 600, Y: 250})
	ecs.AddComponent(em, zombie2, &components.VelocityComponent{VX: -23.0})
	ecs.AddComponent(em, zombie2, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})
	ecs.AddComponent(em, zombie2, &components.ArmorComponent{CurrentArmor: 300, MaxArmor: 370})
	ecs.AddComponent(em, zombie2, &components.ZombieTargetLaneComponent{TargetRow: 2})

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证僵尸数据
	if len(data.Zombies) != 2 {
		t.Fatalf("Expected 2 zombies, got %d", len(data.Zombies))
	}

	// 找到路障僵尸
	var foundConehead bool
	for _, z := range data.Zombies {
		if z.ZombieType == "conehead" {
			foundConehead = true
			if z.ArmorHealth != 300 {
				t.Errorf("Conehead armor health mismatch: expected 300, got %d", z.ArmorHealth)
			}
			if z.Lane != 3 { // TargetRow 2 + 1 = Lane 3
				t.Errorf("Conehead lane mismatch: expected 3, got %d", z.Lane)
			}
		}
	}
	if !foundConehead {
		t.Error("Conehead zombie not found in saved data")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithLawnmowers 测试带除草车的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithLawnmowers(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "with_lawnmowers")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          100,
		SpawnedWaves: []bool{true},
		CurrentLevel: &config.LevelConfig{ID: "1-1"},
	}

	// 创建除草车实体
	for lane := 1; lane <= 5; lane++ {
		lawnmower := em.CreateEntity()
		ecs.AddComponent(em, lawnmower, &components.LawnmowerComponent{
			Lane:        lane,
			IsTriggered: lane == 1, // 第一行已触发
			IsMoving:    lane == 1,
		})
		ecs.AddComponent(em, lawnmower, &components.PositionComponent{
			X: 100,
			Y: float64(lane * 100),
		})
	}

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证除草车数据
	if len(data.Lawnmowers) != 5 {
		t.Fatalf("Expected 5 lawnmowers, got %d", len(data.Lawnmowers))
	}

	// 找到第一行的除草车
	var foundTriggered bool
	for _, lm := range data.Lawnmowers {
		if lm.Lane == 1 {
			foundTriggered = true
			if !lm.Triggered {
				t.Error("Lane 1 lawnmower should be triggered")
			}
			if !lm.Active {
				t.Error("Lane 1 lawnmower should be active")
			}
		}
	}
	if !foundTriggered {
		t.Error("Lane 1 lawnmower not found")
	}
}

// TestBattleSerializer_LoadBattle_NotFound 测试加载不存在的存档
func TestBattleSerializer_LoadBattle_NotFound(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "not_found")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	serializer := NewBattleSerializer(gdataManager)
	_, err := serializer.LoadBattle("nonexistent")
	if err == nil {
		t.Error("Expected error when loading non-existent save")
	}
}

// TestBattleSerializer_LoadBattle_Corrupted 测试加载损坏的存档
func TestBattleSerializer_LoadBattle_Corrupted(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "corrupted")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	// 写入无效数据
	battleKey := "testuser" + BattleSaveKeySuffix
	if err := gdataManager.SaveObjectProp(savesObject, battleKey, []byte("invalid gob data")); err != nil {
		t.Fatalf("Failed to create corrupted save: %v", err)
	}

	serializer := NewBattleSerializer(gdataManager)
	_, err := serializer.LoadBattle("testuser")
	if err == nil {
		t.Error("Expected error when loading corrupted save")
	}
}

// TestBattleSerializer_LoadBattle_NilGdataManager 测试空 gdata Manager
func TestBattleSerializer_LoadBattle_NilGdataManager(t *testing.T) {
	serializer := NewBattleSerializer(nil)
	_, err := serializer.LoadBattle("testuser")
	if err == nil {
		t.Error("Expected error when gdata manager is nil")
	}
}

// TestBattleSerializer_CollectLevelState 测试关卡状态收集
func TestBattleSerializer_CollectLevelState(t *testing.T) {
	serializer := NewBattleSerializer(nil)
	saveData := NewBattleSaveData()

	gs := &GameState{
		Sun:                 250,
		LevelTime:           123.45,
		CurrentWaveIndex:    5,
		SpawnedWaves:        []bool{true, true, true, true, true, false, false},
		TotalZombiesSpawned: 20,
		ZombiesKilled:       15,
		CurrentLevel:        &config.LevelConfig{ID: "2-3"},
	}

	serializer.collectLevelState(gs, saveData)

	if saveData.LevelID != "2-3" {
		t.Errorf("LevelID mismatch: expected '2-3', got %q", saveData.LevelID)
	}
	if saveData.Sun != 250 {
		t.Errorf("Sun mismatch: expected 250, got %d", saveData.Sun)
	}
	if saveData.LevelTime != 123.45 {
		t.Errorf("LevelTime mismatch: expected 123.45, got %f", saveData.LevelTime)
	}
	if saveData.CurrentWaveIndex != 5 {
		t.Errorf("CurrentWaveIndex mismatch: expected 5, got %d", saveData.CurrentWaveIndex)
	}
	if saveData.TotalZombiesSpawned != 20 {
		t.Errorf("TotalZombiesSpawned mismatch: expected 20, got %d", saveData.TotalZombiesSpawned)
	}
	if saveData.ZombiesKilled != 15 {
		t.Errorf("ZombiesKilled mismatch: expected 15, got %d", saveData.ZombiesKilled)
	}
	if len(saveData.SpawnedWaves) != 7 {
		t.Errorf("SpawnedWaves length mismatch: expected 7, got %d", len(saveData.SpawnedWaves))
	}
}

// TestIsZombieBehavior 测试僵尸行为判断
func TestIsZombieBehavior(t *testing.T) {
	tests := []struct {
		behavior components.BehaviorType
		expected bool
	}{
		{components.BehaviorZombieBasic, true},
		{components.BehaviorZombieEating, true},
		{components.BehaviorZombieDying, true},
		{components.BehaviorZombieSquashing, true},
		{components.BehaviorZombieDyingExplosion, true},
		{components.BehaviorZombieConehead, true},
		{components.BehaviorZombieBuckethead, true},
		{components.BehaviorZombiePreview, true},
		{components.BehaviorPeashooter, false},
		{components.BehaviorSunflower, false},
		{components.BehaviorPeaProjectile, false},
		{components.BehaviorWallnut, false},
	}

	for _, tt := range tests {
		result := isZombieBehavior(tt.behavior)
		if result != tt.expected {
			t.Errorf("isZombieBehavior(%v) = %v, expected %v", tt.behavior, result, tt.expected)
		}
	}
}

// TestBehaviorTypeToZombieType 测试行为类型到僵尸类型转换
func TestBehaviorTypeToZombieType(t *testing.T) {
	tests := []struct {
		behavior components.BehaviorType
		expected string
	}{
		{components.BehaviorZombieBasic, "basic"},
		{components.BehaviorZombieEating, "basic"},
		{components.BehaviorZombieDying, "basic"},
		{components.BehaviorZombieConehead, "conehead"},
		{components.BehaviorZombieBuckethead, "buckethead"},
	}

	for _, tt := range tests {
		result := behaviorTypeToZombieType(tt.behavior)
		if result != tt.expected {
			t.Errorf("behaviorTypeToZombieType(%v) = %q, expected %q", tt.behavior, result, tt.expected)
		}
	}
}

// TestBehaviorTypeToString 测试行为类型到字符串转换
func TestBehaviorTypeToString(t *testing.T) {
	tests := []struct {
		behavior components.BehaviorType
		expected string
	}{
		{components.BehaviorZombieBasic, "basic"},
		{components.BehaviorZombieEating, "eating"},
		{components.BehaviorZombieDying, "dying"},
		{components.BehaviorZombieSquashing, "squashing"},
		{components.BehaviorZombieDyingExplosion, "dying_explosion"},
		{components.BehaviorZombieConehead, "conehead"},
		{components.BehaviorZombieBuckethead, "buckethead"},
		{components.BehaviorZombiePreview, "preview"},
		{components.BehaviorPeashooter, "unknown"}, // 非僵尸类型应返回 unknown
		{components.BehaviorSunflower, "unknown"},
		{components.BehaviorType(999), "unknown"}, // 未定义的类型
	}

	for _, tt := range tests {
		result := behaviorTypeToString(tt.behavior)
		if result != tt.expected {
			t.Errorf("behaviorTypeToString(%v) = %q, expected %q", tt.behavior, result, tt.expected)
		}
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithSuns 测试带阳光的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithSuns(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "with_suns")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          100,
		SpawnedWaves: []bool{true},
		CurrentLevel: &config.LevelConfig{ID: "1-1"},
	}

	// 创建阳光实体 - 下落中的阳光
	sun1 := em.CreateEntity()
	ecs.AddComponent(em, sun1, &components.SunComponent{State: components.SunFalling, TargetY: 300})
	ecs.AddComponent(em, sun1, &components.PositionComponent{X: 200, Y: 100})
	ecs.AddComponent(em, sun1, &components.VelocityComponent{VY: 50})
	ecs.AddComponent(em, sun1, &components.LifetimeComponent{MaxLifetime: 10, CurrentLifetime: 2})

	// 创建阳光实体 - 正在收集的阳光
	sun2 := em.CreateEntity()
	ecs.AddComponent(em, sun2, &components.SunComponent{State: components.SunCollecting, TargetY: 50})
	ecs.AddComponent(em, sun2, &components.PositionComponent{X: 300, Y: 200})
	ecs.AddComponent(em, sun2, &components.SunCollectionAnimationComponent{
		StartX:  300,
		StartY:  200,
		TargetX: 50,
		TargetY: 30,
	})

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证阳光数据
	if len(data.Suns) != 2 {
		t.Fatalf("Expected 2 suns, got %d", len(data.Suns))
	}

	// 找到正在收集的阳光
	var foundCollecting bool
	for _, s := range data.Suns {
		if s.IsCollecting {
			foundCollecting = true
			if s.TargetX != 50 || s.TargetY != 30 {
				t.Errorf("Collecting sun target mismatch: expected (50, 30), got (%f, %f)", s.TargetX, s.TargetY)
			}
		}
	}
	if !foundCollecting {
		t.Error("Collecting sun not found in saved data")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithProjectiles 测试带子弹的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithProjectiles(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "with_projectiles")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          150,
		SpawnedWaves: []bool{true},
		CurrentLevel: &config.LevelConfig{ID: "1-2"},
	}

	// 创建子弹实体
	proj1 := em.CreateEntity()
	ecs.AddComponent(em, proj1, &components.BehaviorComponent{Type: components.BehaviorPeaProjectile})
	ecs.AddComponent(em, proj1, &components.PositionComponent{X: 300, Y: 150})
	ecs.AddComponent(em, proj1, &components.VelocityComponent{VX: 400})

	proj2 := em.CreateEntity()
	ecs.AddComponent(em, proj2, &components.BehaviorComponent{Type: components.BehaviorPeaProjectile})
	ecs.AddComponent(em, proj2, &components.PositionComponent{X: 350, Y: 250})
	ecs.AddComponent(em, proj2, &components.VelocityComponent{VX: 400})
	ecs.AddComponent(em, proj2, &components.CollisionComponent{Width: 20, Height: 20})

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证子弹数据
	if len(data.Projectiles) != 2 {
		t.Fatalf("Expected 2 projectiles, got %d", len(data.Projectiles))
	}

	// 验证子弹速度
	for _, p := range data.Projectiles {
		if p.VelocityX != 400 {
			t.Errorf("Projectile velocity mismatch: expected 400, got %f", p.VelocityX)
		}
		if p.Type != "pea" {
			t.Errorf("Projectile type mismatch: expected 'pea', got %q", p.Type)
		}
	}
}

// TestBattleSerializer_SaveAndLoadBattle_CompleteScenario 测试完整的战斗场景
func TestBattleSerializer_SaveAndLoadBattle_CompleteScenario(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "complete")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:                 300,
		LevelTime:           90.5,
		CurrentWaveIndex:    3,
		SpawnedWaves:        []bool{true, true, true, false, false},
		TotalZombiesSpawned: 15,
		ZombiesKilled:       12,
		CurrentLevel:        &config.LevelConfig{ID: "1-4"},
	}

	// 创建多种植物
	plant1 := em.CreateEntity()
	ecs.AddComponent(em, plant1, &components.PlantComponent{PlantType: components.PlantPeashooter, GridRow: 0, GridCol: 2})
	ecs.AddComponent(em, plant1, &components.PositionComponent{X: 200, Y: 50})
	ecs.AddComponent(em, plant1, &components.HealthComponent{CurrentHealth: 300, MaxHealth: 300})

	plant2 := em.CreateEntity()
	ecs.AddComponent(em, plant2, &components.PlantComponent{PlantType: components.PlantSunflower, GridRow: 1, GridCol: 0})
	ecs.AddComponent(em, plant2, &components.PositionComponent{X: 100, Y: 100})
	ecs.AddComponent(em, plant2, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})
	ecs.AddComponent(em, plant2, &components.TimerComponent{Name: "sun_production", TargetTime: 20, CurrentTime: 15})

	plant3 := em.CreateEntity()
	ecs.AddComponent(em, plant3, &components.PlantComponent{PlantType: components.PlantWallnut, GridRow: 2, GridCol: 4})
	ecs.AddComponent(em, plant3, &components.PositionComponent{X: 300, Y: 150})
	ecs.AddComponent(em, plant3, &components.HealthComponent{CurrentHealth: 2500, MaxHealth: 4000})

	// 创建多种僵尸
	zombie1 := em.CreateEntity()
	ecs.AddComponent(em, zombie1, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	ecs.AddComponent(em, zombie1, &components.PositionComponent{X: 600, Y: 100})
	ecs.AddComponent(em, zombie1, &components.VelocityComponent{VX: -23})
	ecs.AddComponent(em, zombie1, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})
	ecs.AddComponent(em, zombie1, &components.ZombieTargetLaneComponent{TargetRow: 1})

	zombie2 := em.CreateEntity()
	ecs.AddComponent(em, zombie2, &components.BehaviorComponent{Type: components.BehaviorZombieBuckethead})
	ecs.AddComponent(em, zombie2, &components.PositionComponent{X: 550, Y: 200})
	ecs.AddComponent(em, zombie2, &components.VelocityComponent{VX: -23})
	ecs.AddComponent(em, zombie2, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})
	ecs.AddComponent(em, zombie2, &components.ArmorComponent{CurrentArmor: 900, MaxArmor: 1100})
	ecs.AddComponent(em, zombie2, &components.ZombieTargetLaneComponent{TargetRow: 2})

	// 创建除草车
	for lane := 1; lane <= 5; lane++ {
		lm := em.CreateEntity()
		ecs.AddComponent(em, lm, &components.LawnmowerComponent{Lane: lane, IsTriggered: false})
		ecs.AddComponent(em, lm, &components.PositionComponent{X: 50, Y: float64(lane * 100)})
	}

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证所有数据
	if data.LevelID != "1-4" {
		t.Errorf("LevelID mismatch: expected '1-4', got %q", data.LevelID)
	}
	if data.Sun != 300 {
		t.Errorf("Sun mismatch: expected 300, got %d", data.Sun)
	}
	if data.LevelTime != 90.5 {
		t.Errorf("LevelTime mismatch: expected 90.5, got %f", data.LevelTime)
	}
	if data.TotalZombiesSpawned != 15 {
		t.Errorf("TotalZombiesSpawned mismatch: expected 15, got %d", data.TotalZombiesSpawned)
	}
	if data.ZombiesKilled != 12 {
		t.Errorf("ZombiesKilled mismatch: expected 12, got %d", data.ZombiesKilled)
	}
	if len(data.Plants) != 3 {
		t.Errorf("Plants count mismatch: expected 3, got %d", len(data.Plants))
	}
	if len(data.Zombies) != 2 {
		t.Errorf("Zombies count mismatch: expected 2, got %d", len(data.Zombies))
	}
	if len(data.Lawnmowers) != 5 {
		t.Errorf("Lawnmowers count mismatch: expected 5, got %d", len(data.Lawnmowers))
	}

	// 验证坚果墙数据
	var foundWallnut bool
	for _, p := range data.Plants {
		if p.PlantType == components.PlantWallnut.String() {
			foundWallnut = true
			if p.Health != 2500 {
				t.Errorf("Wallnut health mismatch: expected 2500, got %d", p.Health)
			}
		}
	}
	if !foundWallnut {
		t.Error("Wallnut not found in saved data")
	}

	// 验证铁桶僵尸数据
	var foundBuckethead bool
	for _, z := range data.Zombies {
		if z.ZombieType == "buckethead" {
			foundBuckethead = true
			if z.ArmorHealth != 900 {
				t.Errorf("Buckethead armor health mismatch: expected 900, got %d", z.ArmorHealth)
			}
		}
	}
	if !foundBuckethead {
		t.Error("Buckethead zombie not found in saved data")
	}
}

// --- 教学数据测试 (Story 20.3 QA Fix) ---

// TestBattleSerializer_CollectTutorialData_NoTutorial 测试非教学关卡
func TestBattleSerializer_CollectTutorialData_NoTutorial(t *testing.T) {
	serializer := NewBattleSerializer(nil)
	em := ecs.NewEntityManager()

	// 没有教学组件的实体
	tutorialData := serializer.collectTutorialData(em)

	if tutorialData != nil {
		t.Error("Expected nil tutorial data for non-tutorial level")
	}
}

// TestBattleSerializer_CollectTutorialData_WithTutorial 测试教学关卡
func TestBattleSerializer_CollectTutorialData_WithTutorial(t *testing.T) {
	serializer := NewBattleSerializer(nil)
	em := ecs.NewEntityManager()

	// 创建教学实体
	tutorialEntity := em.CreateEntity()
	ecs.AddComponent(em, tutorialEntity, &components.TutorialComponent{
		CurrentStepIndex: 3,
		IsActive:         true,
		CompletedSteps: map[string]bool{
			"step1": true,
			"step2": true,
			"step3": true,
		},
	})

	// 创建一些植物实体
	plant1 := em.CreateEntity()
	ecs.AddComponent(em, plant1, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})

	plant2 := em.CreateEntity()
	ecs.AddComponent(em, plant2, &components.PlantComponent{
		PlantType: components.PlantPeashooter,
	})

	plant3 := em.CreateEntity()
	ecs.AddComponent(em, plant3, &components.PlantComponent{
		PlantType: components.PlantSunflower,
	})

	// 收集教学数据
	tutorialData := serializer.collectTutorialData(em)

	if tutorialData == nil {
		t.Fatal("Expected tutorial data, got nil")
	}

	if tutorialData.CurrentStepIndex != 3 {
		t.Errorf("Expected CurrentStepIndex 3, got %d", tutorialData.CurrentStepIndex)
	}

	if !tutorialData.IsActive {
		t.Error("Expected IsActive to be true")
	}

	if len(tutorialData.CompletedSteps) != 3 {
		t.Errorf("Expected 3 completed steps, got %d", len(tutorialData.CompletedSteps))
	}

	if tutorialData.PlantCount != 3 {
		t.Errorf("Expected PlantCount 3, got %d", tutorialData.PlantCount)
	}

	if tutorialData.SunflowerCount != 2 {
		t.Errorf("Expected SunflowerCount 2, got %d", tutorialData.SunflowerCount)
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithTutorial 测试带教学数据的存档
func TestBattleSerializer_SaveAndLoadBattle_WithTutorial(t *testing.T) {
	gdataManager := createTestGdataManagerForBattle(t, "with_tutorial")
	if gdataManager == nil {
		t.Skip("Cannot create gdata manager for testing")
	}

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          100,
		SpawnedWaves: []bool{true},
		CurrentLevel: &config.LevelConfig{ID: "1-1"},
	}

	// 创建教学实体
	tutorialEntity := em.CreateEntity()
	ecs.AddComponent(em, tutorialEntity, &components.TutorialComponent{
		CurrentStepIndex: 2,
		IsActive:         true,
		CompletedSteps: map[string]bool{
			"intro":       true,
			"plantFirst":  true,
		},
	})

	// 创建向日葵
	sunflower := em.CreateEntity()
	ecs.AddComponent(em, sunflower, &components.PlantComponent{
		PlantType: components.PlantSunflower,
		GridRow:   0,
		GridCol:   1,
	})
	ecs.AddComponent(em, sunflower, &components.PositionComponent{X: 150, Y: 80})
	ecs.AddComponent(em, sunflower, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})

	// 保存
	serializer := NewBattleSerializer(gdataManager)
	err := serializer.SaveBattle(em, gs, "testuser")
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle("testuser")
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证教学数据
	if data.Tutorial == nil {
		t.Fatal("Expected tutorial data, got nil")
	}

	if data.Tutorial.CurrentStepIndex != 2 {
		t.Errorf("Expected CurrentStepIndex 2, got %d", data.Tutorial.CurrentStepIndex)
	}

	if !data.Tutorial.IsActive {
		t.Error("Expected IsActive to be true")
	}

	if len(data.Tutorial.CompletedSteps) != 2 {
		t.Errorf("Expected 2 completed steps, got %d", len(data.Tutorial.CompletedSteps))
	}

	if data.Tutorial.PlantCount != 1 {
		t.Errorf("Expected PlantCount 1, got %d", data.Tutorial.PlantCount)
	}

	if data.Tutorial.SunflowerCount != 1 {
		t.Errorf("Expected SunflowerCount 1, got %d", data.Tutorial.SunflowerCount)
	}
}
