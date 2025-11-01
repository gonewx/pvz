package systems

import (
	"testing"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/game"
)

// TestPlantDeathReleasesGrid 验证植物死亡时释放网格占用
// Bug Fix: 确保僵尸吃掉植物后，玩家可以在该位置重新种植植物
func TestPlantDeathReleasesGrid(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	lgs := NewLawnGridSystem(em, []int{1, 2, 3, 4, 5})

	// 创建草坪网格实体
	gridID := em.CreateEntity()
	ecs.AddComponent(em, gridID, &components.LawnGridComponent{})

	// 创建 BehaviorSystem（传入 LawnGridSystem）
	bs := NewBehaviorSystem(em, rm, rs, gs, lgs, gridID)

	// 创建植物实体在网格 (2, 3)
	plantID := em.CreateEntity()
	ecs.AddComponent(em, plantID, &components.PlantComponent{
		PlantType: components.PlantPeashooter,
		GridCol:   2,
		GridRow:   3,
	})
	ecs.AddComponent(em, plantID, &components.HealthComponent{
		CurrentHealth: 100,
		MaxHealth:     300,
	})
	ecs.AddComponent(em, plantID, &components.PositionComponent{
		X: config.GridWorldStartX + 2*config.CellWidth + config.CellWidth/2,
		Y: config.GridWorldStartY + 3*config.CellHeight + config.CellHeight/2,
	})

	// 标记网格为占用
	err := lgs.OccupyCell(gridID, 2, 3, plantID)
	if err != nil {
		t.Fatalf("Failed to occupy cell: %v", err)
	}

	// 验证网格已占用
	if !lgs.IsOccupied(gridID, 2, 3) {
		t.Fatal("Grid should be occupied before plant death")
	}

	// 创建僵尸并模拟啃食植物至死亡
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieEating,
	})
	ecs.AddComponent(em, zombieID, &components.TimerComponent{
		Name:        "eating_damage",
		TargetTime:  config.ZombieEatingDamageInterval,
		CurrentTime: config.ZombieEatingDamageInterval, // 计时器已满
		IsReady:     true,
	})
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: config.GridWorldStartX + 2*config.CellWidth + config.CellWidth/2,   // 与植物同列
		Y: config.GridWorldStartY + 3*config.CellHeight + config.CellHeight/2, // 与植物同行
	})

	// 模拟僵尸啃食，造成足够伤害杀死植物
	// 植物生命值 100，每次伤害 100，需要 1 次更新
	bs.Update(0.1)

	// 验证网格已释放（这是 Bug Fix 的核心验证）
	if lgs.IsOccupied(gridID, 2, 3) {
		t.Fatal("Grid should be released after plant death (Bug Fix failed)")
	}

	t.Log("✅ Bug Fix verified: Grid is released after plant death, allowing replanting")
}

// TestMultiplePlantsDeathReleasesGrid 验证多个植物同时死亡时都正确释放网格
func TestMultiplePlantsDeathReleasesGrid(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	lgs := NewLawnGridSystem(em, []int{1, 2, 3, 4, 5})

	gridID := em.CreateEntity()
	ecs.AddComponent(em, gridID, &components.LawnGridComponent{})

	bs := NewBehaviorSystem(em, rm, rs, gs, lgs, gridID)

	// 创建 3 个植物在不同网格位置
	plantPositions := [][2]int{{0, 0}, {1, 1}, {2, 2}}
	plantIDs := make([]ecs.EntityID, 3)

	for i, pos := range plantPositions {
		col, row := pos[0], pos[1]
		plantID := em.CreateEntity()
		ecs.AddComponent(em, plantID, &components.PlantComponent{
			PlantType: components.PlantSunflower,
			GridCol:   col,
			GridRow:   row,
		})
		ecs.AddComponent(em, plantID, &components.HealthComponent{
			CurrentHealth: 100,
			MaxHealth:     300,
		})
		ecs.AddComponent(em, plantID, &components.PositionComponent{
			X: config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2,
			Y: config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2,
		})

		lgs.OccupyCell(gridID, col, row, plantID)
		plantIDs[i] = plantID
	}

	// 验证所有网格都已占用
	for _, pos := range plantPositions {
		col, row := pos[0], pos[1]
		if !lgs.IsOccupied(gridID, col, row) {
			t.Fatalf("Grid (%d, %d) should be occupied", col, row)
		}
	}

	// 创建 3 个僵尸分别啃食 3 个植物
	for i, pos := range plantPositions {
		col, row := pos[0], pos[1]
		zombieID := em.CreateEntity()
		ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
			Type: components.BehaviorZombieEating,
		})
		ecs.AddComponent(em, zombieID, &components.TimerComponent{
			Name:        "eating_damage",
			TargetTime:  config.ZombieEatingDamageInterval,
			CurrentTime: config.ZombieEatingDamageInterval,
			IsReady:     true,
		})
		ecs.AddComponent(em, zombieID, &components.PositionComponent{
			X: config.GridWorldStartX + float64(col)*config.CellWidth + config.CellWidth/2,
			Y: config.GridWorldStartY + float64(row)*config.CellHeight + config.CellHeight/2,
		})

		t.Logf("Zombie %d eating plant %d at grid (%d, %d)", i, plantIDs[i], col, row)
	}

	// 更新系统，触发所有植物死亡
	bs.Update(0.1)

	// 验证所有网格都已释放
	for _, pos := range plantPositions {
		col, row := pos[0], pos[1]
		if lgs.IsOccupied(gridID, col, row) {
			t.Errorf("Grid (%d, %d) should be released after plant death", col, row)
		}
	}

	t.Log("✅ All grids released correctly after multiple plant deaths")
}

