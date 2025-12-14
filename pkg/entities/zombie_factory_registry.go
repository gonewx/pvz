package entities

import (
	"fmt"

	"github.com/gonewx/pvz/pkg/ecs"
)

// ZombieFactoryFunc 僵尸工厂函数类型
type ZombieFactoryFunc func(em *ecs.EntityManager, rm ResourceLoader, row int, spawnX float64) (ecs.EntityID, error)

// zombieFactoryRegistry 僵尸工厂函数注册表
// 新增僵尸时需要在此处注册工厂函数
var zombieFactoryRegistry = map[string]ZombieFactoryFunc{
	"basic":       NewZombieEntity,
	"zombie":      NewZombieEntity, // 别名
	"conehead":    NewConeheadZombieEntity,
	"buckethead":  NewBucketheadZombieEntity,
	"flag":        NewFlagZombieEntity,
	"polevaulter": NewPolevaulterZombieEntity,
}

// CreateZombieByID 根据配置ID创建僵尸实体
// 使用统一的工厂注册表，新增僵尸时只需在 zombieFactoryRegistry 中添加记录
//
// 参数:
//   - zombieID: 僵尸配置ID (如 "basic", "conehead", "buckethead" 等)
//   - em: 实体管理器
//   - rm: 资源管理器
//   - row: 生成行索引 (0-4)
//   - spawnX: 生成的世界坐标X位置
//
// 返回:
//   - ecs.EntityID: 创建的僵尸实体ID
//   - error: 如果僵尸ID未知或创建失败返回错误
func CreateZombieByID(zombieID string, em *ecs.EntityManager, rm ResourceLoader, row int, spawnX float64) (ecs.EntityID, error) {
	factory, ok := zombieFactoryRegistry[zombieID]
	if !ok {
		return 0, fmt.Errorf("unknown zombie type: %s", zombieID)
	}
	return factory(em, rm, row, spawnX)
}

// RegisterZombieFactory 注册新的僵尸工厂函数
// 允许在运行时动态注册新的僵尸类型
func RegisterZombieFactory(zombieID string, factory ZombieFactoryFunc) {
	zombieFactoryRegistry[zombieID] = factory
}

// GetRegisteredZombieTypes 返回所有已注册的僵尸类型ID
func GetRegisteredZombieTypes() []string {
	types := make([]string, 0, len(zombieFactoryRegistry))
	for id := range zombieFactoryRegistry {
		types = append(types, id)
	}
	return types
}

// IsZombieTypeRegistered 检查僵尸类型是否已注册
func IsZombieTypeRegistered(zombieID string) bool {
	_, ok := zombieFactoryRegistry[zombieID]
	return ok
}
