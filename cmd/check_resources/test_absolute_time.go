package main

import (
	"fmt"
	particlePkg "github.com/decker502/pvz/internal/particle"
)

func main() {
	fmt.Println("=== 测试假设：SystemAlpha 使用绝对时间 ===\n")
	
	// ZombieHead 配置
	systemDuration := 180.0 / 100.0 // 1.8秒
	
	// 假设1：95 是百分比（95%）
	fadeTime1 := 0.95 * systemDuration
	fmt.Printf("假设1：95 = 95%% 的生命周期\n")
	fmt.Printf("  淡出开始时间: %.2f秒 (%.1f%%)\n", fadeTime1, fadeTime1/systemDuration*100)
	fmt.Printf("  效果：僵尸头在1.71秒时完全消失\n\n")
	
	// 假设2：95 是绝对时间（95厘秒 = 0.95秒）
	fadeTime2 := 95.0 / 100.0
	fmt.Printf("假设2：95 = 95厘秒 = 0.95秒\n")
	fmt.Printf("  淡出开始时间: %.2f秒 (%.1f%%)\n", fadeTime2, fadeTime2/systemDuration*100)
	fmt.Printf("  效果：僵尸头在0.95秒时完全消失\n\n")
	
	// 对比 ZombieArm（SystemDuration=60, SystemAlpha=1,90 0）
	armDuration := 60.0 / 100.0 // 0.6秒
	armFade1 := 0.90 * armDuration
	armFade2 := 90.0 / 100.0
	
	fmt.Println("对比 ZombieArm (SystemDuration=60):")
	fmt.Printf("  假设1 (百分比): 在%.2f秒 (90%%) 消失\n", armFade1)
	fmt.Printf("  假设2 (绝对时间): 在%.2f秒 (%.1f%%) 消失\n", armFade2, armFade2/armDuration*100)
	fmt.Println()
	
	// 分析
	fmt.Println("=== 分析 ===")
	fmt.Println("如果是假设2（绝对时间）：")
	fmt.Println("  - ZombieHead 在0.95秒完全消失（生命周期52.8%）")
	fmt.Println("  - ZombieArm 在0.90秒完全消失（生命周期150%，已经超出！）")
	fmt.Println("  ❌ 不合理！ZombieArm 会在生命周期结束前就消失")
	fmt.Println()
	fmt.Println("如果是假设1（百分比）：")
	fmt.Println("  - ZombieHead 在1.71秒完全消失（生命周期95%）")
	fmt.Println("  - ZombieArm 在0.54秒完全消失（生命周期90%）")
	fmt.Println("  ✅ 合理！两个粒子都在各自生命周期的90-95%消失")
	
	// 测试解析结果
	fmt.Println("\n=== 实际解析验证 ===")
	_, _, keys, _ := particlePkg.ParseValue("1,95 0")
	if len(keys) >= 2 {
		fmt.Printf("解析结果: time=%.4f, value=%.2f\n", keys[1].Time, keys[1].Value)
		if keys[1].Time == 0.95 {
			fmt.Println("✅ 当前实现：95 被解析为 0.95 (百分比)")
		}
	}
}
