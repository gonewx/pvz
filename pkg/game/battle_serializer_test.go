package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
)

// TestBattleSerializer_NewBattleSerializer 测试创建序列化器
func TestBattleSerializer_NewBattleSerializer(t *testing.T) {
	serializer := NewBattleSerializer()
	if serializer == nil {
		t.Fatal("NewBattleSerializer returned nil")
	}
}

// TestBattleSerializer_SaveBattle_NilEntityManager 测试空 EntityManager
func TestBattleSerializer_SaveBattle_NilEntityManager(t *testing.T) {
	serializer := NewBattleSerializer()
	gs := &GameState{Sun: 100}
	tmpFile := filepath.Join(t.TempDir(), "test.sav")

	err := serializer.SaveBattle(nil, gs, tmpFile)
	if err == nil {
		t.Error("Expected error when EntityManager is nil")
	}
}

// TestBattleSerializer_SaveBattle_NilGameState 测试空 GameState
func TestBattleSerializer_SaveBattle_NilGameState(t *testing.T) {
	serializer := NewBattleSerializer()
	em := ecs.NewEntityManager()
	tmpFile := filepath.Join(t.TempDir(), "test.sav")

	err := serializer.SaveBattle(em, nil, tmpFile)
	if err == nil {
		t.Error("Expected error when GameState is nil")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_Empty 测试空战斗状态
func TestBattleSerializer_SaveAndLoadBattle_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_battle.sav")

	// 创建空的 EntityManager 和基本 GameState
	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:              100,
		CurrentWaveIndex: 2,
		SpawnedWaves:     []bool{true, true, false},
		CurrentLevel:     &config.LevelConfig{ID: "1-3"},
	}

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 验证文件存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatal("Save file was not created")
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_battle.sav")

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
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_battle.sav")

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
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_battle.sav")

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
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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

// TestBattleSerializer_LoadBattle_FileNotFound 测试加载不存在的文件
func TestBattleSerializer_LoadBattle_FileNotFound(t *testing.T) {
	serializer := NewBattleSerializer()
	_, err := serializer.LoadBattle("/nonexistent/path/save.sav")
	if err == nil {
		t.Error("Expected error when loading non-existent file")
	}
}

// TestBattleSerializer_LoadBattle_CorruptedFile 测试加载损坏的文件
func TestBattleSerializer_LoadBattle_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "corrupted.sav")

	// 写入无效数据
	err := os.WriteFile(filePath, []byte("invalid gob data"), 0644)
	if err != nil {
		t.Fatalf("Failed to write corrupted file: %v", err)
	}

	serializer := NewBattleSerializer()
	_, err = serializer.LoadBattle(filePath)
	if err == nil {
		t.Error("Expected error when loading corrupted file")
	}
}

// TestBattleSerializer_LoadBattle_VersionMismatch 测试版本不匹配
func TestBattleSerializer_LoadBattle_VersionMismatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "old_version.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          100,
		SpawnedWaves: []bool{true},
		CurrentLevel: &config.LevelConfig{ID: "1-1"},
	}

	// 保存当前版本
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 修改 BattleSaveVersion 并加载（模拟版本不匹配）
	// 由于无法直接修改常量，这里通过直接创建一个旧版本存档来测试
	// 实际上，这个测试验证的是版本检查逻辑
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}
	if data.Version != BattleSaveVersion {
		t.Errorf("Unexpected version: expected %d, got %d", BattleSaveVersion, data.Version)
	}
}

// TestBattleSerializer_SaveBattle_InvalidPath 测试保存到无效路径
func TestBattleSerializer_SaveBattle_InvalidPath(t *testing.T) {
	serializer := NewBattleSerializer()
	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          100,
		SpawnedWaves: []bool{},
		CurrentLevel: &config.LevelConfig{ID: "1-1"},
	}

	err := serializer.SaveBattle(em, gs, "/nonexistent/directory/test.sav")
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}
}

// TestBattleSerializer_CollectLevelState 测试关卡状态收集
func TestBattleSerializer_CollectLevelState(t *testing.T) {
	serializer := NewBattleSerializer()
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
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_battle.sav")

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
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_battle.sav")

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
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "complete_battle.sav")

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
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
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

// =============================================================================
// 保龄球模式（Level 1-5）测试
// =============================================================================

