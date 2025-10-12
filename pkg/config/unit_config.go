package config

// 单位配置常量
// 本文件定义了游戏单位（植物、僵尸、子弹等）的位置偏移和行为参数

// Zombie Configuration (僵尸配置)
const (
	// ZombieVerticalOffset 僵尸在格子中的垂直偏移量（像素）
	// 用于微调僵尸在格子中的垂直位置
	// 建议值范围：25.0 - 50.0
	// 当前使用 CellHeight/2 (50.0) 使僵尸在格子中心
	ZombieVerticalOffset = 50.0

	// ZombieWalkAnimationFrames 普通僵尸走路动画的总帧数
	ZombieWalkAnimationFrames = 22

	// ZombieWalkFrameSpeed 僵尸走路动画的帧速率（秒/帧）
	ZombieWalkFrameSpeed = 0.1

	// ZombieWalkSpeed 普通僵尸的移动速度（像素/秒）
	// 负值表示从右向左移动
	ZombieWalkSpeed = -30.0

	// ZombieDefaultHealth 普通僵尸的默认生命值
	ZombieDefaultHealth = 270

	// ZombieCollisionWidth 普通僵尸碰撞盒宽度（像素）
	ZombieCollisionWidth = 40.0

	// ZombieCollisionHeight 普通僵尸碰撞盒高度（像素）
	ZombieCollisionHeight = 115.0

	// ZombieDeletionBoundary 僵尸删除边界（世界坐标X）
	// 僵尸移出此边界后将被删除
	ZombieDeletionBoundary = -100.0
)

// Projectile Configuration (子弹配置)
const (
	// PeaBulletSpeed 豌豆子弹移动速度（像素/秒）
	// 正值表示向右移动
	PeaBulletSpeed = 200.0

	// PeaBulletDamage 豌豆子弹伤害值
	PeaBulletDamage = 20

	// PeaBulletOffsetX 子弹相对豌豆射手中心的水平偏移量（像素）
	// 建议值范围：40.0 - 60.0
	// 50像素使子弹从豌豆射手嘴部发射（豌豆射手朝右，嘴在右侧）
	PeaBulletOffsetX = 25.0

	// PeaBulletOffsetY 子弹相对豌豆射手中心的垂直偏移量（像素）
	// 建议值范围：-10.0 - 10.0
	// 0像素使子弹与豌豆射手在同一水平线，确保能击中同行僵尸
	PeaBulletOffsetY = -18.0

	// PeaBulletWidth 豌豆子弹碰撞盒宽度（像素）
	PeaBulletWidth = 28.0

	// PeaBulletHeight 豌豆子弹碰撞盒高度（像素）
	PeaBulletHeight = 28.0

	// PeaBulletDeletionBoundary 子弹删除边界（屏幕坐标X）
	// 子弹移出此边界后将被删除
	PeaBulletDeletionBoundary = 1500.0
)

// Effect Configuration (效果配置)
const (
	// HitEffectDuration 击中效果显示时长（秒）
	// 击中效果显示短暂时间后自动消失
	HitEffectDuration = 0.2
)

// Plant Configuration (植物配置)
const (
	// PeashooterAttackCooldown 豌豆射手攻击冷却时间（秒）
	PeashooterAttackCooldown = 1.4

	// PeashooterAnimationFrames 豌豆射手动画帧数
	PeashooterAnimationFrames = 13

	// PeashooterFrameSpeed 豌豆射手动画帧速率（秒/帧）
	PeashooterFrameSpeed = 0.08

	// SunflowerProductionCooldown 向日葵阳光生产冷却时间（秒）
	SunflowerProductionCooldown = 24.0

	// SunflowerFirstProductionTime 向日葵首次生产阳光时间（秒）
	SunflowerFirstProductionTime = 7.0

	// SunflowerAnimationFrames 向日葵动画帧数
	SunflowerAnimationFrames = 18

	// SunflowerFrameSpeed 向日葵动画帧速率（秒/帧）
	SunflowerFrameSpeed = 0.08
)
