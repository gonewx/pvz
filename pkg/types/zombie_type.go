// Package types 定义共享的基础类型
package types

// ZombieType 定义僵尸的类型
type ZombieType int

const (
	// ZombieUnknown 未知僵尸类型
	ZombieUnknown ZombieType = iota

	// 一阶僵尸（第1波起）
	ZombieBasic    // 普通僵尸
	ZombieConehead // 路障僵尸
	ZombieFlag     // 旗帜僵尸

	// 二阶僵尸（第3波起）
	ZombieBuckethead // 铁桶僵尸
	ZombieNewspaper  // 读报僵尸
	ZombieScreendoor // 铁栅门僵尸

	// 三阶僵尸（第8波起）
	ZombiePolevaulter  // 撑杆跳僵尸
	ZombieFootball     // 橄榄球僵尸
	ZombieDancing      // 舞王僵尸
	ZombieBackupDancer // 伴舞僵尸
	ZombieSnorkel      // 潜水僵尸
	ZombieDolphinRider // 海豚骑士僵尸
	ZombieDucky        // 鸭子救生圈僵尸
	ZombieJack         // 小丑僵尸
	ZombieBalloon      // 气球僵尸
	ZombieDigger       // 矿工僵尸
	ZombiePogo         // 蹦蹦僵尸
	ZombieZomboni      // 雪橇车僵尸
	ZombieBobsled      // 雪橇队僵尸
	ZombieBungee       // 蹦极僵尸
	ZombieLadder       // 扶梯僵尸
	ZombieCatapult     // 投石车僵尸
	ZombieYeti         // 雪人僵尸

	// 四阶僵尸（第15波起）
	ZombieGargantuar       // 白眼巨人
	ZombieGargantuarRedeye // 红眼巨人
	ZombieImp              // 小鬼僵尸

	// Boss
	ZombieDrZomboss // 僵王博士
)

// UnitID 常量 - 用于 AnimationCommand 组件指定僵尸动画
const (
	UnitIDZombie           = "zombie"
	UnitIDZombieConehead   = "zombie_conehead"
	UnitIDZombieBuckethead = "zombie_buckethead"
	UnitIDZombieFlag       = "zombie_flag"
	UnitIDZombieNewspaper  = "zombie_newspaper"
	UnitIDZombieScreendoor = "zombie_screendoor"
	UnitIDZombiePolevaulter = "zombie_polevaulter"
	UnitIDZombieFootball   = "zombie_football"
	UnitIDZombieDancing    = "zombie_dancing"
	UnitIDZombieBackupDancer = "zombie_backup_dancer"
	UnitIDZombieSnorkel    = "zombie_snorkel"
	UnitIDZombieDolphinRider = "zombie_dolphinrider"
	UnitIDZombieDucky      = "zombie_ducky"
	UnitIDZombieJack       = "zombie_jack"
	UnitIDZombieBalloon    = "zombie_balloon"
	UnitIDZombieDigger     = "zombie_digger"
	UnitIDZombiePogo       = "zombie_pogo"
	UnitIDZombieZomboni    = "zombie_zomboni"
	UnitIDZombieBobsled    = "zombie_bobsled"
	UnitIDZombieBungee     = "zombie_bungee"
	UnitIDZombieLadder     = "zombie_ladder"
	UnitIDZombieCatapult   = "zombie_catapult"
	UnitIDZombieYeti       = "zombie_yeti"
	UnitIDZombieGargantuar = "zombie_gargantuar"
	UnitIDZombieGargantuarRedeye = "zombie_gargantuar_redeye"
	UnitIDZombieImp        = "zombie_imp"
	UnitIDZombieDrZomboss  = "zombie_drzomboss"

	// 特殊状态 UnitID
	UnitIDZombieCharred = "zombie_charred" // 烧焦状态
	UnitIDZombiesWon    = "zombieswon"     // 僵尸胜利动画
)

// zombieTypeStringMap 僵尸类型到配置字符串的映射
var zombieTypeStringMap = map[ZombieType]string{
	ZombieBasic:            "basic",
	ZombieConehead:         "conehead",
	ZombieFlag:             "flag",
	ZombieBuckethead:       "buckethead",
	ZombieNewspaper:        "newspaper",
	ZombieScreendoor:       "screendoor",
	ZombiePolevaulter:      "polevaulter",
	ZombieFootball:         "football",
	ZombieDancing:          "dancing",
	ZombieBackupDancer:     "backup_dancer",
	ZombieSnorkel:          "snorkel",
	ZombieDolphinRider:     "dolphinrider",
	ZombieDucky:            "ducky",
	ZombieJack:             "jack",
	ZombieBalloon:          "balloon",
	ZombieDigger:           "digger",
	ZombiePogo:             "pogo",
	ZombieZomboni:          "zomboni",
	ZombieBobsled:          "bobsled",
	ZombieBungee:           "bungee",
	ZombieLadder:           "ladder",
	ZombieCatapult:         "catapult",
	ZombieYeti:             "yeti",
	ZombieGargantuar:       "gargantuar",
	ZombieGargantuarRedeye: "gargantuar_redeye",
	ZombieImp:              "imp",
	ZombieDrZomboss:        "drzomboss",
}

