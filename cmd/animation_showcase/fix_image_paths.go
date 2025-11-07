// cmd/animation_showcase/fix_image_paths.go
// 自动修复配置文件中的图片路径
//
// 用法：
//   go run cmd/animation_showcase/fix_image_paths.go config_all.yaml > config_fixed.yaml

//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("用法: go run fix_image_paths.go <config_file>")
	}

	configFile := os.Args[1]

	// 1. 扫描所有图片文件
	imageMap := scanImages("assets/reanim")
	log.Printf("找到 %d 个图片文件", len(imageMap))

	// 2. 读取并替换配置文件
	file, err := os.Open(configFile)
	if err != nil {
		log.Fatalf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	fixmePattern := regexp.MustCompile(`"assets/reanim/FIXME_([^"]+)\.png"`)

	for scanner.Scan() {
		line := scanner.Text()

		// 查找需要替换的行
		if strings.Contains(line, "FIXME_") {
			// 提取 IMAGE_REANIM_ 引用名
			matches := fixmePattern.FindStringSubmatch(line)
			if len(matches) >= 2 {
				imageRef := matches[1] // 例如: IMAGE_REANIM_PEASHOOTER_HEAD

				// 查找实际图片路径（先尝试直接匹配）
				actualPath := ""
				if path, found := imageMap[imageRef]; found {
					actualPath = path
				} else {
					// 尝试替换空格为下划线后匹配
					normalizedRef := strings.ReplaceAll(imageRef, " ", "_")
					if path, found := imageMap[normalizedRef]; found {
						actualPath = path
					}
				}

				if actualPath != "" {
					// 替换路径
					line = fixmePattern.ReplaceAllString(line, fmt.Sprintf(`"%s"`, actualPath))
					log.Printf("✓ 替换: %s -> %s", imageRef, actualPath)
				} else {
					log.Printf("⚠ 未找到图片: %s", imageRef)
				}
			}
		}

		fmt.Println(line)
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
}

// scanImages 扫描目录中的所有图片文件，建立 IMAGE_REANIM 引用到实际路径的映射
func scanImages(dir string) map[string]string {
	imageMap := make(map[string]string)

	// 支持 png 和 jpg 两种格式
	patterns := []string{"*.png", "*.jpg"}

	for _, pattern := range patterns {
		files, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			log.Printf("警告: 扫描目录失败: %v", err)
			continue
		}

		for _, file := range files {
			baseName := filepath.Base(file)
			ext := filepath.Ext(baseName)
			nameWithoutExt := strings.TrimSuffix(baseName, ext)

			// 生成可能的 IMAGE_REANIM 引用名
			// 例如: PeaShooter_Head.png -> IMAGE_REANIM_PEASHOOTER_HEAD
			ref := "IMAGE_REANIM_" + strings.ToUpper(strings.ReplaceAll(nameWithoutExt, " ", "_"))

			// 如果已经存在，优先使用 png 格式
			if existing, exists := imageMap[ref]; exists {
				if ext == ".png" && filepath.Ext(existing) == ".jpg" {
					imageMap[ref] = file
				}
			} else {
				imageMap[ref] = file
			}

			// 同时添加原始名称（保留空格）以支持特殊情况
			if strings.Contains(nameWithoutExt, " ") {
				refWithSpace := "IMAGE_REANIM_" + strings.ToUpper(nameWithoutExt)
				imageMap[refWithSpace] = file
			}
		}
	}

	// 扫描子目录（如 assets/images）
	subDirs := []string{
		"assets/images",
		"assets/images/reanim",
	}

	for _, subDir := range subDirs {
		for _, pattern := range patterns {
			files, err := filepath.Glob(filepath.Join(subDir, pattern))
			if err != nil {
				continue
			}

			for _, file := range files {
				baseName := filepath.Base(file)
				ext := filepath.Ext(baseName)
				nameWithoutExt := strings.TrimSuffix(baseName, ext)

				ref := "IMAGE_REANIM_" + strings.ToUpper(strings.ReplaceAll(nameWithoutExt, " ", "_"))

				// 如果主目录没有，使用子目录的
				if _, exists := imageMap[ref]; !exists {
					imageMap[ref] = file
				}
			}
		}
	}

	return imageMap
}
