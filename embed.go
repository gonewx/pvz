// embed.go - 资源嵌入声明
// 必须放在项目根目录（与 assets/ 和 data/ 同级）
// 因为 //go:embed 指令只能嵌入当前包目录及其子目录的文件
package main

import "embed"

//go:embed all:assets
var assetsFS embed.FS

//go:embed data/reanim data/reanim_config data/levels data/particles
//go:embed data/reanim_config.yaml data/spawn_rules.yaml data/zombie_physics.yaml data/zombie_stats.yaml
var dataFS embed.FS

