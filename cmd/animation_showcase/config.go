// cmd/animation_showcase/config.go
// 动画展示系统的配置文件加载和解析模块

package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// GlobalConfig 全局配置
type GlobalConfig struct {
	Window   WindowConfig   `yaml:"window"`
	Grid     GridConfig     `yaml:"grid"`
	Playback PlaybackConfig `yaml:"playback"`
}

// WindowConfig 窗口配置
type WindowConfig struct {
	Width  int    `yaml:"width"`
	Height int    `yaml:"height"`
	Title  string `yaml:"title"`
}

// GridConfig 网格布局配置
type GridConfig struct {
	Columns     int `yaml:"columns"`
	CellWidth   int `yaml:"cell_width"`
	CellHeight  int `yaml:"cell_height"`
	Padding     int `yaml:"padding"`
	ScrollSpeed int `yaml:"scroll_speed"`
	RowsPerPage int `yaml:"rows_per_page"` // 每页显示的行数
}

// PlaybackConfig 播放配置
type PlaybackConfig struct {
	TPS   int     `yaml:"tps"`   // 游戏目标 TPS（Ticks Per Second）
	FPS   int     `yaml:"fps"`   // 默认动画帧率（当 reanim 文件未指定时使用）
	Scale float64 `yaml:"scale"` // 默认缩放比例
}

// AnimationUnitConfig 动画单元配置
type AnimationUnitConfig struct {
	ID                  string                 `yaml:"id"`
	Name                string                 `yaml:"name"`
	ReanimFile          string                 `yaml:"reanim_file"`
	DefaultAnimation    string                 `yaml:"default_animation"`
	Scale               float64                `yaml:"scale"`
	Alignment           string                 `yaml:"alignment"` // 对齐方式: "center"(默认) 或 "top-left"
	Images              map[string]string      `yaml:"images"`
	AvailableAnimations []AnimationInfo        `yaml:"available_animations"`
	AnimationCombos     []AnimationComboConfig `yaml:"animation_combos"`
}

// AnimationInfo 动画信息
type AnimationInfo struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
}

// AnimationComboConfig 多动画组合配置
type AnimationComboConfig struct {
	Name            string            `yaml:"name"`
	DisplayName     string            `yaml:"display_name"`
	Animations      []string          `yaml:"animations"`
	BindingStrategy string            `yaml:"binding_strategy"` // "auto" 或 "manual"
	ParentTracks    map[string]string `yaml:"parent_tracks"`
	HiddenTracks    []string          `yaml:"hidden_tracks"`
	ManualBindings  map[string]string `yaml:"manual_bindings,omitempty"` // 手动绑定时使用
}

// ShowcaseConfig 展示系统完整配置
type ShowcaseConfig struct {
	Global     GlobalConfig          `yaml:"global"`
	Animations []AnimationUnitConfig `yaml:"animations"`
}

// LoadConfig 从文件加载配置
func LoadConfig(configPath string) (*ShowcaseConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config ShowcaseConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if config.Global.Playback.TPS == 0 {
		config.Global.Playback.TPS = 60 // 默认 60 TPS
	}
	if config.Global.Playback.FPS == 0 {
		config.Global.Playback.FPS = 12 // 默认 12 FPS（仅当 reanim 文件未指定时使用）
	}
	if config.Global.Playback.Scale == 0 {
		config.Global.Playback.Scale = 1.0
	}
	if config.Global.Grid.Columns == 0 {
		config.Global.Grid.Columns = 8
	}
	if config.Global.Grid.CellWidth == 0 {
		config.Global.Grid.CellWidth = 200
	}
	if config.Global.Grid.CellHeight == 0 {
		config.Global.Grid.CellHeight = 200
	}
	if config.Global.Grid.Padding == 0 {
		config.Global.Grid.Padding = 10
	}
	if config.Global.Grid.ScrollSpeed == 0 {
		config.Global.Grid.ScrollSpeed = 30
	}
	if config.Global.Grid.RowsPerPage == 0 {
		config.Global.Grid.RowsPerPage = 8 // 默认每页8行
	}

	// 为每个动画单元设置默认值
	for i := range config.Animations {
		if config.Animations[i].Scale == 0 {
			config.Animations[i].Scale = 1.0
		}
	}

	return &config, nil
}

// GetAnimationUnit 根据 ID 获取动画单元配置
func (c *ShowcaseConfig) GetAnimationUnit(id string) *AnimationUnitConfig {
	for i := range c.Animations {
		if c.Animations[i].ID == id {
			return &c.Animations[i]
		}
	}
	return nil
}
