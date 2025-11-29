package game

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"
)

// TestBattleSaveData_NewBattleSaveData 测试创建新的存档数据结构
func TestBattleSaveData_NewBattleSaveData(t *testing.T) {
	data := NewBattleSaveData()

	if data == nil {
		t.Fatal("NewBattleSaveData returned nil")
	}

	if data.Version != BattleSaveVersion {
		t.Errorf("Expected version %d, got %d", BattleSaveVersion, data.Version)
	}

	if data.SaveTime.IsZero() {
		t.Error("Expected SaveTime to be set, got zero value")
	}

	// 验证切片已初始化
	if data.SpawnedWaves == nil {
		t.Error("SpawnedWaves should be initialized")
	}
	if data.Plants == nil {
		t.Error("Plants should be initialized")
	}
	if data.Zombies == nil {
		t.Error("Zombies should be initialized")
	}
	if data.Projectiles == nil {
		t.Error("Projectiles should be initialized")
	}
	if data.Suns == nil {
		t.Error("Suns should be initialized")
	}
	if data.Lawnmowers == nil {
		t.Error("Lawnmowers should be initialized")
	}
}

// TestBattleSaveData_ToBattleSaveInfo 测试转换为预览信息
func TestBattleSaveData_ToBattleSaveInfo(t *testing.T) {
	now := time.Now()
	data := &BattleSaveData{
		LevelID:          "1-3",
		SaveTime:         now,
		Sun:              150,
		CurrentWaveIndex: 5,
	}

	info := data.ToBattleSaveInfo()

	if info == nil {
		t.Fatal("ToBattleSaveInfo returned nil")
	}

	if info.LevelID != "1-3" {
		t.Errorf("Expected LevelID '1-3', got %q", info.LevelID)
	}

	if info.Sun != 150 {
		t.Errorf("Expected Sun 150, got %d", info.Sun)
	}

	if info.WaveIndex != 5 {
		t.Errorf("Expected WaveIndex 5, got %d", info.WaveIndex)
	}

	if !info.SaveTime.Equal(now) {
		t.Errorf("Expected SaveTime %v, got %v", now, info.SaveTime)
	}
}

