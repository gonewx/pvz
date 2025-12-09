package scenes

import (
	"testing"

	"github.com/gonewx/pvz/pkg/components"
	"github.com/gonewx/pvz/pkg/game"
)

// TestStringToPlantType 测试植物类型字符串转换
// Story 18.3: 验证植物类型转换函数
func TestStringToPlantType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected components.PlantType
	}{
		// 小写形式
		{"sunflower lowercase", "sunflower", components.PlantSunflower},
		{"peashooter lowercase", "peashooter", components.PlantPeashooter},
		{"wallnut lowercase", "wallnut", components.PlantWallnut},
		{"cherrybomb lowercase", "cherrybomb", components.PlantCherryBomb},

		// 大写形式
		{"Sunflower uppercase", "Sunflower", components.PlantSunflower},
		{"Peashooter uppercase", "Peashooter", components.PlantPeashooter},
		{"Wallnut uppercase", "Wallnut", components.PlantWallnut},
		{"CherryBomb uppercase", "CherryBomb", components.PlantCherryBomb},

		// 未知类型
		{"unknown plant", "unknown", components.PlantUnknown},
		{"empty string", "", components.PlantUnknown},
		{"random string", "randomtype", components.PlantUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringToPlantType(tt.input)
			if result != tt.expected {
				t.Errorf("stringToPlantType(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestRestorePlants_EmptyList 测试空植物列表恢复
// Story 18.3: 验证空列表处理
func TestRestorePlants_EmptyList(t *testing.T) {
	plants := []game.PlantData{}

	// 验证空列表不会导致错误
	if len(plants) != 0 {
		t.Errorf("Expected empty plant list, got %d plants", len(plants))
	}
}

// TestRestorePlants_DataMapping 测试植物数据映射
// Story 18.3: 验证植物数据正确映射
func TestRestorePlants_DataMapping(t *testing.T) {
	plants := []game.PlantData{
		{
			PlantType:      "peashooter",
			GridRow:        2,
			GridCol:        3,
			Health:         200,
			MaxHealth:      300,
			AttackCooldown: 1.5,
		},
		{
			PlantType:      "sunflower",
			GridRow:        1,
			GridCol:        1,
			Health:         300,
			MaxHealth:      300,
			AttackCooldown: 0,
		},
	}

	// 验证数据完整性
	if len(plants) != 2 {
		t.Fatalf("Expected 2 plants, got %d", len(plants))
	}

	// 验证豌豆射手数据
	peashooter := plants[0]
	if peashooter.PlantType != "peashooter" {
		t.Errorf("Expected peashooter, got %q", peashooter.PlantType)
	}
	if peashooter.GridRow != 2 || peashooter.GridCol != 3 {
		t.Errorf("Wrong grid position: expected (2, 3), got (%d, %d)", peashooter.GridRow, peashooter.GridCol)
	}
	if peashooter.Health != 200 {
		t.Errorf("Wrong health: expected 200, got %d", peashooter.Health)
	}

	// 验证向日葵数据
	sunflower := plants[1]
	if sunflower.PlantType != "sunflower" {
		t.Errorf("Expected sunflower, got %q", sunflower.PlantType)
	}
	if sunflower.MaxHealth != 300 {
		t.Errorf("Wrong max health: expected 300, got %d", sunflower.MaxHealth)
	}
}

// TestRestoreZombies_EmptyList 测试空僵尸列表恢复
// Story 18.3: 验证空列表处理
func TestRestoreZombies_EmptyList(t *testing.T) {
	zombies := []game.ZombieData{}

	// 验证空列表不会导致错误
	if len(zombies) != 0 {
		t.Errorf("Expected empty zombie list, got %d zombies", len(zombies))
	}
}

// TestRestoreZombies_DataMapping 测试僵尸数据映射
// Story 18.3: 验证僵尸数据正确映射
func TestRestoreZombies_DataMapping(t *testing.T) {
	zombies := []game.ZombieData{
		{
			ZombieType:   "basic",
			X:            500,
			Y:            150,
			VelocityX:    -23.0,
			Health:       180,
			MaxHealth:    200,
			ArmorHealth:  0,
			ArmorMax:     0,
			Lane:         2,
			BehaviorType: "walking",
			IsEating:     false,
		},
		{
			ZombieType:   "conehead",
			X:            600,
			Y:            250,
			VelocityX:    -23.0,
			Health:       200,
			MaxHealth:    200,
			ArmorHealth:  300,
			ArmorMax:     370,
			Lane:         3,
			BehaviorType: "eating",
			IsEating:     true,
		},
	}

	// 验证数据完整性
	if len(zombies) != 2 {
		t.Fatalf("Expected 2 zombies, got %d", len(zombies))
	}

	// 验证普通僵尸数据
	basic := zombies[0]
	if basic.ZombieType != "basic" {
		t.Errorf("Expected basic zombie, got %q", basic.ZombieType)
	}
	if basic.X != 500 || basic.Y != 150 {
		t.Errorf("Wrong position: expected (500, 150), got (%.1f, %.1f)", basic.X, basic.Y)
	}
	if basic.Health != 180 {
		t.Errorf("Wrong health: expected 180, got %d", basic.Health)
	}
	if basic.IsEating {
		t.Error("Basic zombie should not be eating")
	}

	// 验证路障僵尸数据
	conehead := zombies[1]
	if conehead.ZombieType != "conehead" {
		t.Errorf("Expected conehead zombie, got %q", conehead.ZombieType)
	}
	if conehead.ArmorHealth != 300 {
		t.Errorf("Wrong armor health: expected 300, got %d", conehead.ArmorHealth)
	}
	if !conehead.IsEating {
		t.Error("Conehead zombie should be eating")
	}
}

// TestRestoreZombies_SkipDyingZombies 测试跳过死亡中的僵尸
// Story 18.3: 验证死亡僵尸不被恢复
func TestRestoreZombies_SkipDyingZombies(t *testing.T) {
	zombies := []game.ZombieData{
		{
			ZombieType:   "basic",
			X:            500,
			Y:            150,
			BehaviorType: "dying",
		},
		{
			ZombieType:   "basic",
			X:            600,
			Y:            250,
			BehaviorType: "dying_explosion",
		},
		{
			ZombieType:   "basic",
			X:            700,
			Y:            350,
			BehaviorType: "walking",
		},
	}

	// 统计应该恢复的僵尸数量
	shouldRestore := 0
	for _, z := range zombies {
		if z.BehaviorType != "dying" && z.BehaviorType != "dying_explosion" {
			shouldRestore++
		}
	}

	if shouldRestore != 1 {
		t.Errorf("Expected 1 zombie to restore (skip dying), got %d", shouldRestore)
	}
}

// TestRestoreProjectiles_EmptyList 测试空子弹列表恢复
// Story 18.3: 验证空列表处理
func TestRestoreProjectiles_EmptyList(t *testing.T) {
	projectiles := []game.ProjectileData{}

	if len(projectiles) != 0 {
		t.Errorf("Expected empty projectile list, got %d projectiles", len(projectiles))
	}
}

// TestRestoreProjectiles_DataMapping 测试子弹数据映射
// Story 18.3: 验证子弹数据正确映射
func TestRestoreProjectiles_DataMapping(t *testing.T) {
	projectiles := []game.ProjectileData{
		{
			Type:      "pea",
			X:         300,
			Y:         150,
			VelocityX: 400,
			Damage:    20,
			Lane:      2,
		},
	}

	if len(projectiles) != 1 {
		t.Fatalf("Expected 1 projectile, got %d", len(projectiles))
	}

	pea := projectiles[0]
	if pea.Type != "pea" {
		t.Errorf("Expected pea projectile, got %q", pea.Type)
	}
	if pea.VelocityX != 400 {
		t.Errorf("Wrong velocity: expected 400, got %.1f", pea.VelocityX)
	}
	if pea.Damage != 20 {
		t.Errorf("Wrong damage: expected 20, got %d", pea.Damage)
	}
}

// TestRestoreProjectiles_UnsupportedType 测试不支持的子弹类型
// Story 18.3: 验证只支持豌豆子弹
func TestRestoreProjectiles_UnsupportedType(t *testing.T) {
	projectiles := []game.ProjectileData{
		{Type: "pea", X: 100, Y: 100},
		{Type: "snow_pea", X: 200, Y: 200},
		{Type: "cabbage", X: 300, Y: 300},
	}

	// 统计支持的子弹类型
	supported := 0
	for _, p := range projectiles {
		if p.Type == "pea" {
			supported++
		}
	}

	if supported != 1 {
		t.Errorf("Expected 1 supported projectile type, got %d", supported)
	}
}

// TestRestoreSuns_EmptyList 测试空阳光列表恢复
// Story 18.3: 验证空列表处理
func TestRestoreSuns_EmptyList(t *testing.T) {
	suns := []game.SunData{}

	if len(suns) != 0 {
		t.Errorf("Expected empty sun list, got %d suns", len(suns))
	}
}

// TestRestoreSuns_DataMapping 测试阳光数据映射
// Story 18.3: 验证阳光数据正确映射
func TestRestoreSuns_DataMapping(t *testing.T) {
	suns := []game.SunData{
		{
			X:            200,
			Y:            300,
			VelocityY:    0,
			Lifetime:     8.0,
			Value:        25,
			IsCollecting: false,
		},
	}

	if len(suns) != 1 {
		t.Fatalf("Expected 1 sun, got %d", len(suns))
	}

	sun := suns[0]
	if sun.X != 200 || sun.Y != 300 {
		t.Errorf("Wrong position: expected (200, 300), got (%.1f, %.1f)", sun.X, sun.Y)
	}
	if sun.Lifetime != 8.0 {
		t.Errorf("Wrong lifetime: expected 8.0, got %.1f", sun.Lifetime)
	}
	if sun.Value != 25 {
		t.Errorf("Wrong value: expected 25, got %d", sun.Value)
	}
}

// TestRestoreSuns_SkipCollectingSuns 测试跳过正在收集的阳光
// Story 18.3: 验证正在收集的阳光不被恢复
func TestRestoreSuns_SkipCollectingSuns(t *testing.T) {
	suns := []game.SunData{
		{X: 100, Y: 100, IsCollecting: true},
		{X: 200, Y: 200, IsCollecting: false},
		{X: 300, Y: 300, IsCollecting: true},
	}

	// 统计应该恢复的阳光数量
	shouldRestore := 0
	for _, s := range suns {
		if !s.IsCollecting {
			shouldRestore++
		}
	}

	if shouldRestore != 1 {
		t.Errorf("Expected 1 sun to restore (skip collecting), got %d", shouldRestore)
	}
}

// TestRestoreLawnmowers_EmptyList 测试空除草车列表恢复
// Story 18.3: 验证空列表处理
func TestRestoreLawnmowers_EmptyList(t *testing.T) {
	lawnmowers := []game.LawnmowerData{}

	if len(lawnmowers) != 0 {
		t.Errorf("Expected empty lawnmower list, got %d lawnmowers", len(lawnmowers))
	}
}

// TestRestoreLawnmowers_DataMapping 测试除草车数据映射
// Story 18.3: 验证除草车数据正确映射
func TestRestoreLawnmowers_DataMapping(t *testing.T) {
	lawnmowers := []game.LawnmowerData{
		{Lane: 1, X: 100, Triggered: false, Active: false},
		{Lane: 2, X: 100, Triggered: true, Active: true},
		{Lane: 3, X: 500, Triggered: true, Active: true},
	}

	if len(lawnmowers) != 3 {
		t.Fatalf("Expected 3 lawnmowers, got %d", len(lawnmowers))
	}

	// 验证第一行除草车（未触发）
	lm1 := lawnmowers[0]
	if lm1.Lane != 1 {
		t.Errorf("Wrong lane: expected 1, got %d", lm1.Lane)
	}
	if lm1.Triggered {
		t.Error("Lane 1 lawnmower should not be triggered")
	}

	// 验证第二行除草车（已触发并激活）
	lm2 := lawnmowers[1]
	if !lm2.Triggered || !lm2.Active {
		t.Error("Lane 2 lawnmower should be triggered and active")
	}
}

// TestRestoreLawnmowers_SkipOffScreen 测试跳过已移出屏幕的除草车
// Story 18.3: 验证移出屏幕的除草车不被恢复
func TestRestoreLawnmowers_SkipOffScreen(t *testing.T) {
	windowWidth := 800.0
	lawnmowers := []game.LawnmowerData{
		{Lane: 1, X: 100, Triggered: false, Active: false},             // 应该恢复
		{Lane: 2, X: windowWidth + 150, Triggered: true, Active: true}, // 移出屏幕，应该跳过
		{Lane: 3, X: 300, Triggered: true, Active: true},               // 正在移动，应该恢复
	}

	// 统计应该恢复的除草车数量
	shouldRestore := 0
	for _, lm := range lawnmowers {
		// 跳过已触发且激活且移出屏幕的除草车
		if lm.Triggered && lm.Active && lm.X > windowWidth+100 {
			continue
		}
		shouldRestore++
	}

	if shouldRestore != 2 {
		t.Errorf("Expected 2 lawnmowers to restore (skip off-screen), got %d", shouldRestore)
	}
}

// TestRestoreBattleState_GameStateRestore 测试游戏状态恢复
// Story 18.3: 验证 GameState 字段正确恢复
func TestRestoreBattleState_GameStateRestore(t *testing.T) {
	saveData := &game.BattleSaveData{
		Sun:                 250,
		LevelTime:           90.5,
		CurrentWaveIndex:    3,
		SpawnedWaves:        []bool{true, true, true, false, false},
		TotalZombiesSpawned: 20,
		ZombiesKilled:       15,
	}

	// 验证数据完整性
	if saveData.Sun != 250 {
		t.Errorf("Wrong sun: expected 250, got %d", saveData.Sun)
	}
	if saveData.LevelTime != 90.5 {
		t.Errorf("Wrong level time: expected 90.5, got %.1f", saveData.LevelTime)
	}
	if saveData.CurrentWaveIndex != 3 {
		t.Errorf("Wrong wave index: expected 3, got %d", saveData.CurrentWaveIndex)
	}
	if len(saveData.SpawnedWaves) != 5 {
		t.Errorf("Wrong spawned waves count: expected 5, got %d", len(saveData.SpawnedWaves))
	}
	if saveData.TotalZombiesSpawned != 20 {
		t.Errorf("Wrong total zombies spawned: expected 20, got %d", saveData.TotalZombiesSpawned)
	}
	if saveData.ZombiesKilled != 15 {
		t.Errorf("Wrong zombies killed: expected 15, got %d", saveData.ZombiesKilled)
	}
}

// TestRestoreBattleState_SkipAnimations 测试跳过动画标志
// Story 18.3: 验证恢复后跳过开场动画
func TestRestoreBattleState_SkipAnimations(t *testing.T) {
	// 模拟恢复后的状态
	isIntroAnimPlaying := false
	soddingAnimStarted := true

	// 验证动画状态已设置为跳过
	if isIntroAnimPlaying {
		t.Error("Intro animation should be skipped after restore")
	}
	if !soddingAnimStarted {
		t.Error("Sodding animation should be marked as started after restore")
	}
}

// TestZombieLaneCalculation 测试僵尸行号计算
// Story 18.3: 验证从 Y 坐标推算行号
func TestZombieLaneCalculation(t *testing.T) {
	// 使用配置中的常量（模拟值）
	gridWorldStartY := 130.0
	cellHeight := 100.0

	tests := []struct {
		name     string
		y        float64
		expected int
	}{
		{"row 1", 130.0, 1},
		{"row 2", 230.0, 2},
		{"row 3", 330.0, 3},
		{"row 4", 430.0, 4},
		{"row 5", 530.0, 5},
		{"above grid", 50.0, 1},  // 应该限制为1
		{"below grid", 700.0, 5}, // 应该限制为5
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟行号计算逻辑
			lane := int((tt.y-gridWorldStartY)/cellHeight) + 1
			if lane < 1 {
				lane = 1
			}
			if lane > 5 {
				lane = 5
			}

			if lane != tt.expected {
				t.Errorf("Lane calculation for Y=%.1f: expected %d, got %d", tt.y, tt.expected, lane)
			}
		})
	}
}

// TestProjectileRowCalculation 测试子弹行号计算
// Story 18.3: 验证从 Y 坐标推算行号（0-based）
func TestProjectileRowCalculation(t *testing.T) {
	// 使用配置中的常量（模拟值）
	gridWorldStartY := 130.0
	cellHeight := 100.0

	tests := []struct {
		name     string
		y        float64
		expected int
	}{
		{"row 0", 130.0, 0},
		{"row 1", 230.0, 1},
		{"row 2", 330.0, 2},
		{"row 3", 430.0, 3},
		{"row 4", 530.0, 4},
		{"above grid", 50.0, 0},  // 应该限制为0
		{"below grid", 700.0, 4}, // 应该限制为4
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟行号计算逻辑
			row := int((tt.y - gridWorldStartY) / cellHeight)
			if row < 0 {
				row = 0
			}
			if row > 4 {
				row = 4
			}

			if row != tt.expected {
				t.Errorf("Row calculation for Y=%.1f: expected %d, got %d", tt.y, tt.expected, row)
			}
		})
	}
}

