package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// 配置结构
type Config struct {
	Global     interface{}           `yaml:"global"`
	Animations []AnimationUnitConfig `yaml:"animations"`
}

type AnimationUnitConfig struct {
	ID               string                   `yaml:"id"`
	Name             string                   `yaml:"name"`
	ReanimFile       string                   `yaml:"reanim_file"`
	DefaultAnimation string                   `yaml:"default_animation"`
	Scale            interface{}              `yaml:"scale,omitempty"`
	Images           map[string]string        `yaml:"images"`
	AvailableAnims   []interface{}            `yaml:"available_animations"`
	AnimationCombos  []AnimationComboConfig   `yaml:"animation_combos,omitempty"`
}

type AnimationComboConfig struct {
	Name            string            `yaml:"name"`
	DisplayName     string            `yaml:"display_name"`
	Animations      []string          `yaml:"animations"`
	BindingStrategy string            `yaml:"binding_strategy"`
	ParentTracks    map[string]string `yaml:"parent_tracks,omitempty"`
	HiddenTracks    []string          `yaml:"hidden_tracks,omitempty"`
}

func main() {
	configPath := "data/reanim_config.yaml"

	// 读取配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("无法解析配置文件: %v", err)
	}

	// 修正 zombie 的默认动画
	for i := range config.Animations {
		unit := &config.Animations[i]
		if unit.ID == "zombie" && unit.DefaultAnimation != "anim_idle" {
			log.Printf("修正 zombie 默认动画: %s -> anim_idle", unit.DefaultAnimation)
			unit.DefaultAnimation = "anim_idle"
		}
	}

	// 写回配置
	output, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("无法序列化配置: %v", err)
	}

	if err := os.WriteFile(configPath, output, 0644); err != nil {
		log.Fatalf("无法写入配置文件: %v", err)
	}

	fmt.Println("✅ zombie 默认动画已修正为 anim_idle")
}
