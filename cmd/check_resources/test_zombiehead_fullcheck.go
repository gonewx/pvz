package main

import (
	"fmt"
	"log"
	particlePkg "github.com/decker502/pvz/internal/particle"
)

func main() {
	config, err := particlePkg.ParseParticleXML("assets/effect/particles/ZombieHead.xml")
	if err != nil {
		log.Fatal(err)
	}

	emitter := config.Emitters[0]
	
	fmt.Println("=== ZombieHead 完整配置检查 ===\n")
	
	// 所有可能影响透明度的字段
	fmt.Println("📊 透明度相关配置：")
	fmt.Printf("  ParticleAlpha: '%s'\n", emitter.ParticleAlpha)
	fmt.Printf("  SystemAlpha: '%s'\n", emitter.SystemAlpha)
	fmt.Println()
	
	// 生命周期配置
	fmt.Println("⏱️  生命周期配置：")
	fmt.Printf("  ParticleDuration: '%s' → %.2f秒\n", emitter.ParticleDuration, 180.0/100.0)
	fmt.Printf("  SystemDuration: '%s' → %.2f秒\n", emitter.SystemDuration, 180.0/100.0)
	fmt.Println()
	
	// 发射配置
	fmt.Println("🚀 发射配置：")
	fmt.Printf("  SpawnMinActive: '%s'\n", emitter.SpawnMinActive)
	fmt.Printf("  SpawnMaxActive: '%s'\n", emitter.SpawnMaxActive)
	fmt.Printf("  SpawnMaxLaunched: '%s'\n", emitter.SpawnMaxLaunched)
	fmt.Printf("  SpawnRate: '%s'\n", emitter.SpawnRate)
	fmt.Println()
	
	// 渲染配置
	fmt.Println("🎨 渲染配置：")
	fmt.Printf("  Image: '%s'\n", emitter.Image)
	fmt.Printf("  ParticleScale: '%s'\n", emitter.ParticleScale)
	fmt.Printf("  Additive: '%s'\n", emitter.Additive)
	fmt.Println()
	
	// 解析 SystemAlpha
	_, _, sysKeys, _ := particlePkg.ParseValue(emitter.SystemAlpha)
	if len(sysKeys) > 0 {
		fmt.Println("📈 SystemAlpha 曲线：")
		for t := 0.0; t <= 1.0; t += 0.05 {
			alpha := particlePkg.EvaluateKeyframes(sysKeys, t, "")
			bar := ""
			for i := 0; i < int(alpha*50); i++ {
				bar += "█"
			}
			fmt.Printf("  %.2f (%.1f%%) [%-50s] %.3f\n", t*1.8, t*100, bar, alpha)
		}
	}
}