// TestCherryBombExplosionReleasesGrid 验证樱桃炸弹爆炸后释放网格占用
// Bug Fix: 确保樱桃炸弹爆炸后，玩家可以在该位置重新种植植物
func TestCherryBombExplosionReleasesGrid(t *testing.T) {
	// 创建测试环境
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(getTestAudioContext())
	rs := NewReanimSystem(em)
	gs := game.GetGameState()
	lgs := NewLawnGridSystem(em, []int{1, 2, 3, 4, 5})

	// 创建草坪网格实体
	gridID := em.CreateEntity()
	ecs.AddComponent(em, gridID, &components.LawnGridComponent{})

	// 创建 BehaviorSystem
	bs := NewBehaviorSystem(em, rm, rs, gs, lgs, gridID)

	// 创建樱桃炸弹实体在网格 (4, 2)
	cherryBombID := em.CreateEntity()
	ecs.AddComponent(em, cherryBombID, &components.PlantComponent{
		PlantType: components.PlantCherryBomb,
		GridCol:   4,
		GridRow:   2,
	})
	ecs.AddComponent(em, cherryBombID, &components.BehaviorComponent{
		Type: components.BehaviorCherryBomb,
	})
	ecs.AddComponent(em, cherryBombID, &components.PositionComponent{
		X: config.GridWorldStartX + 4*config.CellWidth + config.CellWidth/2,
		Y: config.GridWorldStartY + 2*config.CellHeight + config.CellHeight/2,
	})
	ecs.AddComponent(em, cherryBombID, &components.TimerComponent{
		Name:        "fuse",
		TargetTime:  config.CherryBombFuseTime,
		CurrentTime: config.CherryBombFuseTime, // 引信计时完成，准备爆炸
		IsReady:     true,
	})

	// 标记网格为占用
	err := lgs.OccupyCell(gridID, 4, 2, cherryBombID)
	if err != nil {
		t.Fatalf("Failed to occupy cell: %v", err)
	}

	// 验证网格已占用
	if !lgs.IsOccupied(gridID, 4, 2) {
		t.Fatal("Grid should be occupied before cherry bomb explosion")
	}

	// 创建僵尸用于测试爆炸范围（可选，不影响网格释放测试）
	zombieID := em.CreateEntity()
	ecs.AddComponent(em, zombieID, &components.BehaviorComponent{
		Type: components.BehaviorZombieBasic,
	})
	ecs.AddComponent(em, zombieID, &components.HealthComponent{
		CurrentHealth: 270,
		MaxHealth:     270,
	})
	ecs.AddComponent(em, zombieID, &components.PositionComponent{
		X: config.GridWorldStartX + 5*config.CellWidth,   // 樱桃炸弹右侧一格
		Y: config.GridWorldStartY + 2*config.CellHeight + config.CellHeight/2,
	})

	// 触发更新，樱桃炸弹应该爆炸
	bs.Update(0.1)

	// 验证网格已释放（这是 Bug Fix 的核心验证）
	if lgs.IsOccupied(gridID, 4, 2) {
		t.Fatal("Grid should be released after cherry bomb explosion (Bug Fix failed)")
	}

	t.Log("✅ Bug Fix verified: Grid is released after cherry bomb explosion, allowing replanting")
}