// zombieTypeUnitIDMap 僵尸类型到动画 UnitID 的映射
var zombieTypeUnitIDMap = map[ZombieType]string{
	ZombieBasic:            UnitIDZombie,
	ZombieConehead:         UnitIDZombieConehead,
	ZombieBuckethead:       UnitIDZombieBuckethead,
	ZombieFlag:             UnitIDZombieFlag,
	ZombieNewspaper:        UnitIDZombieNewspaper,
	ZombieScreendoor:       UnitIDZombieScreendoor,
	ZombiePolevaulter:      UnitIDZombiePolevaulter,
	ZombieFootball:         UnitIDZombieFootball,
	ZombieDancing:          UnitIDZombieDancing,
	ZombieBackupDancer:     UnitIDZombieBackupDancer,
	ZombieSnorkel:          UnitIDZombieSnorkel,
	ZombieDolphinRider:     UnitIDZombieDolphinRider,
	ZombieDucky:            UnitIDZombieDucky,
	ZombieJack:             UnitIDZombieJack,
	ZombieBalloon:          UnitIDZombieBalloon,
	ZombieDigger:           UnitIDZombieDigger,
	ZombiePogo:             UnitIDZombiePogo,
	ZombieZomboni:          UnitIDZombieZomboni,
	ZombieBobsled:          UnitIDZombieBobsled,
	ZombieBungee:           UnitIDZombieBungee,
	ZombieLadder:           UnitIDZombieLadder,
	ZombieCatapult:         UnitIDZombieCatapult,
	ZombieYeti:             UnitIDZombieYeti,
	ZombieGargantuar:       UnitIDZombieGargantuar,
	ZombieGargantuarRedeye: UnitIDZombieGargantuarRedeye,
	ZombieImp:              UnitIDZombieImp,
	ZombieDrZomboss:        UnitIDZombieDrZomboss,
}

// stringToZombieTypeMap 配置字符串到僵尸类型的反向映射
var stringToZombieTypeMap map[string]ZombieType

func init() {
	stringToZombieTypeMap = make(map[string]ZombieType)
	for zt, s := range zombieTypeStringMap {
		stringToZombieTypeMap[s] = zt
	}
	// 添加别名映射（处理历史命名不一致）
	stringToZombieTypeMap["newspaperzombie"] = ZombieNewspaper
	stringToZombieTypeMap["footballzombie"] = ZombieFootball
	stringToZombieTypeMap["dancingzombie"] = ZombieDancing
	stringToZombieTypeMap["backupdancer"] = ZombieBackupDancer
	stringToZombieTypeMap["jackinthebox"] = ZombieJack
}

// String 返回僵尸类型的配置字符串表示（用于配置文件匹配）
func (z ZombieType) String() string {
	if s, ok := zombieTypeStringMap[z]; ok {
		return s
	}
	return "unknown"
}

// UnitID 返回僵尸类型对应的动画 UnitID
// UnitID 用于 AnimationCommand 组件，以正确显示僵尸的装备外观
func (z ZombieType) UnitID() string {
	if id, ok := zombieTypeUnitIDMap[z]; ok {
		return id
	}
	return UnitIDZombie // 默认返回基础僵尸
}

// ZombieTypeFromString 将配置字符串转换为 ZombieType
// 支持标准名称和历史别名
func ZombieTypeFromString(s string) ZombieType {
	if zt, ok := stringToZombieTypeMap[s]; ok {
		return zt
	}
	return ZombieUnknown
}

// ZombieTypeToUnitID 将僵尸类型字符串转换为动画 UnitID
// 这是一个便捷函数，用于从字符串直接获取 UnitID
func ZombieTypeToUnitID(zombieType string) string {
	zt := ZombieTypeFromString(zombieType)
	return zt.UnitID()
}

// IsWaterZombie 判断是否为水路专属僵尸
func (z ZombieType) IsWaterZombie() bool {
	switch z {
	case ZombieSnorkel, ZombieDolphinRider, ZombieDucky:
		return true
	default:
		return false
	}
}

// Tier 返回僵尸的阶数（1-4）
func (z ZombieType) Tier() int {
	switch z {
	case ZombieBasic, ZombieConehead, ZombieFlag:
		return 1
	case ZombieBuckethead, ZombieNewspaper, ZombieScreendoor:
		return 2
	case ZombiePolevaulter, ZombieFootball, ZombieDancing, ZombieBackupDancer,
		ZombieSnorkel, ZombieDolphinRider, ZombieDucky, ZombieJack,
		ZombieBalloon, ZombieDigger, ZombiePogo, ZombieZomboni,
		ZombieBobsled, ZombieBungee, ZombieLadder, ZombieCatapult, ZombieYeti:
		return 3
	case ZombieGargantuar, ZombieGargantuarRedeye, ZombieImp:
		return 4
	default:
		return 0
	}
}
