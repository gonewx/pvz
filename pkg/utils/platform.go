//go:build !mobile

package utils

import "os"

// IsMobile 检测当前是否在移动设备上运行
// 桌面端编译时返回 false
// 可以通过设置环境变量 PVZ_MOBILE_EMULATE=1 强制启用移动模式（用于本地调试）
func IsMobile() bool {
	return os.Getenv("PVZ_MOBILE_EMULATE") == "1"
}
