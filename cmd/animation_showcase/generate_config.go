// cmd/animation_showcase/generate_config.go
// 辅助工具：扫描 reanim 文件并生成配置模板
//
// 用法：
//   go run cmd/animation_showcase/generate_config.go > config_generated.yaml
//
// 注意：这是一个独立的工具，不要和 main.go 一起编译

//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/decker502/pvz/internal/reanim"
)

func generateConfig() {
	// 扫描所有 reanim 文件
	reanimDir := "assets/effect/reanim"
	files, err := filepath.Glob(filepath.Join(reanimDir, "*.reanim"))
	if err != nil {
		log.Fatalf("扫描文件失败: %v", err)
	}

	fmt.Println("# 动画展示系统配置文件（自动生成）")
	fmt.Println("# 注意：图片映射需要手工配置")
	fmt.Println()

	// 全局配置
	fmt.Println("global:")
	fmt.Println("  window:")
	fmt.Println("    width: 1600")
	fmt.Println("    height: 900")
	fmt.Println("    title: \"PVZ3 Animation Showcase\"")
	fmt.Println()
	fmt.Println("  grid:")
	fmt.Println("    columns: 6")
	fmt.Println("    cell_width: 250")
	fmt.Println("    cell_height: 250")
	fmt.Println("    padding: 10")
	fmt.Println("    scroll_speed: 30")
	fmt.Println()
	fmt.Println("  playback:")
	fmt.Println("    fps: 12")
	fmt.Println("    scale: 1.0")
	fmt.Println()

	// 动画列表
	fmt.Println("animations:")

	for _, file := range files {
		// 加载 reanim 文件
		reanimXML, err := reanim.ParseReanimFile(file)
		if err != nil {
			log.Printf("警告: 无法加载 %s: %v", file, err)
			continue
		}

		baseName := filepath.Base(file)
		nameWithoutExt := strings.TrimSuffix(baseName, ".reanim")
		id := strings.ToLower(nameWithoutExt)

		fmt.Printf("  # %s\n", nameWithoutExt)
		fmt.Printf("  - id: \"%s\"\n", id)
		fmt.Printf("    name: \"%s\"\n", nameWithoutExt)
		fmt.Printf("    reanim_file: \"%s\"\n", file)

		// 查找可用的动画
		animationTracks := []string{}
		for _, track := range reanimXML.Tracks {
			if strings.HasPrefix(track.Name, "anim_") {
				animationTracks = append(animationTracks, track.Name)
			}
		}

		// 默认动画
		defaultAnim := "anim_idle"
		if len(animationTracks) > 0 {
			defaultAnim = animationTracks[0]
		}
		fmt.Printf("    default_animation: \"%s\"\n", defaultAnim)
		fmt.Printf("    scale: 1.0\n")
		fmt.Println()

		// 图片映射（需要手工配置）
		fmt.Println("    images:")
		imageRefs := make(map[string]bool)
		for _, track := range reanimXML.Tracks {
			for _, frame := range track.Frames {
				if frame.ImagePath != "" {
					imageRefs[frame.ImagePath] = true
				}
			}
		}
		for ref := range imageRefs {
			fmt.Printf("      %s: \"assets/reanim/FIXME_%s.png\"  # TODO: 修正路径\n", ref, ref)
		}
		fmt.Println()

		// 可用动画列表
		if len(animationTracks) > 0 {
			fmt.Println("    available_animations:")
			for _, anim := range animationTracks {
				displayName := strings.TrimPrefix(anim, "anim_")
				fmt.Printf("      - name: \"%s\"\n", anim)
				fmt.Printf("        display_name: \"%s\"\n", displayName)
			}
			fmt.Println()
		}
	}
}

func main() {
	generateConfig()
}
