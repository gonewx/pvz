package components

// ArmorComponent 存储实体的护甲信息
// 用于路障僵尸、铁桶僵尸等拥有额外防护层的单位
//
// 设计说明:
// - 当实体同时拥有 HealthComponent 和 ArmorComponent 时,伤害优先扣除护甲
// - 当 CurrentArmor <= 0 时,护甲层被破坏,开始扣除生命值
// - 护甲耗尽时需要触发外观切换(路障/铁桶掉落)
type ArmorComponent struct {
	CurrentArmor int // 当前护甲值
	MaxArmor     int // 最大护甲值
}
