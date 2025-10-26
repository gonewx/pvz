package main

import (
	"fmt"
	"log"
	"math"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
)

func main() {
	// 创建实体管理器和资源管理器
	em := ecs.NewEntityManager()
	rm := game.NewResourceManager(nil)

	// 注意：不加载图片资源，只测试粒子生成逻辑
	// 粒子会创建但图片为 nil，不影响数量统计

	// 创建粒子系统
	ps := systems.NewParticleSystem(em, rm)

	// 创建 Planting 粒子效果
	log.Printf("=== 创建 Planting 粒子效果 ===")
	emitterID, err := entities.NewPlantingParticleEffect(em, rm, 400, 300)
	if err != nil {
		log.Fatalf("创建粒子效果失败: %v", err)
	}

	log.Printf("发射器 ID: %d", emitterID)

	// 获取发射器组件
	emitter, ok := ecs.GetComponent[*components.EmitterComponent](em, emitterID)
	if !ok {
		log.Fatal("获取发射器组件失败")
	}

	// 模拟帧更新
	dt := 1.0 / 60.0 // 60 FPS

	// 第一帧更新（应该生成粒子）
	log.Printf("\n=== 第一帧更新 ===")
	ps.Update(dt)
	em.RemoveMarkedEntities()

	log.Printf("发射器年龄: %.3fs", emitter.Age)
	log.Printf("总发射数: %d", emitter.TotalLaunched)
	log.Printf("活跃粒子数: %d", len(emitter.ActiveParticles))

	// 立即检查粒子初始位置（在粒子移动之前）
	log.Printf("\n--- 粒子初始位置分布检查（第1帧，刚生成时） ---")
	for _, particleID := range emitter.ActiveParticles {
		pos, ok := ecs.GetComponent[*components.PositionComponent](em, particleID)
		if ok {
			distX := pos.X - 400.0
			distY := pos.Y - 300.0
			dist := math.Sqrt(distX*distX + distY*distY)
			log.Printf("粒子 ID=%d: 位置=(%.1f, %.1f), 偏移=(%.1f, %.1f), 距离=%.1f",
				particleID, pos.X, pos.Y, distX, distY, dist)
		}
	}

	// 继续更新几帧，观察粒子数量变化
	for i := 2; i <= 10; i++ {
		log.Printf("\n=== 第 %d 帧更新 ===", i)
		ps.Update(dt)
		em.RemoveMarkedEntities()

		log.Printf("发射器年龄: %.3fs", emitter.Age)
		log.Printf("总发射数: %d", emitter.TotalLaunched)
		log.Printf("活跃粒子数: %d", len(emitter.ActiveParticles))

		// 如果发射器已停止且没有活跃粒子，提前退出
		if !emitter.Active && len(emitter.ActiveParticles) == 0 {
			log.Printf("\n发射器已完成并清理")
			break
		}
	}

	// 最终统计
	log.Printf("\n=== 最终统计 ===")
	log.Printf("总发射粒子数: %d (预期: 8)", emitter.TotalLaunched)
	log.Printf("EmitterRadiusMin: %.1f", emitter.EmitterRadiusMin)
	log.Printf("EmitterRadiusMax: %.1f (预期: 10)", emitter.EmitterRadiusMax)

	if emitter.TotalLaunched == 8 {
		fmt.Println("\n✅ 粒子数量测试通过（8个）")
	} else {
		fmt.Printf("\n❌ 粒子数量测试失败（预期8，实际%d）\n", emitter.TotalLaunched)
	}

	if emitter.EmitterRadiusMax == 10.0 {
		fmt.Println("✅ EmitterRadius 解析测试通过（max=10）")
	} else {
		fmt.Printf("❌ EmitterRadius 解析测试失败（预期max=10，实际max=%.1f）\n", emitter.EmitterRadiusMax)
	}
}
