package components

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// LevelProgressBarComponent 关卡进度条组件（纯数据，无方法）
//
// Story 11.5: 实现原版 PvZ 的进度条机制：
// - 双段式结构：红字波段（每波12格）+ 普通波段（平均分配）
// - 双重进度计算：max(时间进度, 血量削减进度)
// - 虚拟/现实双层追踪：平滑动画效果
type LevelProgressBarComponent struct {
	// 资源引用
	BackgroundImage  *ebiten.Image // FlagMeter.png - 进度条背景框
	ProgressBarImage *ebiten.Image // FlagMeterLevelProgress.png - 绿色进度填充条
	PartsImage       *ebiten.Image // FlagMeterParts.png - 精灵图（包含旗帜和僵尸头图标）

	// 旗帜配置
	FlagPositions []float64 // 旗帜在进度条上的位置百分比列表

	// 显示配置
	LevelText         string // 关卡文本（如"关卡 1-1"）
	ShowLevelTextOnly bool   // 是否只显示文本（第一波前 = true）

	// 位置配置（屏幕坐标）
	X float64 // 进度条X坐标（屏幕右下角）
	Y float64 // 进度条Y坐标（屏幕右下角）

	// === Story 11.5: 原版进度条机制字段 ===

	// 进度条结构配置
	TotalProgressLength int // 进度条总长度（默认 150）
	FlagSegmentLength   int // 红字波段长度（每波 12）
	NormalSegmentBase   int // 普通波段基础长度（计算得出）

	// 波次追踪
	TotalWaves     int // 总波次数
	CurrentWaveNum int // 当前波次号（从 1 开始）
	FlagWaveCount  int // 已完成的红字波数量

	// 时间进度追踪
	WaveStartTime    float64 // 本波开始时间（游戏时间，秒）
	WaveInitialDelay float64 // 本波初始刷新倒计时（秒）

	// 血量削减追踪
	WaveInitialHealth  float64 // 本波僵尸初始总血量
	WaveCurrentHealth  float64 // 本波僵尸当前总血量
	WaveRequiredDamage float64 // 本波激活所需削减血量

	// 虚拟/现实进度
	VirtualProgress float64 // 虚拟进度条值（可超过 1.0）
	RealProgress    float64 // 现实进度条值（平滑追踪，用于渲染）

	// 游戏时钟
	GameTickCS       int     // 游戏时钟（厘秒，centiseconds，1cs = 0.01秒）
	LastTrackUpdateCS int    // 上次追踪更新的游戏时钟值

	// === 废弃字段（保留向后兼容） ===
	// @Deprecated: 使用 VirtualProgress/RealProgress 替代
	TotalZombies    int     // 总僵尸数（从关卡配置计算）
	KilledZombies   int     // 已击杀僵尸数
	ProgressPercent float64 // 进度百分比 (0.0 - 1.0)，现在作为 RealProgress 的别名
}