// TestBattleSaveData_GobSerialization 测试 gob 序列化/反序列化
func TestBattleSaveData_GobSerialization(t *testing.T) {
	original := &BattleSaveData{
		Version:             BattleSaveVersion,
		SaveTime:            time.Now(),
		LevelID:             "1-2",
		LevelTime:           45.5,
		CurrentWaveIndex:    3,
		SpawnedWaves:        []bool{true, true, true, false, false},
		TotalZombiesSpawned: 10,
		ZombiesKilled:       7,
		Sun:                 200,
		Plants: []PlantData{
			{PlantType: "peashooter", GridRow: 2, GridCol: 3, Health: 300, MaxHealth: 300},
			{PlantType: "sunflower", GridRow: 1, GridCol: 1, Health: 200, MaxHealth: 200},
		},
		Zombies: []ZombieData{
			{ZombieType: "basic", X: 500, Y: 100, Health: 150, MaxHealth: 200, Lane: 1},
		},
		Projectiles: []ProjectileData{
			{Type: "pea", X: 300, Y: 100, VelocityX: 400, Damage: 20},
		},
		Suns: []SunData{
			{X: 200, Y: 150, Value: 25, Lifetime: 5.0},
		},
		Lawnmowers: []LawnmowerData{
			{Lane: 1, X: 100, Triggered: false, Active: false},
			{Lane: 2, X: 100, Triggered: true, Active: true},
		},
	}

	// 序列化
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(original)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}

	// 反序列化
	dec := gob.NewDecoder(&buf)
	var loaded BattleSaveData
	err = dec.Decode(&loaded)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// 验证基本字段
	if loaded.Version != original.Version {
		t.Errorf("Version mismatch: expected %d, got %d", original.Version, loaded.Version)
	}
	if loaded.LevelID != original.LevelID {
		t.Errorf("LevelID mismatch: expected %q, got %q", original.LevelID, loaded.LevelID)
	}
	if loaded.LevelTime != original.LevelTime {
		t.Errorf("LevelTime mismatch: expected %f, got %f", original.LevelTime, loaded.LevelTime)
	}
	if loaded.Sun != original.Sun {
		t.Errorf("Sun mismatch: expected %d, got %d", original.Sun, loaded.Sun)
	}
	if loaded.CurrentWaveIndex != original.CurrentWaveIndex {
		t.Errorf("CurrentWaveIndex mismatch: expected %d, got %d", original.CurrentWaveIndex, loaded.CurrentWaveIndex)
	}

	// 验证切片长度
	if len(loaded.SpawnedWaves) != len(original.SpawnedWaves) {
		t.Errorf("SpawnedWaves length mismatch: expected %d, got %d", len(original.SpawnedWaves), len(loaded.SpawnedWaves))
	}
	if len(loaded.Plants) != len(original.Plants) {
		t.Errorf("Plants length mismatch: expected %d, got %d", len(original.Plants), len(loaded.Plants))
	}
	if len(loaded.Zombies) != len(original.Zombies) {
		t.Errorf("Zombies length mismatch: expected %d, got %d", len(original.Zombies), len(loaded.Zombies))
	}
	if len(loaded.Projectiles) != len(original.Projectiles) {
		t.Errorf("Projectiles length mismatch: expected %d, got %d", len(original.Projectiles), len(loaded.Projectiles))
	}
	if len(loaded.Suns) != len(original.Suns) {
		t.Errorf("Suns length mismatch: expected %d, got %d", len(original.Suns), len(loaded.Suns))
	}
	if len(loaded.Lawnmowers) != len(original.Lawnmowers) {
		t.Errorf("Lawnmowers length mismatch: expected %d, got %d", len(original.Lawnmowers), len(loaded.Lawnmowers))
	}

	// 验证植物数据
	if len(loaded.Plants) > 0 {
		if loaded.Plants[0].PlantType != "peashooter" {
			t.Errorf("Plant type mismatch: expected 'peashooter', got %q", loaded.Plants[0].PlantType)
		}
		if loaded.Plants[0].GridRow != 2 || loaded.Plants[0].GridCol != 3 {
			t.Errorf("Plant grid position mismatch: expected (2, 3), got (%d, %d)", loaded.Plants[0].GridRow, loaded.Plants[0].GridCol)
		}
	}

	// 验证僵尸数据
	if len(loaded.Zombies) > 0 {
		if loaded.Zombies[0].ZombieType != "basic" {
			t.Errorf("Zombie type mismatch: expected 'basic', got %q", loaded.Zombies[0].ZombieType)
		}
		if loaded.Zombies[0].Health != 150 {
			t.Errorf("Zombie health mismatch: expected 150, got %d", loaded.Zombies[0].Health)
		}
	}
}

// TestPlantData_GobSerialization 测试植物数据序列化
func TestPlantData_GobSerialization(t *testing.T) {
	original := PlantData{
		PlantType:       "wallnut",
		GridRow:         3,
		GridCol:         5,
		Health:          3000,
		MaxHealth:       4000,
		AttackCooldown:  1.5,
		BlinkTimer:      3.2,
		AttackAnimState: 1,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Failed to encode PlantData: %v", err)
	}

	var loaded PlantData
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&loaded); err != nil {
		t.Fatalf("Failed to decode PlantData: %v", err)
	}

	if loaded.PlantType != original.PlantType {
		t.Errorf("PlantType mismatch: expected %q, got %q", original.PlantType, loaded.PlantType)
	}
	if loaded.Health != original.Health {
		t.Errorf("Health mismatch: expected %d, got %d", original.Health, loaded.Health)
	}
	if loaded.AttackCooldown != original.AttackCooldown {
		t.Errorf("AttackCooldown mismatch: expected %f, got %f", original.AttackCooldown, loaded.AttackCooldown)
	}
}

