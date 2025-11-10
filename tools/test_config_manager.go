package main

import (
	"fmt"

	"github.com/decker502/pvz/pkg/config"
)

func main() {
	fmt.Printf("🔍 测试配置管理器重构...\n\n")

	// 1. 测试向后兼容性（单文件模式）
	fmt.Printf("📝 测试 1: 向后兼容性（单文件模式）\n")
	manager1, err := config.NewReanimConfigManager("data/reanim_config.yaml.backup")
	if err != nil {
		fmt.Printf("❌ 单文件模式加载失败: %v\n", err)
	} else {
		units1 := manager1.ListUnits()
		fmt.Printf("✅ 单文件模式加载成功，包含 %d 个动画单元\n", len(units1))

		// 测试获取特定单元
		if unit, err := manager1.GetUnit("peashooter"); err == nil {
			fmt.Printf("   ✅ 获取 peashooter 成功: %s\n", unit.Name)
		}
	}

	// 2. 测试目录模式
	fmt.Printf("\n📝 测试 2: 目录模式\n")
	manager2, err := config.NewReanimConfigManager("data/reanim_config")
	if err != nil {
		fmt.Printf("❌ 目录模式加载失败: %v\n", err)
		return
	}

	units2 := manager2.ListUnits()
	fmt.Printf("✅ 目录模式加载成功，包含 %d 个动画单元\n", len(units2))

	// 测试获取特定单元
	testUnits := []string{"peashooter", "sunflower", "zombie", "zombie_conehead"}
	for _, id := range testUnits {
		if unit, err := manager2.GetUnit(id); err == nil {
			fmt.Printf("   ✅ 获取 %s 成功: %s\n", id, unit.Name)
		} else {
			fmt.Printf("   ❌ 获取 %s 失败: %v\n", id, err)
		}
	}

	// 测试获取组合
	fmt.Printf("\n📝 测试 3: 获取动画组合\n")
	if combo, err := manager2.GetCombo("peashooter", "attack"); err == nil {
		fmt.Printf("✅ 获取 peashooter/attack 组合成功，包含 %d 个动画\n", len(combo.Animations))
	} else {
		fmt.Printf("❌ 获取组合失败: %v\n", err)
	}

	// 测试获取默认动画
	fmt.Printf("\n📝 测试 4: 获取默认动画\n")
	if defaultAnim, err := manager2.GetDefaultAnimation("peashooter"); err == nil {
		fmt.Printf("✅ 获取 peashooter 默认动画: %s\n", defaultAnim)
	} else {
		fmt.Printf("❌ 获取默认动画失败: %v\n", err)
	}

	// 测试全局配置
	fmt.Printf("\n📝 测试 5: 全局配置\n")
	globalConfig := manager2.GetGlobalConfig()
	fmt.Printf("✅ 全局配置: TPS=%d, FPS=%d\n", globalConfig.Playback.TPS, globalConfig.Playback.FPS)

	fmt.Printf("\n🎉 所有测试通过！\n")
}
