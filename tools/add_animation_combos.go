package main

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// 配置结构（简化版）
type Config struct {
	Global     GlobalConfig          `yaml:"global"`
	Animations []AnimationUnitConfig `yaml:"animations"`
}

type GlobalConfig struct {
	Window   interface{} `yaml:"window"`
	Grid     interface{} `yaml:"grid"`
	Playback interface{} `yaml:"playback"`
}

type AnimationUnitConfig struct {
	ID                  string                   `yaml:"id"`
	Name                string                   `yaml:"name"`
	ReanimFile          string                   `yaml:"reanim_file"`
	DefaultAnimation    string                   `yaml:"default_animation"`
	Scale               float64                  `yaml:"scale,omitempty"`
	Images              map[string]string        `yaml:"images"`
	AvailableAnimations []interface{}            `yaml:"available_animations"`
	AnimationCombos     []AnimationComboConfig   `yaml:"animation_combos,omitempty"`
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

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("无法读取配置文件: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("无法解析配置文件: %v", err)
	}

	// 为关键实体添加 animation_combos
	for i := range config.Animations {
		unit := &config.Animations[i]

		switch unit.ID {
		case "peashooter", "peashootersingle":
			// 豌豆射手
			if len(unit.AnimationCombos) == 0 {
				unit.AnimationCombos = []AnimationComboConfig{
					{
						Name:            "idle",
						DisplayName:     "待机",
						Animations:      []string{"anim_full_idle"},
						BindingStrategy: "auto",
					},
					{
						Name:            "attack",
						DisplayName:     "攻击",
						Animations:      []string{"anim_shooting", "anim_head_idle"},
						BindingStrategy: "auto",
						ParentTracks: map[string]string{
							"anim_face": "anim_stem",
						},
					},
				}
				log.Printf("✓ 为 %s 添加 animation_combos", unit.ID)
			}

		case "sunflower":
			// 向日葵
			if len(unit.AnimationCombos) == 0 {
				unit.AnimationCombos = []AnimationComboConfig{
					{
						Name:            "idle",
						DisplayName:     "待机",
						Animations:      []string{"anim_idle"},
						BindingStrategy: "auto",
					},
				}
				log.Printf("✓ 为 %s 添加 animation_combos", unit.ID)
			}

		case "zombie":
			// 僵尸
			if len(unit.AnimationCombos) == 0 {
				unit.AnimationCombos = []AnimationComboConfig{
					{
						Name:            "idle",
						DisplayName:     "待机",
						Animations:      []string{"anim_idle"},
						BindingStrategy: "auto",
					},
					{
						Name:            "walk",
						DisplayName:     "行走",
						Animations:      []string{"anim_walk"},
						BindingStrategy: "auto",
					},
					{
						Name:            "eat",
						DisplayName:     "吃植物",
						Animations:      []string{"anim_eat"},
						BindingStrategy: "auto",
					},
				}
				log.Printf("✓ 为 %s 添加 animation_combos", unit.ID)
			}

		case "wallnut":
			// 坚果
			if len(unit.AnimationCombos) == 0 {
				unit.AnimationCombos = []AnimationComboConfig{
					{
						Name:            "idle",
						DisplayName:     "待机",
						Animations:      []string{"anim_idle"},
						BindingStrategy: "auto",
					},
				}
				log.Printf("✓ 为 %s 添加 animation_combos", unit.ID)
			}

		case "cherrybomb":
			// 樱桃炸弹
			if len(unit.AnimationCombos) == 0 {
				unit.AnimationCombos = []AnimationComboConfig{
					{
						Name:            "idle",
						DisplayName:     "待机",
						Animations:      []string{"anim_idle"},
						BindingStrategy: "auto",
					},
					{
						Name:            "explode",
						DisplayName:     "爆炸",
						Animations:      []string{"anim_explode"},
						BindingStrategy: "auto",
					},
				}
				log.Printf("✓ 为 %s 添加 animation_combos", unit.ID)
			}

		case "lawnmower":
			// 割草机
			if len(unit.AnimationCombos) == 0 {
				unit.AnimationCombos = []AnimationComboConfig{
					{
						Name:            "idle",
						DisplayName:     "待机",
						Animations:      []string{"anim_normal"},
						BindingStrategy: "auto",
					},
				}
				log.Printf("✓ 为 %s 添加 animation_combos", unit.ID)
			}
		}
	}

	// 写回配置文件
	output, err := yaml.Marshal(&config)
	if err != nil {
		log.Fatalf("无法序列化配置: %v", err)
	}

	// 备份原文件
	backupPath := configPath + ".bak"
	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		log.Fatalf("无法创建备份: %v", err)
	}
	log.Printf("✓ 已备份原配置到 %s", backupPath)

	// 写入新配置
	if err := os.WriteFile(configPath, output, 0644); err != nil {
		log.Fatalf("无法写入配置文件: %v", err)
	}

	fmt.Println("\n✅ 配置文件更新成功!")
	fmt.Printf("   - 配置文件: %s\n", configPath)
	fmt.Printf("   - 备份文件: %s\n", backupPath)
}
