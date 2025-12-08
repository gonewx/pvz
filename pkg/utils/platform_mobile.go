//go:build mobile

package utils

// IsMobile 检测当前是否在移动设备上运行
// 移动端编译时返回 true
func IsMobile() bool {
	return true
}
