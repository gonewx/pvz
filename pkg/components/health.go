package components

// DeathEffectType 死亡效果类型
// 用于区分不同的死亡表现方式
type DeathEffectType int

const (
	// DeathEffectNormal 普通死亡：头部掉落、手臂掉落粒子效果
	DeathEffectNormal DeathEffectType = iota
	// DeathEffectExplosion 爆炸死亡：烧焦动画，无肢体掉落效果
	DeathEffectExplosion
	// DeathEffectInstant 瞬间死亡：无肢体掉落效果（如坚果保龄球撞击）
	DeathEffectInstant
)

// HealthComponent 存储实体的生命值信息
// 用于僵尸、植物等可被攻击的实体
type HealthComponent struct {
	CurrentHealth   int             // 当前生命值
	MaxHealth       int             // 最大生命值
	ArmLost         bool            // 僵尸手臂是否已掉落（用于防止重复触发粒子效果）
	DeathEffectType DeathEffectType // 死亡效果类型
}
