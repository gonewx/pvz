package main

import (
	"fmt"
	"log"

	particlePkg "github.com/decker502/pvz/internal/particle"
)

func main() {
	// 解析 ZombieHead.xml
	config, err := particlePkg.ParseParticleXML("assets/effect/particles/ZombieHead.xml")
	if err != nil {
		log.Fatal(err)
	}

	if len(config.Emitters) == 0 {
		log.Fatal("No emitters found")
	}

	emitter := config.Emitters[0]
	
	fmt.Println("=== ZombieHead 透明度分析 ===\n")
	
	// 1. 粒子自身的 Alpha
	fmt.Println("1. ParticleAlpha (粒子自身透明度):")
	fmt.Printf("   XML值: '%s'\n", emitter.ParticleAlpha)
	alphaMin, alphaMax, alphaKeys, alphaInterp := particlePkg.ParseValue(emitter.ParticleAlpha)
	fmt.Printf("   解析: min=%.2f, max=%.2f, keyframes数=%d, interp=%s\n", alphaMin, alphaMax, len(alphaKeys), alphaInterp)
	if len(alphaKeys) == 0 && alphaMin == 0 && alphaMax == 0 {
		fmt.Printf("   ⚠️  未配置 ParticleAlpha，默认值=1.0（完全不透明）\n")
	}
	fmt.Println()
	
	// 2. 系统级 Alpha
	fmt.Println("2. SystemAlpha (系统级透明度):")
	fmt.Printf("   XML值: '%s'\n", emitter.SystemAlpha)
	sysMin, sysMax, sysKeys, sysInterp := particlePkg.ParseValue(emitter.SystemAlpha)
	fmt.Printf("   解析: min=%.2f, max=%.2f, keyframes数=%d, interp=%s\n", sysMin, sysMax, len(sysKeys), sysInterp)
	if len(sysKeys) > 0 {
		fmt.Println("   关键帧:")
		for i, kf := range sysKeys {
			fmt.Printf("     [%d] time=%.1f%%, value=%.3f\n", i, kf.Time*100, kf.Value)
		}
	}
	fmt.Println()
	
	// 3. 模拟粒子在整个生命周期的透明度
	fmt.Println("3. 透明度时间曲线模拟 (1.8秒生命周期):")
	fmt.Println("   时间(秒) | 粒子Alpha | 系统Alpha | 最终Alpha")
	fmt.Println("   -------- | --------- | --------- | ---------")
	
	particleDuration := 1.8 // 180厘秒 = 1.8秒
	systemDuration := 1.8   // 180厘秒 = 1.8秒
	
	for t := 0.0; t <= particleDuration; t += 0.2 {
		// 粒子自身 Alpha (如果有关键帧)
		var particleAlpha float64
		if len(alphaKeys) > 0 {
			particleT := t / particleDuration
			particleAlpha = particlePkg.EvaluateKeyframes(alphaKeys, particleT, alphaInterp)
		} else if alphaMin > 0 || alphaMax > 0 {
			particleAlpha = particlePkg.RandomInRange(alphaMin, alphaMax)
		} else {
			particleAlpha = 1.0 // 默认完全不透明
		}
		
		// 系统 Alpha
		var systemAlpha float64
		if len(sysKeys) > 0 {
			systemT := t / systemDuration
			systemAlpha = particlePkg.EvaluateKeyframes(sysKeys, systemT, sysInterp)
		} else {
			systemAlpha = 1.0 // 默认完全不透明
		}
		
		// 最终 Alpha = 粒子Alpha * 系统Alpha
		finalAlpha := particleAlpha * systemAlpha
		
		fmt.Printf("   %.2f秒   | %.3f     | %.3f     | %.3f\n", 
			t, particleAlpha, systemAlpha, finalAlpha)
	}
	fmt.Println()
	
	// 4. 关键时间点分析
	fmt.Println("4. 关键时间点分析:")
	
	// 95% 时间点（SystemAlpha 开始消失）
	timeAt95 := systemDuration * 0.95
	fmt.Printf("   95%%时间点 (%.2f秒):\n", timeAt95)
	if len(sysKeys) > 0 {
		alpha95 := particlePkg.EvaluateKeyframes(sysKeys, 0.95, sysInterp)
		fmt.Printf("     SystemAlpha = %.3f\n", alpha95)
		fmt.Printf("     最终Alpha = 1.0 * %.3f = %.3f\n", alpha95, alpha95)
	}
	fmt.Println()
	
	// 100% 时间点（粒子结束）
	fmt.Printf("   100%%时间点 (%.2f秒 - 粒子消失):\n", systemDuration)
	if len(sysKeys) > 0 {
		alpha100 := particlePkg.EvaluateKeyframes(sysKeys, 1.0, sysInterp)
		fmt.Printf("     SystemAlpha = %.3f\n", alpha100)
		fmt.Printf("     最终Alpha = 1.0 * %.3f = %.3f\n", alpha100, alpha100)
	}
	fmt.Println()
	
	fmt.Println("=== 诊断结论 ===")
	fmt.Println("ZombieHead 粒子的透明度特点:")
	fmt.Println("1. 粒子自身没有配置 ParticleAlpha，默认保持1.0（完全不透明）")
	fmt.Println("2. SystemAlpha 配置为 '1,95 0'，表示：")
	fmt.Println("   - 从 0% 到 95% 时间，系统透明度从 1.0 线性衰减到 0")
	fmt.Println("   - 从 95% 到 100% 时间，系统透明度保持 0（完全透明）")
	fmt.Println("3. 最终渲染透明度 = 粒子Alpha(1.0) * SystemAlpha(1.0→0)")
	fmt.Println("4. 在 1.71秒 (95%) 时，头部应该完全消失")
	fmt.Println("5. 在 1.71秒 到 1.8秒 之间，头部应该完全不可见")
}
