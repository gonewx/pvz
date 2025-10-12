package components

// BehaviorType 定义实体的行为类型
// 用于 BehaviorSystem 决定如何处理该实体
type BehaviorType int

const (
	// BehaviorSunflower 向日葵行为：定期生产阳光
	BehaviorSunflower BehaviorType = iota
	// BehaviorPeashooter 豌豆射手行为：攻击同行僵尸
	BehaviorPeashooter
	// BehaviorPeaProjectile 豌豆子弹行为：向右移动并检测碰撞
	BehaviorPeaProjectile
	// BehaviorPeaBulletHit 豌豆子弹击中效果：显示击中水花动画，短暂显示后消失
	BehaviorPeaBulletHit
	// BehaviorZombieBasic 普通僵尸行为：从右向左移动并攻击植物
	BehaviorZombieBasic
	// BehaviorZombieEating 僵尸啃食植物行为：停止移动，周期性对植物造成伤害
	BehaviorZombieEating
	// BehaviorZombieDying 僵尸死亡中：播放死亡动画，动画完成后删除实体
	BehaviorZombieDying
	// BehaviorWallnut 坚果墙行为：无攻击能力的纯防御植物，根据生命值百分比切换外观状态
	// 坚果墙拥有极高的生命值(4000)，用于阻挡僵尸前进
	// 外观状态：完好(>66%) → 轻伤(33-66%) → 重伤(<33%)
	BehaviorWallnut
)

// BehaviorComponent 标识实体的行为类型
// 此组件用于让 BehaviorSystem 识别实体应执行何种行为逻辑
type BehaviorComponent struct {
	Type BehaviorType // 行为类型（向日葵、豌豆射手等）
}
