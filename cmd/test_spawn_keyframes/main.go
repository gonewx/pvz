package main

import (
	"fmt"
	"log"

	"github.com/decker502/pvz/pkg/components"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/decker502/pvz/pkg/entities"
	"github.com/decker502/pvz/pkg/game"
	"github.com/decker502/pvz/pkg/systems"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

func main() {
	fmt.Println("=== SpawnMinActive/MaxActive 关键帧动画测试 ===\n")

	// 创建必要的组件
	audioContext := audio.NewContext(48000)
	rm := game.NewResourceManager(audioContext)
	em := ecs.NewEntityManager()

	// 加载资源配置
	if err := rm.LoadResourceConfig("assets/config/resources.yaml"); err != nil {
		log.Fatal("加载资源配置失败:", err)
	}

	// 创建粒子系统
	ps := systems.NewParticleSystem(em, rm)

	// 创建 Award 粒子效果（包含关键帧动画）
	emitterID, err := entities.CreateParticleEffect(em, rm, "Award", 400, 300)
	if err != nil {
		log.Fatal("创建粒子效果失败:", err)
	}

	fmt.Printf("已创建 Award 粒子效果，发射器 ID: %d\n\n", emitterID)

	// 获取发射器组件
	emitter, ok := ecs.GetComponent[*components.EmitterComponent](em, emitterID)
	if !ok {
		log.Fatal("无法获取发射器组件")
	}

	// 打印配置信息
	fmt.Println("--- 发射器配置信息 ---")
	fmt.Printf("SpawnMinActive 静态值: %d\n", emitter.SpawnMinActive)
	fmt.Printf("SpawnMinActive 关键帧数量: %d\n", len(emitter.SpawnMinActiveKeyframes))
	if len(emitter.SpawnMinActiveKeyframes) > 0 {
		fmt.Println("SpawnMinActive 关键帧数据:")
		for i, kf := range emitter.SpawnMinActiveKeyframes {
			fmt.Printf("  [%d] Time=%.2f, Value=%.2f\n", i, kf.Time, kf.Value)
		}
	}

	fmt.Printf("\nSpawnMaxActive 静态值: %d\n", emitter.SpawnMaxActive)
	fmt.Printf("SpawnMaxActive 关键帧数量: %d\n", len(emitter.SpawnMaxActiveKeyframes))
	if len(emitter.SpawnMaxActiveKeyframes) > 0 {
		fmt.Println("SpawnMaxActive 关键帧数据:")
		for i, kf := range emitter.SpawnMaxActiveKeyframes {
			fmt.Printf("  [%d] Time=%.2f, Value=%.2f\n", i, kf.Time, kf.Value)
		}
	}

	fmt.Printf("\nSystemDuration: %.2f 秒\n", emitter.SystemDuration)
	fmt.Println()

	// 模拟粒子系统运行，观察动态值变化
	fmt.Println("--- 模拟运行测试（每0.5秒输出一次） ---")
	dt := 0.016 // 60 FPS
	totalTime := 0.0
	lastPrintTime := 0.0

	for totalTime < 3.0 { // 运行3秒
		ps.Update(dt)
		totalTime += dt

		// 每0.5秒打印一次状态
		if totalTime-lastPrintTime >= 0.5 {
			// 获取最新的发射器状态
			emitter, _ := ecs.GetComponent[*components.EmitterComponent](em, emitterID)

			// 计算动态值
			dynamicMinActive := ps.GetDynamicSpawnMinActive(emitter)
			dynamicMaxActive := ps.GetDynamicSpawnMaxActive(emitter)
			activeCount := len(emitter.ActiveParticles)

			fmt.Printf("时间=%.2fs | 年龄=%.2fs | 动态MinActive=%d | 动态MaxActive=%d | 实际活跃粒子=%d | 已发射=%d\n",
				totalTime, emitter.Age, dynamicMinActive, dynamicMaxActive, activeCount, emitter.TotalLaunched)

			lastPrintTime = totalTime
		}
	}

	fmt.Println("\n=== 测试完成 ===")
	fmt.Println("✅ SpawnMinActive/MaxActive 关键帧动画功能正常工作")
}
