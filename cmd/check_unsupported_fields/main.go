package main

import (
	"encoding/xml"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 支持的字段列表（基于 internal/particle/types.go 中的 EmitterConfig 结构体）
var supportedFields = map[string]bool{
	// 根标签（不是字段，但需要排除）
	"Emitter": true,

	// Emitter 标签
	"Name": true,

	// Spawn properties
	"SpawnMinActive":   true,
	"SpawnMaxActive":   true,
	"SpawnMaxLaunched": true,
	"SpawnRate":        true,

	// Particle properties
	"ParticleDuration":    true,
	"ParticleAlpha":       true,
	"ParticleScale":       true,
	"ParticleSpinAngle":   true,
	"ParticleSpinSpeed":   true,
	"ParticleRed":         true,
	"ParticleGreen":       true,
	"ParticleBlue":        true,
	"ParticleBrightness":  true,
	"ParticleLoops":       true,
	"ParticleStretch":     true,
	"ParticlesDontFollow": true,

	// Launch properties
	"LaunchSpeed":      true,
	"LaunchAngle":      true,
	"AlignLaunchSpin":  true,
	"RandomLaunchSpin": true,
	"RandomStartTime":  true,

	// Emitter properties
	"EmitterBoxX":    true,
	"EmitterBoxY":    true,
	"EmitterRadius":  true,
	"EmitterType":    true,
	"EmitterSkewX":   true,
	"EmitterOffsetX": true,
	"EmitterOffsetY": true,

	// System properties
	"SystemDuration": true,
	"SystemAlpha":    true,
	"SystemLoops":    true,
	"SystemField":    true,

	// Image properties
	"Image":         true,
	"ImageFrames":   true,
	"ImageRow":      true,
	"ImageCol":      true,
	"Animated":      true,
	"AnimationRate": true,

	// Rendering properties
	"Additive":     true,
	"FullScreen":   true,
	"HardwareOnly": true,
	"ClipTop":      true,

	// Cross-fade and lifecycle
	"CrossFadeDuration": true,
	"DieIfOverloaded":   true,

	// Collision properties
	"CollisionReflect": true,
	"CollisionSpin":    true,

	// Field 子标签
	"Field":     true,
	"FieldType": true,
	"X":         true,
	"Y":         true,
}

// GenericEmitter 用于解析任意字段的 Emitter
type GenericEmitter struct {
	XMLName xml.Name
	Fields  map[string]string `xml:",any"`
}

// GenericParticleConfig 用于解析任意字段的粒子配置
type GenericParticleConfig struct {
	XMLName  xml.Name
	Emitters []xml.Token `xml:",any"`
}

// FieldUsage 记录字段的使用情况
type FieldUsage struct {
	FieldName string
	Files     []string
	Count     int
}

func main() {
	particleDir := "assets/effect/particles"

	// 检查目录是否存在
	if _, err := os.Stat(particleDir); os.IsNotExist(err) {
		fmt.Printf("错误: 目录不存在: %s\n", particleDir)
		os.Exit(1)
	}

	// 收集所有不支持的字段
	unsupportedFields := make(map[string]*FieldUsage)

	// 遍历所有 XML 文件
	err := filepath.WalkDir(particleDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理 XML 文件
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".xml") {
			return nil
		}

		// 读取文件
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("警告: 无法读取文件 %s: %v\n", path, err)
			return nil
		}

		// 解析 XML 获取所有字段
		fields := extractAllFields(data)

		// 检查不支持的字段
		relPath, _ := filepath.Rel(particleDir, path)
		for _, field := range fields {
			if !supportedFields[field] {
				if usage, exists := unsupportedFields[field]; exists {
					usage.Files = append(usage.Files, relPath)
					usage.Count++
				} else {
					unsupportedFields[field] = &FieldUsage{
						FieldName: field,
						Files:     []string{relPath},
						Count:     1,
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("错误: 遍历目录失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	if len(unsupportedFields) == 0 {
		fmt.Println("✅ 所有粒子配置文件中的字段都已支持！")
		return
	}

	fmt.Printf("❌ 发现 %d 个不支持的字段:\n\n", len(unsupportedFields))

	// 按使用次数排序
	usages := make([]*FieldUsage, 0, len(unsupportedFields))
	for _, usage := range unsupportedFields {
		usages = append(usages, usage)
	}
	sort.Slice(usages, func(i, j int) bool {
		return usages[i].Count > usages[j].Count
	})

	// 输出详细信息
	for _, usage := range usages {
		fmt.Printf("字段: %s\n", usage.FieldName)
		fmt.Printf("  使用次数: %d\n", usage.Count)
		fmt.Printf("  出现文件:\n")
		for _, file := range usage.Files {
			fmt.Printf("    - %s\n", file)
		}
		fmt.Println()
	}

	// 输出汇总
	fmt.Println("=== 汇总 ===")
	fmt.Printf("不支持的字段列表 (按使用频率排序):\n")
	for i, usage := range usages {
		fmt.Printf("%d. %s (使用 %d 次)\n", i+1, usage.FieldName, usage.Count)
	}
}

// extractAllFields 从 XML 数据中提取所有字段名
func extractAllFields(data []byte) []string {
	fields := make(map[string]bool)
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			// 记录元素名
			fields[t.Name.Local] = true
		}
	}

	// 转换为切片
	result := make([]string, 0, len(fields))
	for field := range fields {
		result = append(result, field)
	}
	return result
}
