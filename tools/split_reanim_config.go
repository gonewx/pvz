package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GameReanimConfig 游戏 Reanim 配置文件的顶层结构
type GameReanimConfig struct {
	Global     GlobalConfig          `yaml:"global"`
	Animations []AnimationUnitConfig `yaml:"animations"`
}

// GlobalConfig 全局配置
type GlobalConfig struct {
	Playback interface{} `yaml:"playback"`
	Grid     interface{} `yaml:"grid,omitempty"`
	Window   interface{} `yaml:"window,omitempty"`
}

// AnimationUnitConfig 动画单元配置
type AnimationUnitConfig struct {
	ID                  string                 `yaml:"id"`
	Name                string                 `yaml:"name"`
	ReanimFile          string                 `yaml:"reanim_file"`
	DefaultAnimation    string                 `yaml:"default_animation"`
	Scale               interface{}            `yaml:"scale,omitempty"`
	Images              map[string]string      `yaml:"images"`
	AvailableAnimations []AnimationInfo        `yaml:"available_animations"`
	AnimationCombos     []AnimationComboConfig `yaml:"animation_combos,omitempty"`
}

// AnimationInfo 动画信息
type AnimationInfo struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
}

// AnimationComboConfig 动画组合配置
type AnimationComboConfig struct {
	Name            string            `yaml:"name"`
	DisplayName     string            `yaml:"display_name,omitempty"`
	Animations      []string          `yaml:"animations"`
	BindingStrategy string            `yaml:"binding_strategy,omitempty"`
	ManualBindings  map[string]string `yaml:"manual_bindings,omitempty"`
	ParentTracks    map[string]string `yaml:"parent_tracks,omitempty"`
	HiddenTracks    []string          `yaml:"hidden_tracks,omitempty"`
}

func main() {
	// 1. 读取配置文件
	configPath := "data/reanim_config.yaml"
	fmt.Printf("📖 读取配置文件: %s\n", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Printf("❌ 无法读取配置文件: %v\n", err)
		os.Exit(1)
	}

	// 2. 解析 YAML
	var config GameReanimConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ 无法解析配置文件: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 配置文件解析成功\n")
	fmt.Printf("   - 动画单元数量: %d\n", len(config.Animations))

	// 3. 创建输出目录
	outputDir := "data/reanim_config"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Printf("❌ 无法创建目录 %s: %v\n", outputDir, err)
		os.Exit(1)
	}
	fmt.Printf("✅ 创建输出目录: %s\n", outputDir)

	// 4. 拆分动画单元配置
	fmt.Printf("\n🔧 拆分动画单元配置...\n")
	successCount := 0
	errorCount := 0

	for _, unit := range config.Animations {
		if unit.ID == "" {
			fmt.Printf("⚠️  跳过缺少 ID 的动画单元\n")
			errorCount++
			continue
		}

		// 构造输出文件名
		filename := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", unit.ID))

		// 序列化为 YAML
		yamlData, err := yaml.Marshal(unit)
		if err != nil {
			fmt.Printf("❌ 无法序列化动画单元 '%s': %v\n", unit.ID, err)
			errorCount++
			continue
		}

		// 写入文件
		if err := os.WriteFile(filename, yamlData, 0644); err != nil {
			fmt.Printf("❌ ��法写入文件 %s: %v\n", filename, err)
			errorCount++
			continue
		}

		successCount++
		if successCount%20 == 0 {
			fmt.Printf("   ... %d/%d 已处理\n", successCount, len(config.Animations))
		}
	}

	fmt.Printf("\n✅ 动画单元配置拆分完成:\n")
	fmt.Printf("   - 成功: %d\n", successCount)
	fmt.Printf("   - 失败: %d\n", errorCount)

	// 5. 保存全局配置到原文件
	fmt.Printf("\n🔧 更新全局配置文件...\n")

	// 创建仅包含全局配置的结构
	globalOnlyConfig := struct {
		Global GlobalConfig `yaml:"global"`
	}{
		Global: config.Global,
	}

	globalData, err := yaml.Marshal(globalOnlyConfig)
	if err != nil {
		fmt.Printf("❌ 无法序列化全局配置: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(configPath, globalData, 0644); err != nil {
		fmt.Printf("❌ 无法写入全局配置文件: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 全局配置已更新到: %s\n", configPath)

	// 6. 验证输出
	fmt.Printf("\n🔍 验证拆分结果...\n")
	files, err := filepath.Glob(filepath.Join(outputDir, "*.yaml"))
	if err != nil {
		fmt.Printf("❌ 无法扫描输出目录: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ 输出目录包含 %d 个 YAML 文件\n", len(files))

	// 抽查几个文件
	fmt.Printf("\n📝 抽查文件内容:\n")
	sampleIDs := []string{"peashooter", "sunflower", "zombie"}
	for _, id := range sampleIDs {
		filename := filepath.Join(outputDir, fmt.Sprintf("%s.yaml", id))
		if _, err := os.Stat(filename); err == nil {
			fmt.Printf("   ✅ %s.yaml 存在\n", id)
		} else {
			fmt.Printf("   ⚠️  %s.yaml 不存在\n", id)
		}
	}

	fmt.Printf("\n🎉 配置拆分完成！\n")
	fmt.Printf("\n📂 输出结构:\n")
	fmt.Printf("   data/reanim_config.yaml         (仅全局配置)\n")
	fmt.Printf("   data/reanim_config/*.yaml       (%d 个动画单元配置)\n", len(files))
}
