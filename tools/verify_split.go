package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Printf("🔍 验证拆分后的配置文件...\n\n")

	// 1. 验证所有 YAML 文件格式
	files, err := filepath.Glob("data/reanim_config/*.yaml")
	if err != nil {
		fmt.Printf("❌ 无法扫描目录: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("📂 找到 %d 个 YAML 文件\n\n", len(files))

	errorCount := 0
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("❌ 无法读取文件 %s: %v\n", file, err)
			errorCount++
			continue
		}

		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err != nil {
			fmt.Printf("❌ YAML 格式错误: %s - %v\n", filepath.Base(file), err)
			errorCount++
		}
	}

	if errorCount == 0 {
		fmt.Printf("✅ 所有文件 YAML 格式正确\n\n")
	} else {
		fmt.Printf("❌ 有 %d 个文件格式错误\n\n", errorCount)
		os.Exit(1)
	}

	// 2. 抽查关键文件
	fmt.Printf("📝 抽查关键文件内容...\n")
	samples := []string{
		"peashooter", "sunflower", "zombie",
		"zombie_conehead", "zombie_buckethead",
		"cherry_bomb", "wallnut", "potatomine",
	}

	type AnimationUnit struct {
		ID               string `yaml:"id"`
		Name             string `yaml:"name"`
		ReanimFile       string `yaml:"reanim_file"`
		DefaultAnimation string `yaml:"default_animation"`
	}

	for _, id := range samples {
		filename := filepath.Join("data/reanim_config", fmt.Sprintf("%s.yaml", id))
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Printf("   ⚠️  %s.yaml 不存在\n", id)
			continue
		}

		var unit AnimationUnit
		if err := yaml.Unmarshal(data, &unit); err != nil {
			fmt.Printf("   ❌ %s.yaml 解析失败: %v\n", id, err)
			continue
		}

		// 验证必要字段
		if unit.ID == "" || unit.Name == "" || unit.ReanimFile == "" {
			fmt.Printf("   ❌ %s.yaml 缺少必要字段\n", id)
			continue
		}

		fmt.Printf("   ✅ %s.yaml - ID=%s, Name=%s\n", id, unit.ID, unit.Name)
	}

	// 3. 验证全局配置
	fmt.Printf("\n📝 验证全局配置文件...\n")
	globalData, err := os.ReadFile("data/reanim_config.yaml")
	if err != nil {
		fmt.Printf("❌ 无法读取全局配置文件: %v\n", err)
		os.Exit(1)
	}

	type GlobalConfigFile struct {
		Global struct {
			Playback struct {
				TPS int `yaml:"tps"`
				FPS int `yaml:"fps"`
			} `yaml:"playback"`
		} `yaml:"global"`
		Animations interface{} `yaml:"animations,omitempty"`
	}

	var globalConfig GlobalConfigFile
	if err := yaml.Unmarshal(globalData, &globalConfig); err != nil {
		fmt.Printf("❌ 全局配置解析失败: %v\n", err)
		os.Exit(1)
	}

	if globalConfig.Global.Playback.TPS == 0 || globalConfig.Global.Playback.FPS == 0 {
		fmt.Printf("❌ 全局配置缺少必要字段\n")
		os.Exit(1)
	}

	if globalConfig.Animations != nil {
		fmt.Printf("⚠️  全局配置文件仍包含 animations 字段（应该已删除）\n")
	}

	fmt.Printf("✅ 全局配置正确: TPS=%d, FPS=%d\n",
		globalConfig.Global.Playback.TPS,
		globalConfig.Global.Playback.FPS)

	fmt.Printf("\n🎉 所有验证通过！\n")
}
