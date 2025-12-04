package components

// HealthComponent 存储实体的生命值信息
// 用于僵尸、植物等可被攻击的实体
type HealthComponent struct {
	CurrentHealth     int  // 当前生命值
	MaxHealth         int  // 最大生命值
	ArmLost           bool // 僵尸手臂是否已掉落（用于防止重复触发粒子效果）
	KilledByExplosion bool // 是否被爆炸杀死（爆炸坚果、樱桃炸弹等），用于触发烧焦死亡动画
}
