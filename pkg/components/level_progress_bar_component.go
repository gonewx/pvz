package components

import (
	"github.com/hajimehoshi/ebiten/v2"
)

// LevelProgressBarComponent 关卡进度条组件（纯数据，无方法）
type LevelProgressBarComponent struct {
	// 资源引用
	BackgroundImage  *ebiten.Image // FlagMeter.png - 进度条背景框
	ProgressBarImage *ebiten.Image // FlagMeterLevelProgress.png - 绿色进度填充条
	PartsImage       *ebiten.Image // FlagMeterParts.png - 精灵图（包含旗帜和僵尸头图标）

	// 进度数据
	TotalZombies    int     // 总僵尸数（从关卡配置计算）
	KilledZombies   int     // 已击杀僵尸数
	ProgressPercent float64 // 进度百分比 (0.0 - 1.0)

	// 旗帜配置
	FlagPositions []float64 // 旗帜在进度条上的位置百分比列表

	// 显示配置
	LevelText         string // 关卡文本（如"关卡 1-1"）
	ShowLevelTextOnly bool   // 是否只显示文本（第一波前 = true）

	// 位置配置（屏幕坐标）
	X float64 // 进度条X坐标（屏幕右下角）
	Y float64 // 进度条Y坐标（屏幕右下角）
}