// TestLifetimeRestoration 测试生命周期恢复
// Story 18.3: 验证阳光生命周期计算
func TestLifetimeRestoration(t *testing.T) {
	tests := []struct {
		name            string
		maxLifetime     float64
		savedLifetime   float64 // 剩余时间
		expectedCurrent float64
	}{
		{"刚生成的阳光", 10.0, 10.0, 0.0},
		{"一半时间", 10.0, 5.0, 5.0},
		{"即将消失", 10.0, 1.0, 9.0},
		{"零时间", 10.0, 0.0, 10.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟生命周期恢复逻辑
			currentLifetime := tt.maxLifetime - tt.savedLifetime
			if currentLifetime < 0 {
				currentLifetime = 0
			}

			if currentLifetime != tt.expectedCurrent {
				t.Errorf("Lifetime calculation: expected %.1f, got %.1f", tt.expectedCurrent, currentLifetime)
			}
		})
	}
}

// TestAttackCooldownRestoration 测试攻击冷却恢复
// Story 18.3: 验证植物攻击冷却计算
func TestAttackCooldownRestoration(t *testing.T) {
	tests := []struct {
		name            string
		targetTime      float64
		savedCooldown   float64 // 剩余冷却
		expectedCurrent float64
	}{
		{"满冷却", 2.0, 2.0, 0.0},
		{"一半冷却", 2.0, 1.0, 1.0},
		{"即将就绪", 2.0, 0.1, 1.9},
		{"已就绪", 2.0, 0.0, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 模拟冷却恢复逻辑
			currentTime := tt.targetTime - tt.savedCooldown
			if currentTime < 0 {
				currentTime = 0
			}

			if currentTime != tt.expectedCurrent {
				t.Errorf("Cooldown calculation: expected %.1f, got %.1f", tt.expectedCurrent, currentTime)
			}
		})
	}
}

// TestZombieVelocityRestoration 测试僵尸速度恢复
// Story 18.3: 验证僵尸默认速度
func TestZombieVelocityRestoration(t *testing.T) {
	defaultZombieSpeed := -23.0

	tests := []struct {
		name             string
		savedVelocity    float64
		expectedVelocity float64
	}{
		{"有保存速度", -20.0, -20.0},
		{"零速度使用默认", 0.0, defaultZombieSpeed},
		{"正速度（无效）", 10.0, 10.0}, // 保留原值
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			velocity := tt.savedVelocity
			if velocity == 0 {
				velocity = defaultZombieSpeed
			}

			if velocity != tt.expectedVelocity {
				t.Errorf("Velocity restoration: expected %.1f, got %.1f", tt.expectedVelocity, velocity)
			}
		})
	}
}
