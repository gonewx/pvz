package config

// 单位配置常量
// 本文件定义了游戏单位（植物、僵尸、子弹等）的位置偏移和行为参数

// Zombie Configuration (僵尸配置)
const (
	// ZombieVerticalOffset 僵尸在格子中的垂直偏移量（像素）
	// 用于微调僵尸在格子中的垂直位置
	// 建议值范围：25.0 - 50.0
	// 当前使用 CellHeight/2 (50.0) 使僵尸在格子中心
	ZombieVerticalOffset = -25.0

	// ZombieWalkSpeed 普通僵尸的移动速度（像素/秒）
	// 负值表示从右向左移动
	ZombieWalkSpeed = -30.0

	// ZombieDefaultHealth 普通僵尸的默认生命值
	ZombieDefaultHealth = 270

	// ZombieCollisionWidth 普通僵尸碰撞盒宽度（像素）
	ZombieCollisionWidth = 40.0

	// ZombieCollisionHeight 普通僵尸碰撞盒高度（像素）
	ZombieCollisionHeight = 115.0

	// ZombieFlagCollisionOffsetX 旗帜僵尸碰撞盒X偏移量（像素）
	// 正值向右偏移，使碰撞盒只检测身体部分而非旗子手
	// 旗子手向前伸出约40像素，偏移量设可使碰撞盒居中于身体
	ZombieFlagCollisionOffsetX = 30.0

	// ZombieDeletionBoundary 僵尸删除边界（世界坐标X）
	// 僵尸移出此边界后将被删除
	ZombieDeletionBoundary = -100.0

	// ZombieDieAnimationFrames 普通僵尸死亡动画的总帧数
	ZombieDieAnimationFrames = 10

	// ZombieDieFrameSpeed 僵尸死亡动画的帧速率（秒/帧）
	ZombieDieFrameSpeed = 0.1

	// ConeheadZombieArmorHealth 路障僵尸护甲值
	// 路障僵尸拥有370点护甲，护甲破坏后变为普通僵尸
	ConeheadZombieArmorHealth = 370

	// BucketheadZombieArmorHealth 铁桶僵尸护甲值
	// 铁桶僵尸拥有1100点护甲，护甲破坏后变为普通僵尸
	BucketheadZombieArmorHealth = 1100

	// ZombieGroanMinInterval 僵尸呻吟音效最小间隔（秒）
	// 控制呻吟音效不要太频繁
	ZombieGroanMinInterval = 5.0

	// ZombieGroanMaxInterval 僵尸呻吟音效最大间隔（秒）
	// 增加随机性，避免呻吟过于规律
	ZombieGroanMaxInterval = 10.0

	// ZombieActivationDelayMin 僵尸激活延迟最小值（秒）
	// 同一波次僵尸触发后，每个僵尸等待随机延迟后才开始移动
	// 用于实现散落入场效果，避免同时进入画面显得整齐
	ZombieActivationDelayMin = 0.5

	// ZombieActivationDelayMax 僵尸激活延迟最大值（秒）
	// 建议值范围：1.0 - 3.0
	// 值越大，同一波僵尸入场时间差越明显
	ZombieActivationDelayMax = 3.0
)

// Projectile Configuration (子弹配置)
const (
	// PeaBulletSpeed 豌豆子弹移动速度（像素/秒）
	// 正值表示向右移动
	PeaBulletSpeed = 333.0

	// PeaBulletDamage 豌豆子弹伤害值
	PeaBulletDamage = 20

	// PeaBulletOffsetX 子弹相对豌豆射手中心的水平偏移量（像素）
	PeaBulletOffsetX = 35.0

	// PeaBulletOffsetY 子弹相对豌豆射手中心的垂直偏移量（像素）
	PeaBulletOffsetY = -35.0

	// PeaBulletWidth 豌豆子弹碰撞盒宽度（像素）
	PeaBulletWidth = 28.0

	// PeaBulletHeight 豌豆子弹碰撞盒高度（像素）
	PeaBulletHeight = 28.0

	// PeaBulletDeletionBoundary 子弹删除边界（屏幕坐标X）
	// 子弹移出此边界后将被删除
	PeaBulletDeletionBoundary = 1500.0
)

