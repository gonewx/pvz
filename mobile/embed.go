//go:build mobile

// embed.go - 移动端资源嵌入声明
//
// 此文件仅在使用 -tags mobile 构建时编译。
// Makefile 中的 build-android 和 build-ios 目标会自动：
//   1. 运行 prepare-mobile 复制资源到此目录
//   2. 使用 -tags mobile 进行构建
//
// 手动构建：
//
//	make prepare-mobile
//	go build -tags mobile ./mobile
package mobile

import "embed"

//go:embed all:assets
var assetsFS embed.FS

//go:embed data/reanim data/reanim_config data/levels data/particles
//go:embed data/reanim_config.yaml data/spawn_rules.yaml data/zombie_physics.yaml data/zombie_stats.yaml
var dataFS embed.FS
