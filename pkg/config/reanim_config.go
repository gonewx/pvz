package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ReanimConfig Reanim 配置文件的顶层结构
// 用于定义 Reanim 动画的配置，包括动画列表和动画组合
type ReanimConfig struct {
	// ReanimFile Reanim 文件路径（相对于项目根目录）
	ReanimFile string `yaml:"reanim_file"`

	// Animations 动画列表（可选，用于文档和验证）
	Animations []AnimationDef `yaml:"animations,omitempty"`

	// AnimationCombos 动画组合配置列表
	AnimationCombos []AnimationComboConfig `yaml:"animation_combos"`
}

// AnimationDef 动画定义（用于文档和验证）
type AnimationDef struct {
	// Name 动画名称（如 "anim_idle"）
	Name string `yaml:"name"`

	// DisplayName 显示名称（如 "待机"）
	DisplayName string `yaml:"display_name,omitempty"`
}

// AnimationComboConfig 动画组合配置
// 定义如何组合多个动画、轨道绑定策略、父子关系等
type AnimationComboConfig struct {
	// Name 组合名称（代码中引用）
	Name string `yaml:"name"`

	// DisplayName 显示名称（用于调试）
	DisplayName string `yaml:"display_name,omitempty"`

	// Animations 播放的动画列表
	Animations []string `yaml:"animations"`

	// Loop 是否循环播放（默认 true）
	Loop *bool `yaml:"loop,omitempty"`

	// AnimationLoopStates 每个动画的独立循环状态（可选）
	// 格式：动画名 -> 是否循环
	// 如果设置，将覆盖全局 Loop 设置
	AnimationLoopStates map[string]bool `yaml:"animation_loop_states,omitempty"`

	// BindingStrategy 轨道绑定策略（"auto" 或 "manual"）
	// auto: 自动分析轨道绑定，manual: 手动指定绑定
	BindingStrategy string `yaml:"binding_strategy,omitempty"`

	// ManualBindings 手动绑定配置（可选，当 strategy=manual 时使用）
	// 格式：轨道名 -> 动画名
	ManualBindings map[string]string `yaml:"manual_bindings,omitempty"`

	// ParentTracks 父子关系定义（可选）
	// 格式：子轨道名 -> 父轨道名
	ParentTracks map[string]string `yaml:"parent_tracks,omitempty"`

	// HiddenTracks 隐藏轨道列表（可选）
	HiddenTracks []string `yaml:"hidden_tracks,omitempty"`
}

// LoadReanimConfig 从 YAML 文件加载 Reanim 配置
//
// 参数：
//   - path: 配置文件路径（相对于项目根目录）
//
// 返回：
//   - *ReanimConfig: 解析后的配置对象
//   - error: 加载或解析错误
func LoadReanimConfig(path string) (*ReanimConfig, error) {
	// 1. 读取文件
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("无法读取配置文件 %s: %w", path, err)
	}

	// 2. 解析 YAML
	var config ReanimConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("无法解析配置文件 %s: %w", path, err)
	}

	// 3. 验证配置
	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("配置文件 %s 验证失败: %w", path, err)
	}

	return &config, nil
}

// validateConfig 验证配置的完整性和正确性
func validateConfig(config *ReanimConfig) error {
	// 验证必填字段
	if config.ReanimFile == "" {
		return fmt.Errorf("缺少必填字段 'reanim_file'")
	}

	// 构建动画名称映射（用于验证引用）
	animNames := make(map[string]bool)
	for _, anim := range config.Animations {
		if anim.Name == "" {
			return fmt.Errorf("动画定义缺少 'name' 字段")
		}
		animNames[anim.Name] = true
	}

	// 验证动画组合配置
	for i, combo := range config.AnimationCombos {
		// 验证组合名称
		if combo.Name == "" {
			return fmt.Errorf("动画组合 #%d 缺少 'name' 字段", i)
		}

		// 验证动画列表
		if len(combo.Animations) == 0 {
			return fmt.Errorf("动画组合 '%s' 的 'animations' 列表为空", combo.Name)
		}

		// 验证动画引用（如果提供了 animations 列表）
		if len(animNames) > 0 {
			for _, animName := range combo.Animations {
				if !animNames[animName] {
					return fmt.Errorf("动画组合 '%s' 引用了不存在的动画 '%s'", combo.Name, animName)
				}
			}
		}

		// 验证绑定策略
		if combo.BindingStrategy != "" &&
			combo.BindingStrategy != "auto" &&
			combo.BindingStrategy != "manual" {
			return fmt.Errorf("动画组合 '%s' 的绑定策略 '%s' 无效，只能是 'auto' 或 'manual'",
				combo.Name, combo.BindingStrategy)
		}

		// 如果是手动绑定，验证是否提供了绑定配置
		if combo.BindingStrategy == "manual" && len(combo.ManualBindings) == 0 {
			return fmt.Errorf("动画组合 '%s' 使用 manual 绑定策略但未提供 'manual_bindings'", combo.Name)
		}

		// 验证手动绑定中的动画引用
		if len(animNames) > 0 && len(combo.ManualBindings) > 0 {
			for _, animName := range combo.ManualBindings {
				if !animNames[animName] {
					return fmt.Errorf("动画组合 '%s' 的手动绑定引用了不存在的动画 '%s'",
						combo.Name, animName)
				}
			}
		}

		// 验证父子关系配置（避免循环依赖）
		if len(combo.ParentTracks) > 0 {
			if err := validateParentTracks(combo.ParentTracks); err != nil {
				return fmt.Errorf("动画组合 '%s' 的父子关系配置无效: %w", combo.Name, err)
			}
		}
	}

	return nil
}

// validateParentTracks 验证父子关系配置（避免循环依赖）
func validateParentTracks(parentTracks map[string]string) error {
	// 使用深度优先搜索检测循环
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(track string) bool {
		visited[track] = true
		recStack[track] = true

		// 检查父轨道
		if parent, exists := parentTracks[track]; exists {
			if !visited[parent] {
				if hasCycle(parent) {
					return true
				}
			} else if recStack[parent] {
				// 检测到循环
				return true
			}
		}

		recStack[track] = false
		return false
	}

	// 检查所有轨道
	for track := range parentTracks {
		if !visited[track] {
			if hasCycle(track) {
				return fmt.Errorf("父子关系配置存在循环依赖，涉及轨道: %s", track)
			}
		}
	}

	return nil
}
