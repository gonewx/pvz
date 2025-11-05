package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

// ReanimPlaybackConfigFile 定义 Reanim 播放模式配置文件结构
type ReanimPlaybackConfigFile struct {
	Version    string                          `yaml:"version"`
	Animations map[string]ReanimAnimationConfig `yaml:"animations"`
}

// ReanimAnimationConfig 定义单个 Reanim 动画的配置
type ReanimAnimationConfig struct {
	Mode                   string                              `yaml:"mode"`                      // 播放模式：Simple, Skeleton, Sequence, ComplexScene, Blended
	Description            string                              `yaml:"description"`               // 描述
	Notes                  string                              `yaml:"notes"`                     // 备注
	IndependentAnimations  []string                            `yaml:"independent_animations"`    // 独立播放的动画轨道列表（ComplexScene 使用）
	IndependentAnimConfigs map[string]IndependentAnimConfig    `yaml:"independent_anim_configs"`  // 独立动画的详细配置
	ParentBone             string                              `yaml:"parent_bone"`               // 父骨骼轨道名称（Blended 使用）
}

// IndependentAnimConfig 定义独立动画的播放配置
type IndependentAnimConfig struct {
	DelayDuration     float64  `yaml:"delay_duration"`      // 延迟时长（秒），0 表示无延迟立即循环
	IsLooping         *bool    `yaml:"is_looping"`          // 是否循环，nil 表示使用默认值 true
	IsActive          *bool    `yaml:"is_active"`           // 是否默认激活（控制帧推进），nil 表示使用默认值 true
	RenderWhenStopped *bool    `yaml:"render_when_stopped"` // 停止推进后是否继续渲染，nil 表示使用默认值 true
	LockAtFrame       *int     `yaml:"lock_at_frame"`       // 可选：锁定在指定帧（nil = 使用最后一帧）
	ControlledTracks  []string `yaml:"controlled_tracks"`   // 该动画控制的轨道列表（覆盖默认命名匹配）
}

// ReanimPlaybackConfig 是全局的 Reanim 播放模式配置实例
var ReanimPlaybackConfig *ReanimPlaybackConfigFile

// LoadReanimPlaybackConfig 加载 Reanim 播放模式配置文件
//
// 配置文件路径：data/reanim_playback_config.yaml
//
// 如果配置文件不存在或加载失败，将使用启发式算法作为后备。
func LoadReanimPlaybackConfig() error {
	configPath := "data/reanim_playback_config.yaml"

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Printf("[ReanimConfig] Warning: Failed to load config file '%s': %v", configPath, err)
		log.Printf("[ReanimConfig] Will use heuristic algorithm for mode detection")
		return err
	}

	// 解析 YAML
	config := &ReanimPlaybackConfigFile{}
	if err := yaml.Unmarshal(data, config); err != nil {
		log.Printf("[ReanimConfig] Error: Failed to parse config file '%s': %v", configPath, err)
		return fmt.Errorf("failed to parse reanim config: %w", err)
	}

	// 验证配置
	if config.Version == "" {
		log.Printf("[ReanimConfig] Warning: Config file has no version field")
	}

	if len(config.Animations) == 0 {
		log.Printf("[ReanimConfig] Warning: Config file has no animations defined")
	}

	// 保存到全局变量
	ReanimPlaybackConfig = config

	log.Printf("[ReanimConfig] ✅ Loaded Reanim playback config (version=%s, animations=%d)",
		config.Version, len(config.Animations))

	return nil
}

// GetAnimationMode 查询指定 Reanim 动画的播放模式
//
// 参数：
//   - animName: Reanim 动画名称（不含 .reanim 后缀）
//
// 返回：
//   - mode: 播放模式字符串（"Simple", "Skeleton", "Sequence", "ComplexScene", "Blended"）
//   - found: 是否在配置中找到该动画
func GetAnimationMode(animName string) (mode string, found bool) {
	if ReanimPlaybackConfig == nil {
		return "", false
	}

	animConfig, exists := ReanimPlaybackConfig.Animations[animName]
	if !exists {
		return "", false
	}

	return animConfig.Mode, true
}

// GetAnimationConfig 查询指定 Reanim 动画的完整配置
//
// 参数：
//   - animName: Reanim 动画名称（不含 .reanim 后缀）
//
// 返回：
//   - config: 动画配置
//   - found: 是否在配置中找到该动画
func GetAnimationConfig(animName string) (config ReanimAnimationConfig, found bool) {
	if ReanimPlaybackConfig == nil {
		return ReanimAnimationConfig{}, false
	}

	animConfig, exists := ReanimPlaybackConfig.Animations[animName]
	if !exists {
		return ReanimAnimationConfig{}, false
	}

	return animConfig, true
}
