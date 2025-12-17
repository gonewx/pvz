// Package config 提供游戏配置管理
package config

// 实体 Reanim 资源名称常量
// 这些实体是唯一的（不需要注册表），但资源名称应集中管理
// 新增实体或修改资源名称时，只需修改此文件

// 阳光实体
const (
	// ReanimNameSun 阳光动画资源名称
	// 对应文件：data/reanim/Sun.reanim
	ReanimNameSun = "Sun"
)

// 除草车实体
const (
	// ReanimNameLawnMower 除草车动画资源名称
	// 对应文件：data/reanim/LawnMower.reanim
	ReanimNameLawnMower = "LawnMower"
)

// 疯狂戴夫实体
const (
	// ReanimNameCrazyDave 疯狂戴夫动画资源名称
	// 对应文件：data/reanim/CrazyDave.reanim
	ReanimNameCrazyDave = "CrazyDave"
)

// 最终波警告实体
const (
	// ReanimNameFinalWave 最终波警告动画资源名称
	// 对应文件：data/reanim/FinalWave.reanim
	ReanimNameFinalWave = "FinalWave"
)

// 选卡界面实体
const (
	// ReanimNameSelectorScreen 选卡界面动画资源名称
	// 对应文件：data/reanim/SelectorScreen.reanim
	ReanimNameSelectorScreen = "SelectorScreen"
)

// 僵尸手实体（主菜单过渡动画）
const (
	// ReanimNameZombieHand 僵尸手动画资源名称
	// 对应文件：data/reanim/Zombie_hand.reanim
	ReanimNameZombieHand = "Zombie_hand"
)

// 旗帜僵尸旗杆（特殊子动画资源）
const (
	// ReanimNameZombieFlagPole 旗帜僵尸旗杆动画资源名称
	// 对应文件：data/reanim/Zombie_FlagPole.reanim
	// 用于旗帜僵尸的轨道合并
	ReanimNameZombieFlagPole = "Zombie_FlagPole"
)