// Sun Configuration (阳光配置)
const (
	// SunClickableWidth 阳光可点击区域宽度（像素）
	// 建议值范围：80.0 - 120.0
	// 增大此值可以让阳光更容易点击
	// 原版游戏阳光图片约80x80，建议设置为100-110以提高手感
	SunClickableWidth = 80.0

	// SunClickableHeight 阳光可点击区域高度（像素）
	// 建议值范围：80.0 - 120.0
	// 增大此值可以让阳光更容易点击
	SunClickableHeight = 80.0
)

// Effect Configuration (效果配置)
const (
	// HitEffectDuration 击中效果显示时长（秒）
	// 击中效果显示短暂时间后自动消失
	HitEffectDuration = 0.2
)

// Plant Configuration (植物配置)
const (
	// Sunflower (向日葵)
	// SunflowerSunCost 向日葵的阳光消耗
	SunflowerSunCost = 50

	// SunflowerRechargeTime 向日葵的冷却时间（秒）
	SunflowerRechargeTime = 7.5

	// SunflowerProductionCooldown 向日葵阳光生产冷却时间（秒）
	SunflowerProductionCooldown = 24.0

	// SunflowerFirstProductionTime 向日葵首次生产阳光时间（秒）
	SunflowerFirstProductionTime = 7.0

	// SunflowerAnimationFrames 向日葵动画帧数
	SunflowerAnimationFrames = 18

	// SunflowerFrameSpeed 向日葵动画帧速率（秒/帧）
	SunflowerFrameSpeed = 0.08

	// Peashooter (豌豆射手)
	// PeashooterSunCost 豌豆射手的阳光消耗
	PeashooterSunCost = 100

	// PeashooterRechargeTime 豌豆射手的冷却时间（秒）
	PeashooterRechargeTime = 7.5

	// Wallnut (坚果墙)
	// WallnutAnimationFrames 坚果墙动画帧数
	// 坚果墙的完好、轻伤、重伤状态都使用16帧动画
	WallnutAnimationFrames = 16

	// WallnutFrameSpeed 坚果墙动画帧速率（秒/帧）
	WallnutFrameSpeed = 0.1

	// WallnutCost 坚果墙的阳光消耗
	WallnutCost = 50

	// WallnutRechargeTime 坚果墙的冷却时间（秒）
	WallnutRechargeTime = 30.0

	// WallnutCracked1Threshold 坚果墙轻伤状态生命值阈值（百分比）
	// 当生命值 <= 66% 时，坚果墙进入轻伤状态（出现第一级裂痕）
	WallnutCracked1Threshold = 0.66

	// WallnutCracked2Threshold 坚果墙重伤状态生命值阈值（百分比）
	// 当生命值 <= 33% 时，坚果墙进入重伤状态（出现第二级裂痕）
	WallnutCracked2Threshold = 0.33

	// WallnutHitGlowColorR 坚果墙被啃食发光效果的红色通道
	// 使用白色/浅黄色发光效果
	WallnutHitGlowColorR = 1.5

	// WallnutHitGlowColorG 坚果墙被啃食发光效果的绿色通道
	WallnutHitGlowColorG = 1.5

	// WallnutHitGlowColorB 坚果墙被啃食发光效果的蓝色通道
	WallnutHitGlowColorB = 1.2

	// WallnutHitGlowFadeSpeed 坚果墙被啃食发光效果的衰减速度（每秒）
	// 4.0 表示 0.25 秒内从最亮完全衰减（快速闪烁效果）
	WallnutHitGlowFadeSpeed = 4.0

	// WallnutBlinkIntervalMin 坚果墙眨眼最小间隔（秒）
	WallnutBlinkIntervalMin = 4.0

	// WallnutBlinkIntervalMax 坚果墙眨眼最大间隔（秒）
	WallnutBlinkIntervalMax = 8.0
)

