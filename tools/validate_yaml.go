package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Global     interface{}              `yaml:"global"`
	Animations []map[string]interface{} `yaml:"animations"`
}

func main() {
	data, err := os.ReadFile("data/reanim_config.yaml")
	if err != nil {
		fmt.Printf("❌ 读取文件失败: %v\n", err)
		os.Exit(1)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		fmt.Printf("❌ YAML 解析失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ YAML 格式正确\n")
	fmt.Printf("✅ 动画单元数量: %d\n", len(config.Animations))

	missingID := 0
	for i, anim := range config.Animations {
		if _, ok := anim["id"]; !ok {
			fmt.Printf("❌ 第 %d 个单元缺少 id\n", i+1)
			missingID++
		}
	}

	if missingID == 0 {
		fmt.Printf("✅ 所有单元都有有效的 id 字段\n")
	} else {
		fmt.Printf("❌ 有 %d 个单元缺少 id 字段\n", missingID)
		os.Exit(1)
	}
}