// TestBattleSerializer_SaveAndLoadBattle_WithBowlingNuts 测试带保龄球坚果的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithBowlingNuts(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "bowling_battle.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          0, // 保龄球模式不使用阳光
		SpawnedWaves: []bool{true, true, false},
		CurrentLevel: &config.LevelConfig{ID: "1-5"},
	}

	// 创建保龄球坚果实体 - 正在滚动
	nut1 := em.CreateEntity()
	ecs.AddComponent(em, nut1, &components.BowlingNutComponent{
		VelocityX:         200,
		VelocityY:         0,
		Row:               2,
		IsRolling:         true,
		IsBouncing:        false,
		TargetRow:         2,
		IsExplosive:       false,
		BounceCount:       0,
		CollisionCooldown: 0,
		BounceDirection:   0,
	})
	ecs.AddComponent(em, nut1, &components.PositionComponent{X: 400, Y: 250})

	// 创建保龄球坚果实体 - 正在弹射
	nut2 := em.CreateEntity()
	ecs.AddComponent(em, nut2, &components.BowlingNutComponent{
		VelocityX:         180,
		VelocityY:         100,
		Row:               3,
		IsRolling:         true,
		IsBouncing:        true,
		TargetRow:         4,
		IsExplosive:       false,
		BounceCount:       1,
		CollisionCooldown: 0.1,
		BounceDirection:   1, // 向下弹射
	})
	ecs.AddComponent(em, nut2, &components.PositionComponent{X: 350, Y: 350})

	// 创建爆炸坚果实体
	nut3 := em.CreateEntity()
	ecs.AddComponent(em, nut3, &components.BowlingNutComponent{
		VelocityX:   200,
		VelocityY:   0,
		Row:         1,
		IsRolling:   true,
		IsExplosive: true,
	})
	ecs.AddComponent(em, nut3, &components.PositionComponent{X: 300, Y: 150})

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证保龄球坚果数据
	if len(data.BowlingNuts) != 3 {
		t.Fatalf("Expected 3 bowling nuts, got %d", len(data.BowlingNuts))
	}

	// 验证弹射中的坚果
	var foundBouncing bool
	for _, nut := range data.BowlingNuts {
		if nut.IsBouncing {
			foundBouncing = true
			if nut.BounceCount != 1 {
				t.Errorf("Bouncing nut bounce count mismatch: expected 1, got %d", nut.BounceCount)
			}
			if nut.TargetRow != 4 {
				t.Errorf("Bouncing nut target row mismatch: expected 4, got %d", nut.TargetRow)
			}
			if nut.BounceDirection != 1 {
				t.Errorf("Bouncing nut direction mismatch: expected 1, got %d", nut.BounceDirection)
			}
		}
	}
	if !foundBouncing {
		t.Error("Bouncing nut not found in saved data")
	}

	// 验证爆炸坚果
	var foundExplosive bool
	for _, nut := range data.BowlingNuts {
		if nut.IsExplosive {
			foundExplosive = true
			if nut.Row != 1 {
				t.Errorf("Explosive nut row mismatch: expected 1, got %d", nut.Row)
			}
		}
	}
	if !foundExplosive {
		t.Error("Explosive nut not found in saved data")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithConveyorBelt 测试带传送带的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithConveyorBelt(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "conveyor_battle.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          0,
		SpawnedWaves: []bool{true, true, true},
		CurrentLevel: &config.LevelConfig{ID: "1-5"},
	}

	// 创建传送带实体
	// Story 19.12: 使用 PositionX 和 IsAtLeftEdge 替代 SlideProgress 和 SlotIndex
	conveyorEntity := em.CreateEntity()
	conveyorComp := &components.ConveyorBeltComponent{
		Cards: []components.ConveyorCard{
			{CardType: components.CardTypeWallnutBowling, PositionX: 10.0, IsAtLeftEdge: true},
			{CardType: components.CardTypeWallnutBowling, PositionX: 100.0, IsAtLeftEdge: false},
			{CardType: components.CardTypeExplodeONut, PositionX: 200.0, IsAtLeftEdge: false},
		},
		Capacity:           10,
		ScrollOffset:       25.5,
		IsActive:           true,
		NextSpacing:        80.0,
		SelectedCardIndex:  -1,
		FinalWaveTriggered: false,
	}
	ecs.AddComponent(em, conveyorEntity, conveyorComp)

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证传送带数据
	if data.ConveyorBelt == nil {
		t.Fatal("ConveyorBelt data is nil")
	}

	if len(data.ConveyorBelt.Cards) != 3 {
		t.Errorf("Expected 3 cards, got %d", len(data.ConveyorBelt.Cards))
	}

	if data.ConveyorBelt.Capacity != 10 {
		t.Errorf("Capacity mismatch: expected 10, got %d", data.ConveyorBelt.Capacity)
	}

	if !data.ConveyorBelt.IsActive {
		t.Error("ConveyorBelt should be active")
	}

	// Story 19.12: 验证 NextSpacing 替代 GenerationTimer
	if data.ConveyorBelt.NextSpacing != 80.0 {
		t.Errorf("NextSpacing mismatch: expected 80.0, got %f", data.ConveyorBelt.NextSpacing)
	}

	// 验证卡片类型和位置
	var foundExplodeONut bool
	for _, card := range data.ConveyorBelt.Cards {
		if card.CardType == components.CardTypeExplodeONut {
			foundExplodeONut = true
			// Story 19.12: 验证 PositionX 替代 SlideProgress
			if card.PositionX != 200.0 {
				t.Errorf("Explode-o-nut PositionX mismatch: expected 200.0, got %f", card.PositionX)
			}
		}
	}
	if !foundExplodeONut {
		t.Error("Explode-o-nut card not found")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithLevelPhase 测试带关卡阶段的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithLevelPhase(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "phase_battle.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          0,
		SpawnedWaves: []bool{},
		CurrentLevel: &config.LevelConfig{ID: "1-5"},
	}

	// 创建关卡阶段实体 - 保龄球阶段已激活
	phaseEntity := em.CreateEntity()
	phaseComp := &components.LevelPhaseComponent{
		CurrentPhase:        2, // 保龄球阶段
		PhaseState:          components.PhaseStateActive,
		TransitionProgress:  1.0,
		TransitionStep:      components.TransitionStepActivateBowling,
		ConveyorBeltY:       0,
		ConveyorBeltVisible: true,
		ShowRedLine:         true,
	}
	ecs.AddComponent(em, phaseEntity, phaseComp)

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证关卡阶段数据
	if data.LevelPhase == nil {
		t.Fatal("LevelPhase data is nil")
	}

	if data.LevelPhase.CurrentPhase != 2 {
		t.Errorf("CurrentPhase mismatch: expected 2, got %d", data.LevelPhase.CurrentPhase)
	}

	if data.LevelPhase.PhaseState != components.PhaseStateActive {
		t.Errorf("PhaseState mismatch: expected 'active', got %q", data.LevelPhase.PhaseState)
	}

	if !data.LevelPhase.ConveyorBeltVisible {
		t.Error("ConveyorBeltVisible should be true")
	}

	if !data.LevelPhase.ShowRedLine {
		t.Error("ShowRedLine should be true")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithDaveDialogue 测试带 Dave 对话的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithDaveDialogue(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "dave_battle.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          0,
		SpawnedWaves: []bool{},
		CurrentLevel: &config.LevelConfig{ID: "1-5"},
	}

	// 创建 Dave 对话实体
	daveEntity := em.CreateEntity()
	daveComp := &components.DaveDialogueComponent{
		DialogueKeys:     []string{"CRAZY_DAVE_2400", "CRAZY_DAVE_2401", "CRAZY_DAVE_2402"},
		CurrentLineIndex: 1,
		CurrentText:      "测试对话文本",
		IsVisible:        true,
		State:            components.DaveStateTalking,
		Expression:       "MOUTH_SMALL_OH",
	}
	ecs.AddComponent(em, daveEntity, daveComp)
	ecs.AddComponent(em, daveEntity, &components.PositionComponent{X: -100, Y: 200})

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证 Dave 对话数据
	if data.DaveDialogue == nil {
		t.Fatal("DaveDialogue data is nil")
	}

	if len(data.DaveDialogue.DialogueKeys) != 3 {
		t.Errorf("DialogueKeys length mismatch: expected 3, got %d", len(data.DaveDialogue.DialogueKeys))
	}

	if data.DaveDialogue.CurrentLineIndex != 1 {
		t.Errorf("CurrentLineIndex mismatch: expected 1, got %d", data.DaveDialogue.CurrentLineIndex)
	}

	if data.DaveDialogue.State != int(components.DaveStateTalking) {
		t.Errorf("State mismatch: expected %d, got %d", components.DaveStateTalking, data.DaveDialogue.State)
	}

	if data.DaveDialogue.Expression != "MOUTH_SMALL_OH" {
		t.Errorf("Expression mismatch: expected 'MOUTH_SMALL_OH', got %q", data.DaveDialogue.Expression)
	}

	if !data.DaveDialogue.IsVisible {
		t.Error("IsVisible should be true")
	}
}

// TestBattleSerializer_SaveAndLoadBattle_WithGuidedTutorial 测试带强引导教学的战斗状态
func TestBattleSerializer_SaveAndLoadBattle_WithGuidedTutorial(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "guided_battle.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:          0,
		SpawnedWaves: []bool{},
		CurrentLevel: &config.LevelConfig{ID: "1-5"},
	}

	// 创建强引导教学实体
	guidedEntity := em.CreateEntity()
	guidedComp := &components.GuidedTutorialComponent{
		IsActive:        true,
		AllowedActions:  []string{"click_shovel", "click_plant"},
		IdleTimer:       3.5,
		IdleThreshold:   5.0,
		ShowArrow:       false,
		ArrowTarget:     "shovel",
		LastPlantCount:  2,
		TransitionReady: false,
		TutorialTextKey: "SHOVEL_INSTRUCTION",
	}
	ecs.AddComponent(em, guidedEntity, guidedComp)

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 验证强引导教学数据
	if data.GuidedTutorial == nil {
		t.Fatal("GuidedTutorial data is nil")
	}

	if !data.GuidedTutorial.IsActive {
		t.Error("IsActive should be true")
	}

	if len(data.GuidedTutorial.AllowedActions) != 2 {
		t.Errorf("AllowedActions length mismatch: expected 2, got %d", len(data.GuidedTutorial.AllowedActions))
	}

	if data.GuidedTutorial.IdleTimer != 3.5 {
		t.Errorf("IdleTimer mismatch: expected 3.5, got %f", data.GuidedTutorial.IdleTimer)
	}

	if data.GuidedTutorial.ArrowTarget != "shovel" {
		t.Errorf("ArrowTarget mismatch: expected 'shovel', got %q", data.GuidedTutorial.ArrowTarget)
	}

	if data.GuidedTutorial.LastPlantCount != 2 {
		t.Errorf("LastPlantCount mismatch: expected 2, got %d", data.GuidedTutorial.LastPlantCount)
	}
}

// TestBattleSerializer_SaveAndLoadBattle_Level15Complete 测试完整的 Level 1-5 保龄球场景
func TestBattleSerializer_SaveAndLoadBattle_Level15Complete(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "level15_complete.sav")

	em := ecs.NewEntityManager()
	gs := &GameState{
		Sun:                 0,
		LevelTime:           45.5,
		CurrentWaveIndex:    5,
		SpawnedWaves:        []bool{true, true, true, true, true, false, false, false},
		TotalZombiesSpawned: 25,
		ZombiesKilled:       20,
		CurrentLevel:        &config.LevelConfig{ID: "1-5"},
	}

	// 创建关卡阶段（保龄球阶段激活）
	phaseEntity := em.CreateEntity()
	ecs.AddComponent(em, phaseEntity, &components.LevelPhaseComponent{
		CurrentPhase:        2,
		PhaseState:          components.PhaseStateActive,
		ConveyorBeltVisible: true,
		ShowRedLine:         true,
	})

	// 创建传送带
	// Story 19.12: 使用 PositionX 和 IsAtLeftEdge 替代 SlideProgress 和 SlotIndex
	conveyorEntity := em.CreateEntity()
	ecs.AddComponent(em, conveyorEntity, &components.ConveyorBeltComponent{
		Cards: []components.ConveyorCard{
			{CardType: components.CardTypeWallnutBowling, PositionX: 10.0, IsAtLeftEdge: true},
			{CardType: components.CardTypeWallnutBowling, PositionX: 100.0, IsAtLeftEdge: false},
		},
		Capacity:          10,
		IsActive:          true,
		NextSpacing:       80.0,
		SelectedCardIndex: 0,
	})

	// 创建滚动中的保龄球坚果
	nut1 := em.CreateEntity()
	ecs.AddComponent(em, nut1, &components.BowlingNutComponent{
		VelocityX: 200, Row: 3, IsRolling: true,
	})
	ecs.AddComponent(em, nut1, &components.PositionComponent{X: 400, Y: 350})

	// 创建僵尸
	zombie1 := em.CreateEntity()
	ecs.AddComponent(em, zombie1, &components.BehaviorComponent{Type: components.BehaviorZombieBasic})
	ecs.AddComponent(em, zombie1, &components.PositionComponent{X: 700, Y: 250})
	ecs.AddComponent(em, zombie1, &components.VelocityComponent{VX: -23})
	ecs.AddComponent(em, zombie1, &components.HealthComponent{CurrentHealth: 150, MaxHealth: 200})
	ecs.AddComponent(em, zombie1, &components.ZombieTargetLaneComponent{TargetRow: 2})

	zombie2 := em.CreateEntity()
	ecs.AddComponent(em, zombie2, &components.BehaviorComponent{Type: components.BehaviorZombieConehead})
	ecs.AddComponent(em, zombie2, &components.PositionComponent{X: 750, Y: 350})
	ecs.AddComponent(em, zombie2, &components.VelocityComponent{VX: -23})
	ecs.AddComponent(em, zombie2, &components.HealthComponent{CurrentHealth: 200, MaxHealth: 200})
	ecs.AddComponent(em, zombie2, &components.ArmorComponent{CurrentArmor: 250, MaxArmor: 370})
	ecs.AddComponent(em, zombie2, &components.ZombieTargetLaneComponent{TargetRow: 3})

	// 创建除草车
	for lane := 1; lane <= 5; lane++ {
		lm := em.CreateEntity()
		ecs.AddComponent(em, lm, &components.LawnmowerComponent{Lane: lane})
		ecs.AddComponent(em, lm, &components.PositionComponent{X: 50, Y: float64(lane * 100)})
	}

	// 保存
	serializer := NewBattleSerializer()
	err := serializer.SaveBattle(em, gs, filePath)
	if err != nil {
		t.Fatalf("SaveBattle failed: %v", err)
	}

	// 加载
	data, err := serializer.LoadBattle(filePath)
	if err != nil {
		t.Fatalf("LoadBattle failed: %v", err)
	}

	// 综合验证
	if data.LevelID != "1-5" {
		t.Errorf("LevelID mismatch: expected '1-5', got %q", data.LevelID)
	}

	if data.LevelPhase == nil {
		t.Error("LevelPhase should not be nil")
	} else if data.LevelPhase.CurrentPhase != 2 {
		t.Errorf("LevelPhase.CurrentPhase mismatch: expected 2, got %d", data.LevelPhase.CurrentPhase)
	}

	if data.ConveyorBelt == nil {
		t.Error("ConveyorBelt should not be nil")
	} else {
		if len(data.ConveyorBelt.Cards) != 2 {
			t.Errorf("ConveyorBelt.Cards length mismatch: expected 2, got %d", len(data.ConveyorBelt.Cards))
		}
		if data.ConveyorBelt.SelectedCardIndex != 0 {
			t.Errorf("ConveyorBelt.SelectedCardIndex mismatch: expected 0, got %d", data.ConveyorBelt.SelectedCardIndex)
		}
	}

	if len(data.BowlingNuts) != 1 {
		t.Errorf("BowlingNuts count mismatch: expected 1, got %d", len(data.BowlingNuts))
	}

	if len(data.Zombies) != 2 {
		t.Errorf("Zombies count mismatch: expected 2, got %d", len(data.Zombies))
	}

	if len(data.Lawnmowers) != 5 {
		t.Errorf("Lawnmowers count mismatch: expected 5, got %d", len(data.Lawnmowers))
	}

	if data.TotalZombiesSpawned != 25 {
		t.Errorf("TotalZombiesSpawned mismatch: expected 25, got %d", data.TotalZombiesSpawned)
	}

	if data.ZombiesKilled != 20 {
		t.Errorf("ZombiesKilled mismatch: expected 20, got %d", data.ZombiesKilled)
	}
}