// Plant Health Configuration (植物生命值配置)
const (
	// SunflowerDefaultHealth 向日葵默认生命值
	// 向日葵较脆弱，生命值较低
	SunflowerDefaultHealth = 300

	// PeashooterDefaultHealth 豌豆射手默认生命值
	// 豌豆射手生命值略高于向日葵
	PeashooterDefaultHealth = 300

	// WallnutDefaultHealth 坚果墙默认生命值
	// 坚果墙作为防御植物，拥有远高于其他植物的生命值
	WallnutDefaultHealth = 4000 // 原版游戏数值，是向日葵的13倍
)

// Zombie Eating Configuration (僵尸啃食配置)
const (
	// ZombieEatingDamage 僵尸每次啃食造成的伤害
	// 伤害触发时机由动画帧控制（与音效同步）：
	// - 普通僵尸（双手啃食）：每次动画循环触发 2 次（开始和中间点）
	// - 旗帜僵尸（单手啃食）：每次动画循环触发 1 次（开始）
	ZombieEatingDamage = 100

	// ZombieEatAnimationFrames 僵尸啃食动画帧数
	// 需要根据实际资源文件确定
	ZombieEatAnimationFrames = 21

	// ZombieEatParticleOffsetX 僵尸啃食粒子效果 X 偏移量
	// 僵尸面朝左，嘴巴在身体前方（左侧）
	// 负值表示向左偏移（朝向植物方向，即僵尸嘴巴位置）
	ZombieEatParticleOffsetX = -30.0

	// ZombieEatParticleOffsetY 僵尸啃食粒子效果 Y 偏移量
	// 嘴巴位置相对于僵尸中心略微偏上（头部位置）
	ZombieEatParticleOffsetY = -20.0
)

// Potato Mine Configuration (土豆雷配置)
const (
	// PotatoMineSunCost 土豆雷的阳光消耗
	// 土豆雷是低成本的一次性爆炸植物
	PotatoMineSunCost = 25

	// PotatoMineRechargeTime 土豆雷的冷却时间（秒）
	PotatoMineRechargeTime = 30.0
)

// Cherry Bomb Configuration (樱桃炸弹配置)
const (
	// CherryBombSunCost 樱桃炸弹的阳光消耗
	// 樱桃炸弹是高成本的一次性爆炸植物
	CherryBombSunCost = 150

	// CherryBombFuseTime 樱桃炸弹引信时间（秒）
	// 种植后到爆炸的延迟时间
	CherryBombFuseTime = 1.5

	// CherryBombDamage 樱桃炸弹爆炸伤害
	// 1800点伤害足以秒杀所有僵尸（包括铁桶僵尸1370总生命值）
	CherryBombDamage = 1800

	// CherryBombExplosionCenterOffsetX 爆炸圆心相对于植物位置的X偏移（像素）
	// 修正：植物坐标本身已是网格中心，偏移量设为 0
	CherryBombExplosionCenterOffsetX = 0.0

	// CherryBombExplosionCenterOffsetY 爆炸圆心相对于植物位置的Y偏移（像素）
	// 修正：植物坐标本身已是网格中心（微调），偏移量设为 0
	CherryBombExplosionCenterOffsetY = 0.0

	// CherryBombExplosionRadius 爆炸范围半径（像素）
	CherryBombExplosionRadius = 115.0

	// CherryBombCooldown 樱桃炸弹的冷却时间（秒）
	CherryBombCooldown = 50.0

	// ExplosiveNutDamage 爆炸坚果爆炸伤害值
	// Story 19.8: 与樱桃炸弹相同（1800），足以秒杀所有僵尸
	ExplosiveNutDamage = 1800

	// ExplosiveNutParticleEffect 爆炸坚果粒子效果名称
	// Story 19.8: 使用 Powie.xml 粒子配置（3个发射器）
	ExplosiveNutParticleEffect = "Powie"
)
