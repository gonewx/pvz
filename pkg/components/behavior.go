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
	// BehaviorZombieSquashing 僵尸被压扁中：播放除草车碾压动画（位移、旋转、缩放）
	// 动画完成后切换为 BehaviorZombieDying，触发粒子效果并删除实体
	BehaviorZombieSquashing
	// BehaviorZombieDyingExplosion 僵尸死亡中（爆炸烧焦死亡动画）
	//
	// 使用场景：僵尸被爆炸类攻击（樱桃炸弹、土豆雷、辣椒等）杀死时
	//
	// 与其他死亡类型的区别：
	//   - BehaviorZombieDying: 普通死亡（豌豆、坚果等）- 头部掉落动画
	//   - BehaviorZombieSquashing: 压扁死亡（除草车）- 铲起旋转压扁动画
	//   - BehaviorZombieDyingExplosion: 爆炸死亡 - 烧焦黑化动画
	//
	// 动画资源：使用 Zombie_charred.reanim（烧焦僵尸骨骼动画），非循环
	//
	// 参考实现：Story 10.6 (压扁动画)
	BehaviorZombieDyingExplosion
	// BehaviorWallnut 坚果墙行为：无攻击能力的纯防御植物，根据生命值百分比切换外观状态
	// 坚果墙拥有极高的生命值(4000)，用于阻挡僵尸前进
	// 外观状态：完好(>66%) → 轻伤(33-66%) → 重伤(<33%)
	BehaviorWallnut
	// BehaviorZombieConehead 路障僵尸行为：带护甲的僵尸，拥有额外的防护层(370护甲值)
	// 当护甲被完全破坏后，外观切换为普通僵尸，行为转变为 BehaviorZombieBasic
	BehaviorZombieConehead
	// BehaviorZombieBuckethead 铁桶僵尸行为：带高强度护甲的僵尸(1100护甲值)
	// 当护甲被完全破坏后，外观切换为普通僵尸，行为转变为 BehaviorZombieBasic
	BehaviorZombieBuckethead
	// BehaviorZombieFlag 旗帜僵尸行为：与普通僵尸行为相同，但外观不同
	// 旗帜僵尸在旗帜波出现，标志着大量僵尸即将来袭
	BehaviorZombieFlag
	// BehaviorFallingPart 掉落部件效果：僵尸手臂或头部掉落的动画效果
	// 部件以抛物线轨迹飞出，一段时间后消失
	BehaviorFallingPart
	// BehaviorCherryBomb 樱桃炸弹行为：种植后进入引信倒计时，倒计时结束后爆炸
	// 爆炸对以自身为中心的3x3范围内的所有僵尸造成1800点伤害（足以秒杀所有僵尸）
	// 爆炸后樱桃炸弹实体被移除
	BehaviorCherryBomb
	// BehaviorZombiePreview 僵尸预告行为：开场动画中的僵尸预览，不移动、不攻击、只播放 idle 动画
	BehaviorZombiePreview
)

// ZombieAnimState 定义僵尸的动画状态
type ZombieAnimState int

const (
	// ZombieAnimIdle 待机/静止状态（用于开场预览）
	ZombieAnimIdle ZombieAnimState = iota
	// ZombieAnimWalking 行走状态
	ZombieAnimWalking
	// ZombieAnimEating 啃食/攻击状态
	ZombieAnimEating
	// ZombieAnimDying 死亡状态
	ZombieAnimDying
)

// BehaviorComponent 标识实体的行为类型
// 此组件用于让 BehaviorSystem 识别实体应执行何种行为逻辑
type BehaviorComponent struct {
	Type            BehaviorType    // 行为类型（向日葵、豌豆射手等）
	ZombieAnimState ZombieAnimState // 僵尸当前动画状态（仅用于僵尸）
	UnitID          string          // 动画配置 ID（僵尸专用，如 "zombie_flag"）
}
