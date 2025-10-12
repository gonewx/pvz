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
	// BehaviorZombieDying 僵尸死亡中：播放死亡动画，动画完成后删除实体
	BehaviorZombieDying
)

// BehaviorComponent 标识实体的行为类型
// 此组件用于让 BehaviorSystem 识别实体应执行何种行为逻辑
type BehaviorComponent struct {
	Type BehaviorType // 行为类型（向日葵、豌豆射手等）
}
