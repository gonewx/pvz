//go:build !mobile

package utils

import "testing"

// TestIsMobile_Desktop 测试桌面端编译时 IsMobile() 返回 false
func TestIsMobile_Desktop(t *testing.T) {
	if IsMobile() {
		t.Error("IsMobile() should return false on desktop")
	}
}
