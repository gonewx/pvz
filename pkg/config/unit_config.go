package config

// 单位配置常量
// 本文件定义了游戏单位（植物、僵尸、子弹等）的位置偏移和行为参数

// Zombie Configuration (僵尸配置)
const (
	// ZombieVerticalOffset 僵尸在格子中的垂直偏移量（像素）
	// 使用 CellHeight/2 (50.0) 使僵尸脚底对齐到格子底部
	// 渲染公式：screenY = pos.Y - CenterOffsetY，CenterOffsetY = maxY（脚底）
	// 因此 pos.Y 需要指向脚底应该在的位置（格子底部）
	ZombieVerticalOffset = PlantOffsetY

	// ZombieWalkSpeed 普通僵尸的移动速度（像素/秒）
	// 负值表示从右向左移动
	ZombieWalkSpeed = -30.0

	// ZombieDefaultHealth 普通僵尸的默认生命值
	ZombieDefaultHealth = 270

	// ZombieCollisionWidth 普通僵尸碰撞盒宽度（像素）
	ZombieCollisionWidth = 40.0

	// ZombieCollisionHeight 普通僵尸碰撞盒高度（像素）
	ZombieCollisionHeight = 115.0

	// ZombieCollisionOffsetX 普通僵尸碰撞盒X偏移量（像素）
	// 正值向右偏移（朝向植物方向），负值向左偏移
	// 僵尸身体中心相对于锚点（脚底中心）的水平偏移
	ZombieCollisionOffsetX = 20.0

	// ZombieCollisionOffsetY 普通僵尸碰撞盒Y偏移量（像素）
	// 负值向上偏移，使碰撞盒覆盖身体而不是地面
	// 僵尸位置指向脚底，碰撞盒需要向上偏移到躯干位置
	// 计算：碰撞盒底部 = pos.Y + OffsetY + Height/2
	// 希望碰撞盒底部略高于脚底（约 10 像素）
	// OffsetY = -Height/2 + 10 = -57.5 + 10 = -47.5，取 -50
	ZombieCollisionOffsetY = -50.0

	// ZombieFlagCollisionOffsetX 旗帜僵尸碰撞盒X偏移量（像素）
	// 正值向右偏移，使碰撞盒只检测身体部分而非旗子手
	// 旗子手向前伸出约40像素，偏移量设可使碰撞盒居中于身体
	ZombieFlagCollisionOffsetX = 20.0

	// PolevaulterCollisionOffsetX 撑杆僵尸碰撞盒X偏移量（像素）
	// 撑杆僵尸持杆时身体前倾，需要向前（左）调整碰撞盒
	// 负值向左偏移（朝向房子方向）
	PolevaulterCollisionOffsetX = 15.0

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

// Pole Vaulting Zombie Configuration (撑杆僵尸配置)
// Story 8.9: Level 1-6 新增僵尸类型
const (
	// PolevaulterZombieHealth 撑杆僵尸的生命值
	// 高于普通僵尸（270）的生命值
	PolevaulterZombieHealth = 500

	// PolevaulterZombieRunSpeed 撑杆僵尸持杆时的移动速度（像素/秒）
	// 约为普通僵尸的 1.8 倍（30 * 1.8 ≈ 54）
	// 负值表示从右向左移动
	PolevaulterZombieRunSpeed = -54.0

	// PolevaulterZombieWalkSpeed 撑杆僵尸跳跃后的移动速度（像素/秒）
	// 与普通僵尸相同
	PolevaulterZombieWalkSpeed = -30.0

	// PolevaulterZombieJumpDuration 撑杆僵尸跳跃动画持续时间（秒）
	// 增加到 1.5 秒让跳跃看起来更自然
	PolevaulterZombieJumpDuration = 1.5

	// PolevaulterZombieJumpDistance 撑杆僵尸跳跃时实体需要移动的距离（像素）
	// 从 Zombie_polevaulter.reanim 的 body1 轨道分析得出：
	// - jump 动画期间 (帧50-92)，body1.X 从 42.0 变化到 -109.2
	// - walk 动画开始时 (帧93)，body1.X 重置为 40.1
	// - 为保持视觉连续，实体 Position.X 需要补偿这个差值
	// - 补偿量 = jump结束时body1.X - walk开始时body1.X = -109.2 - 40.1 = -149.3
	// 负值表示向左移动
	PolevaulterZombieJumpDistance = -149.3
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
	PeaBulletOffsetY = -65.0

	// PeaBulletWidth 豌豆子弹碰撞盒宽度（像素）
	PeaBulletWidth = 28.0

	// PeaBulletHeight 豌豆子弹碰撞盒高度（像素）
	PeaBulletHeight = 28.0

	// PeaBulletTrailOffsetY 子弹拖尾粒子的 Y 偏移（像素）
	// 粒子应该在子弹视觉中心发射，而不是图像左上角
	// 值为子弹高度的一半
	PeaBulletTrailOffsetY = PeaBulletHeight / 2.0

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
)

// Potato Mine Configuration (土豆雷配置)
const (
	// PotatoMineSunCost 土豆雷的阳光消耗
	// 土豆雷是低成本的一次性爆炸植物
	PotatoMineSunCost = 25

	// PotatoMineRechargeTime 土豆雷的冷却时间（秒）
	PotatoMineRechargeTime = 30.0

	// PotatoMineDefaultHealth 土豆地雷默认生命值
	// 与其他普通植物相同
	PotatoMineDefaultHealth = 300

	// Story 8.9: 土豆地雷武装和爆炸配置
	// PotatoMineArmingTime 土豆雷武装时间（秒）
	PotatoMineArmingTime = 15.0

	// PotatoMineExplosionDamage 土豆雷爆炸伤害
	// 1800 点伤害，足以秒杀大多数普通僵尸
	PotatoMineExplosionDamage = 1800

	// PotatoMineExplosionRadius 土豆雷爆炸范围（格数）
	// 1x1 格（单格）
	PotatoMineExplosionRadius = 1.0

	// PotatoMineWarningLightMinSpeed 警告灯最慢闪烁速度倍率
	// 僵尸很远或没有僵尸时使用
	// 注意：此常量已废弃，改用 PotatoMineBlinkIntervalMax/Min
	PotatoMineWarningLightMinSpeed = 1.0

	// PotatoMineWarningLightMaxSpeed 警告灯最快闪烁速度倍率
	// 僵尸很近时使用
	// 注意：此常量已废弃，改用 PotatoMineBlinkIntervalMax/Min
	PotatoMineWarningLightMaxSpeed = 4.0

	// PotatoMineWarningDistanceMax 开始检测僵尸的最大距离（像素）
	// 超过此距离使用最慢速度
	PotatoMineWarningDistanceMax = 400.0

	// PotatoMineWarningDistanceMin 警告灯达到最快速度的距离（像素）
	// 僵尸距离小于此值时使用最快速度
	PotatoMineWarningDistanceMin = 50.0

	// PotatoMineBlinkPeriodMax 闪烁周期最大值（秒）
	// 僵尸很远时使用，红灯亮和灭各占一半
	PotatoMineBlinkPeriodMax = 0.8

	// PotatoMineBlinkPeriodMin 闪烁周期最小值（秒）
	// 僵尸很近时使用，红灯亮和灭各占一半
	PotatoMineBlinkPeriodMin = 0.16

	// PotatoMineLight1Size 灰灯图片尺寸（像素）
	PotatoMineLight1Size = 28.0

	// PotatoMineLight2Size 红灯图片尺寸（像素）
	PotatoMineLight2Size = 46.0

	// PotatoMineLightOffsetX 红灯替换灰灯时的 X 偏移（未缩放）
	// 计算：(灰灯中心 - 红灯中心) = (14 - 23) = -9
	PotatoMineLightOffsetX = -9.0

	// PotatoMineLightOffsetY 红灯替换灰灯时的 Y 偏移（未缩放）
	// 计算：(灰灯中心 - 红灯中心) = (14 - 23) = -9
	PotatoMineLightOffsetY = -9.0
)

// Snow Pea Configuration (寒冰射手配置)
// Story 8.9: Level 1-6 解锁植物
const (
	// SnowPeaSunCost 寒冰射手的阳光消耗
	// 比豌豆射手贵 75 阳光（100 + 75 = 175）
	SnowPeaSunCost = 175

	// SnowPeaRechargeTime 寒冰射手的冷却时间（秒）
	// 与豌豆射手相同，属于快速冷却类型
	SnowPeaRechargeTime = 7.5

	// SnowPeaSlowDuration 冰豌豆减速效果持续时间（秒）
	// 每次命中刷新计时器
	SnowPeaSlowDuration = 10.0

	// SnowPeaSlowMultiplier 冰豌豆减速效果倍率
	// 0.5 表示减速 50%（速度变为原来的一半）
	SnowPeaSlowMultiplier = 0.5
)

// Chomper Configuration (大嘴花配置)
// Story 8.11: Level 1-7 解锁植物
const (
	// ChomperSunCost 大嘴花的阳光消耗
	ChomperSunCost = 150

	// ChomperRechargeTime 大嘴花的冷却时间（秒）
	// 快速冷却类型
	ChomperRechargeTime = 7.5

	// ChomperBiteDamage 大嘴花撕咬伤害
	// 对无法吞噬的目标（巨人僵尸等）每次撕咬造成的伤害
	ChomperBiteDamage = 40

	// ChomperSwallowDamage 大嘴花吞噬伤害
	// 秒杀级伤害，一口吃掉普通僵尸
	ChomperSwallowDamage = 1800

	// ChomperDigestTime 大嘴花消化时间（秒）
	// 吞噬后进入消化期，期间无法攻击
	ChomperDigestTime = 42.0

	// ChomperAttackRange 大嘴花攻击范围（格子数）
	// 检测前方 2 格内的僵尸
	ChomperAttackRange = 2

	// ChomperHealth 大嘴花生命值
	// 标准植物生命值
	ChomperHealth = 300
)

// Repeater Configuration (双发射手配置)
// Story 8.12: Level 1-8 解锁植物
const (
	// RepeaterSunCost 双发射手的阳光消耗
	RepeaterSunCost = 200

	// RepeaterRechargeTime 双发射手的冷却时间（秒）
	// 快速冷却类型
	RepeaterRechargeTime = 7.5

	// RepeaterHealth 双发射手生命值
	// 标准植物生命值
	RepeaterHealth = 300

	// RepeaterBurstCount 双发射手每次攻击发射的子弹数量
	RepeaterBurstCount = 2

	// RepeaterBurstInterval 双发射手连发子弹之间的间隔（秒）
	// 约 0.15 秒，符合原版 PvZ 行为
	RepeaterBurstInterval = 0.15
)

// Cherry Bomb Configuration (樱桃炸弹配置)
const (
	// CherryBombSunCost 樱桃炸弹的阳光消耗
	// 樱桃炸弹是高成本的一次性爆炸植物
	CherryBombSunCost = 150

	// CherryBombFuseTime 樱桃炸弹引信时间（秒）
	// 种植后到爆炸的延迟时间（匹配原版：1秒后爆炸）
	CherryBombFuseTime = 1.0

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
