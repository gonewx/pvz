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
	fmt.Println("=== ZombieHead.xml 解析结果 ===\n")

	// 测试关键字段的解析
	fmt.Println("1. ParticleDuration (粒子生命周期):")
	fmt.Printf("   原始值: %s\n", emitter.ParticleDuration)
	dMin, dMax, dKeys, dInterp := particlePkg.ParseValue(emitter.ParticleDuration)
	fmt.Printf("   解析: min=%.2f, max=%.2f, keyframes=%v, interp=%s\n", dMin, dMax, dKeys, dInterp)
	fmt.Printf("   实际生命周期: %.2f秒 (180厘秒 = 1.8秒)\n\n", dMin/100.0)

	fmt.Println("2. SystemDuration (系统持续时间):")
	fmt.Printf("   原始值: %s\n", emitter.SystemDuration)
	sMin, sMax, sKeys, sInterp := particlePkg.ParseValue(emitter.SystemDuration)
	fmt.Printf("   解析: min=%.2f, max=%.2f, keyframes=%v, interp=%s\n", sMin, sMax, sKeys, sInterp)
	fmt.Printf("   实际持续时间: %.2f秒 (180厘秒 = 1.8秒)\n\n", sMin/100.0)

	fmt.Println("3. SystemAlpha (系统透明度):")
	fmt.Printf("   原始值: %s\n", emitter.SystemAlpha)
	aMin, aMax, aKeys, aInterp := particlePkg.ParseValue(emitter.SystemAlpha)
	fmt.Printf("   解析: min=%.2f, max=%.2f, interp=%s\n", aMin, aMax, aInterp)
	fmt.Printf("   关键帧数量: %d\n", len(aKeys))
	if len(aKeys) > 0 {
		fmt.Println("   关键帧详情:")
		for i, kf := range aKeys {
			fmt.Printf("     [%d] time=%.4f (%.1f%%), value=%.4f\n", i, kf.Time, kf.Time*100, kf.Value)
		}
	}
	fmt.Println()

	fmt.Println("4. CollisionReflect (碰撞反弹):")
	fmt.Printf("   原始值: %s\n", emitter.CollisionReflect)
	rMin, rMax, rKeys, rInterp := particlePkg.ParseValue(emitter.CollisionReflect)
	fmt.Printf("   解析: min=%.4f, max=%.4f, interp=%s\n", rMin, rMax, rInterp)
	fmt.Printf("   关键帧数量: %d\n", len(rKeys))
	if len(rKeys) > 0 {
		fmt.Println("   关键帧详情:")
		for i, kf := range rKeys {
			fmt.Printf("     [%d] time=%.4f (%.1f%%), value=%.4f\n", i, kf.Time, kf.Time*100, kf.Value)
		}
	}
	fmt.Println()

	fmt.Println("5. CollisionSpin (碰撞旋转):")
	fmt.Printf("   原始值: %s\n", emitter.CollisionSpin)
	spinMin, spinMax, spinKeys, spinInterp := particlePkg.ParseValue(emitter.CollisionSpin)
	fmt.Printf("   解析: min=%.2f, max=%.2f, interp=%s\n", spinMin, spinMax, spinInterp)
	fmt.Printf("   关键帧数量: %d\n", len(spinKeys))
	if len(spinKeys) > 0 {
		fmt.Println("   关键帧详情:")
		for i, kf := range spinKeys {
			fmt.Printf("     [%d] time=%.4f (%.1f%%), value=%.4f\n", i, kf.Time, kf.Time*100, kf.Value)
		}
	}
	fmt.Println()

	// 模拟 SystemAlpha 在不同时间点的值
	if len(aKeys) > 0 {
		fmt.Println("6. SystemAlpha 时间曲线模拟:")
		fmt.Println("   时间点 -> Alpha值")
		for t := 0.0; t <= 1.0; t += 0.1 {
			alpha := particlePkg.EvaluateKeyframes(aKeys, t, aInterp)
			fmt.Printf("   t=%.2f (%.1f%%) -> alpha=%.4f\n", t, t*100, alpha)
		}
	}
}
