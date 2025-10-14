package config

// 单位配置常量
// 本文件定义了游戏单位（植物、僵尸、子弹等）的位置偏移和行为参数

// Zombie Configuration (僵尸配置)
const (
	// ZombieVerticalOffset 僵尸在格子中的垂直偏移量（像素）
	// 用于微调僵尸在格子中的垂直位置
	// 建议值范围：25.0 - 50.0
	// 当前使用 CellHeight/2 (50.0) 使僵尸在格子中心
	ZombieVerticalOffset = 25.0

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

	// ZombieDieAnimationFrames 普通僵尸死亡动画的总帧数
	ZombieDieAnimationFrames = 10

	// ZombieDieFrameSpeed 僵尸死亡动画的帧速率（秒/帧）
	ZombieDieFrameSpeed = 0.1

	// ConeheadZombieArmorHealth 路障僵尸护甲值
	// 路障僵尸拥有370点护甲，护甲破坏后变为普通僵尸
	ConeheadZombieArmorHealth = 370

	// ConeheadZombieWalkAnimationFrames 路障僵尸走路动画帧数
	ConeheadZombieWalkAnimationFrames = 21

	// ConeheadZombieEatAnimationFrames 路障僵尸啃食动画帧数
	ConeheadZombieEatAnimationFrames = 11

	// BucketheadZombieArmorHealth 铁桶僵尸护甲值
	// 铁桶僵尸拥有1100点护甲，护甲破坏后变为普通僵尸
	BucketheadZombieArmorHealth = 1100

	// BucketheadZombieWalkAnimationFrames 铁桶僵尸走路动画帧数
	BucketheadZombieWalkAnimationFrames = 15

	// BucketheadZombieEatAnimationFrames 铁桶僵尸啃食动画帧数
	BucketheadZombieEatAnimationFrames = 11
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

// Audio Configuration (音频配置)
const (
	// ZombieHitSoundPath 击中普通僵尸的音效文件路径
	//
	// 🎵 推荐音效选项：
	//   - "assets/audio/Sound/tap.ogg"    - ⭐ 推荐：轻快敲击音（适合普通僵尸）
	//   - "assets/audio/Sound/bleep.mp3"  - 电子音效1（清脆）
	//   - "assets/audio/Sound/bleep1.mp3" - 电子音效2（略低沉）
	//
	// 📝 音效用途说明：
	//   - shieldhit.ogg → 击中铁桶僵尸/路障僵尸的音效（后续实现）
	//   - chomp.ogg → 僵尸啃食植物的音效
	//   - groan.ogg → 僵尸的呻吟声
	//
	// 💡 如何测试不同音效：
	//   1. 修改下面的路径为其他音效文件
	//   2. 运行游戏：go run .
	//   3. 部署豌豆射手攻击僵尸，听音效是否合适
	//   4. 选择最不突兀、最符合游戏风格的音效
	ZombieHitSoundPath = "assets/audio/Sound/tap.ogg"

	// ArmorBreakSoundPath 护甲破坏音效路径
	// 当路障僵尸或铁桶僵尸的护甲被完全破坏时播放
	ArmorBreakSoundPath = "assets/audio/Sound/shieldhit.ogg"

	// PeashooterShootSoundPath 豌豆射手发射子弹的音效文件路径
	//
	// 🎵 可用音效选项：
	//   - "" (空字符串)                        - 不播放音效（静音）
	//   - "assets/audio/Sound/kernelpult2.ogg" - 玉米投手发射音（略重）
	//   - "assets/audio/Sound/tap.ogg"         - 轻敲音（轻快）
	//   - "assets/audio/Sound/buttonclick.ogg" - 点击音（清脆）
	//
	// 📝 说明：
	//   原版游戏中豌豆射手发射音效非常轻微，几乎听不到
	//   如果希望保持原版体验，建议设置为空字符串 ""
	//   如果希望增强反馈感，可以选择轻快的音效
	//
	// 💡 推荐设置：
	//   - 静音模式（原版风格）: ""
	//   - 轻量反馈: "assets/audio/Sound/tap.ogg"
	//   - 明显反馈: "assets/audio/Sound/buttonclick.ogg"
	PeashooterShootSoundPath = "" // 默认：不播放（保持原版风格）
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
	WallnutDefaultHealth = 800 // 原版游戏数值，是向日葵的13倍
)

// Zombie Eating Configuration (僵尸啃食配置)
const (
	// ZombieEatingDamage 僵尸每次啃食造成的伤害
	ZombieEatingDamage = 100

	// ZombieEatingDamageInterval 僵尸啃食伤害间隔（秒）
	// 僵尸每 1.5 秒对植物造成一次伤害
	ZombieEatingDamageInterval = 1.5

	// ZombieEatAnimationFrames 僵尸啃食动画帧数
	// 需要根据实际资源文件确定
	ZombieEatAnimationFrames = 21

	// ZombieEatFrameSpeed 僵尸啃食动画帧速率（秒/帧）
	ZombieEatFrameSpeed = 0.1

	// ZombieEatingSoundPath 僵尸啃食音效路径
	ZombieEatingSoundPath = "assets/audio/Sound/chomp.ogg"
)
