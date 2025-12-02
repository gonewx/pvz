package systems

import (
	"log"

	"github.com/decker502/pvz/internal/reanim"
	"github.com/decker502/pvz/pkg/config"
	"github.com/decker502/pvz/pkg/ecs"
	"github.com/hajimehoshi/ebiten/v2"
)

// ReanimResourceLoader 定义 ReanimSystem 所需的资源加载接口
// 用于运行时切换单位时加载新的 Reanim 数据
//
// 这个接口由 ResourceManager 实现，通过接口注入避免循环依赖
// Story 5.4.1: 支持运行时单位切换（如僵尸切换到烧焦僵尸）
type ReanimResourceLoader interface {
	// GetReanimXML 获取指定单位的 ReanimXML 数据
	GetReanimXML(unitName string) *reanim.ReanimXML
	// GetReanimPartImages 获取指定单位的部件图片
	GetReanimPartImages(unitName string) map[string]*ebiten.Image
	// LoadImage 加载指定路径的图片（用于 ImageOverrides）
	LoadImage(path string) (*ebiten.Image, error)
}

// ReanimSystem 是 Reanim 动画系统
// 基于 animation_showcase/AnimationCell 重写，简化并修复遗留问题
//
// 文件拆分结构：
//   - reanim_system.go: 核心结构定义和构造函数
//   - reanim_api.go: 核心播放 API (PlayAnimation, PlayCombo 等)
//   - reanim_update.go: Update 循环和命令处理逻辑
//   - reanim_render.go: 渲染缓存和纹理渲染逻辑
//   - reanim_helpers.go: 辅助函数和工具方法
//
// Story 5.4.1: 支持运行时单位切换（如僵尸切换到烧焦僵尸）
type ReanimSystem struct {
	entityManager  *ecs.EntityManager
	configManager  *config.ReanimConfigManager
	resourceLoader ReanimResourceLoader // Story 5.4.1: 用于运行时加载不同单位的 Reanim 数据

	// 游戏 TPS（用于帧推进计算）
	targetTPS float64

	enableCommandCleanup bool    // 是否启用自动清理
	cleanupInterval      float64 // 清理间隔（秒）
	cleanupTimer         float64 // 清理计时器
}

// NewReanimSystem 创建新的 Reanim 动画系统
func NewReanimSystem(em *ecs.EntityManager) *ReanimSystem {
	return &ReanimSystem{
		entityManager:        em,
		targetTPS:            60.0, // 默认 60 TPS
		enableCommandCleanup: false,
		cleanupInterval:      1.0, // 每秒清理一次
		cleanupTimer:         0.0,
	}
}

// SetConfigManager 设置配置管理器
func (s *ReanimSystem) SetConfigManager(cm *config.ReanimConfigManager) {
	s.configManager = cm
}

// SetResourceLoader 设置资源加载器
// Story 5.4.1: 用于运行时单位切换时加载新的 Reanim 数据
//
// 参数：
//   - loader: 实现 ReanimResourceLoader 接口的资源管理器（通常是 ResourceManager）
func (s *ReanimSystem) SetResourceLoader(loader ReanimResourceLoader) {
	s.resourceLoader = loader
	log.Printf("[ReanimSystem] 资源加载器已设置")
}

// SetTargetTPS 设置目标 TPS（用于帧推进计算）
func (s *ReanimSystem) SetTargetTPS(tps float64) {
	s.targetTPS = tps
}

// SetCommandCleanup 设置命令清理策略（可选 API）
// 用于配置动画命令组件的自动清理
func (s *ReanimSystem) SetCommandCleanup(enable bool, interval float64) {
	s.enableCommandCleanup = enable
	s.cleanupInterval = interval
	log.Printf("[ReanimSystem] 命令清理配置: enable=%v, interval=%.2f秒", enable, interval)
}