// TestZombieData_GobSerialization 测试僵尸数据序列化
func TestZombieData_GobSerialization(t *testing.T) {
	original := ZombieData{
		ZombieType:   "buckethead",
		X:            600,
		Y:            200,
		VelocityX:    -23.0,
		Health:       200,
		MaxHealth:    200,
		ArmorHealth:  800,
		ArmorMax:     1100,
		Lane:         2,
		BehaviorType: "eating",
		IsEating:     true,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Failed to encode ZombieData: %v", err)
	}

	var loaded ZombieData
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&loaded); err != nil {
		t.Fatalf("Failed to decode ZombieData: %v", err)
	}

	if loaded.ZombieType != original.ZombieType {
		t.Errorf("ZombieType mismatch: expected %q, got %q", original.ZombieType, loaded.ZombieType)
	}
	if loaded.ArmorHealth != original.ArmorHealth {
		t.Errorf("ArmorHealth mismatch: expected %d, got %d", original.ArmorHealth, loaded.ArmorHealth)
	}
	if loaded.IsEating != original.IsEating {
		t.Errorf("IsEating mismatch: expected %v, got %v", original.IsEating, loaded.IsEating)
	}
}

// TestSunData_GobSerialization 测试阳光数据序列化
func TestSunData_GobSerialization(t *testing.T) {
	original := SunData{
		X:            300,
		Y:            250,
		VelocityY:    50,
		Lifetime:     8.0,
		Value:        25,
		IsCollecting: true,
		TargetX:      50,
		TargetY:      30,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Failed to encode SunData: %v", err)
	}

	var loaded SunData
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&loaded); err != nil {
		t.Fatalf("Failed to decode SunData: %v", err)
	}

	if loaded.Value != original.Value {
		t.Errorf("Value mismatch: expected %d, got %d", original.Value, loaded.Value)
	}
	if loaded.IsCollecting != original.IsCollecting {
		t.Errorf("IsCollecting mismatch: expected %v, got %v", original.IsCollecting, loaded.IsCollecting)
	}
	if loaded.TargetX != original.TargetX {
		t.Errorf("TargetX mismatch: expected %f, got %f", original.TargetX, loaded.TargetX)
	}
}

// TestLawnmowerData_GobSerialization 测试除草车数据序列化
func TestLawnmowerData_GobSerialization(t *testing.T) {
	original := LawnmowerData{
		Lane:      3,
		X:         120,
		Triggered: true,
		Active:    true,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Failed to encode LawnmowerData: %v", err)
	}

	var loaded LawnmowerData
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&loaded); err != nil {
		t.Fatalf("Failed to decode LawnmowerData: %v", err)
	}

	if loaded.Lane != original.Lane {
		t.Errorf("Lane mismatch: expected %d, got %d", original.Lane, loaded.Lane)
	}
	if loaded.Triggered != original.Triggered {
		t.Errorf("Triggered mismatch: expected %v, got %v", original.Triggered, loaded.Triggered)
	}
	if loaded.Active != original.Active {
		t.Errorf("Active mismatch: expected %v, got %v", original.Active, loaded.Active)
	}
}

// TestProjectileData_GobSerialization 测试子弹数据序列化
func TestProjectileData_GobSerialization(t *testing.T) {
	original := ProjectileData{
		Type:      "pea",
		X:         400,
		Y:         150,
		VelocityX: 400,
		Damage:    20,
		Lane:      2,
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Failed to encode ProjectileData: %v", err)
	}

	var loaded ProjectileData
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&loaded); err != nil {
		t.Fatalf("Failed to decode ProjectileData: %v", err)
	}

	if loaded.Type != original.Type {
		t.Errorf("Type mismatch: expected %q, got %q", original.Type, loaded.Type)
	}
	if loaded.Damage != original.Damage {
		t.Errorf("Damage mismatch: expected %d, got %d", original.Damage, loaded.Damage)
	}
}

// TestBattleSaveData_EmptyLists 测试空列表序列化
func TestBattleSaveData_EmptyLists(t *testing.T) {
	original := NewBattleSaveData()
	original.LevelID = "1-1"
	original.Sun = 50

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatalf("Failed to encode empty save data: %v", err)
	}

	var loaded BattleSaveData
	dec := gob.NewDecoder(&buf)
	if err := dec.Decode(&loaded); err != nil {
		t.Fatalf("Failed to decode empty save data: %v", err)
	}

	if len(loaded.Plants) != 0 {
		t.Errorf("Expected empty Plants list, got %d items", len(loaded.Plants))
	}
	if len(loaded.Zombies) != 0 {
		t.Errorf("Expected empty Zombies list, got %d items", len(loaded.Zombies))
	}
}
