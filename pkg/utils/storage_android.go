//go:build android

package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureStorageDir 确保 Android 存储目录存在并可写
// gdata 库在 Android 上使用 /data/data/{package}/ 作为存储路径，
// 但不会预先创建子目录。此函数在 gdata 初始化前调用，
// 确保 saves 目录存在且可写。
//
// 返回：
//   - error: 如果创建目录失败返回错误
func EnsureStorageDir() error {
	// 检测 Android 应用包名
	app, err := detectAndroidApp()
	if err != nil {
		return fmt.Errorf("failed to detect Android app: %w", err)
	}

	// 构建存储路径: /data/data/{package}/saves
	savesDir := filepath.Join("/data/data", app, "saves")

	// 创建目录（如果不存在）
	if err := os.MkdirAll(savesDir, 0755); err != nil {
		return fmt.Errorf("failed to create saves directory %s: %w", savesDir, err)
	}

	// 验证目录可写
	testFile := filepath.Join(savesDir, ".write_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("saves directory %s is not writable: %w", savesDir, err)
	}
	os.Remove(testFile)

	return nil
}

// detectAndroidApp 检测 Android 应用包名
// 从 /proc/self/cmdline 读取应用标识符
func detectAndroidApp() (string, error) {
	data, err := os.ReadFile("/proc/self/cmdline")
	if err != nil {
		return "", err
	}

	// 移除 null 字节和换行符
	copied := make([]byte, 0, len(data))
	for _, ch := range data {
		switch ch {
		case 0, '\n':
			continue
		}
		copied = append(copied, ch)
	}

	result := string(copied)
	if result == "" {
		return "", fmt.Errorf("got empty output from /proc/self/cmdline")
	}

	return result, nil
}

// GetStoragePath 获取 Android 存储路径（用于调试）
func GetStoragePath() string {
	app, err := detectAndroidApp()
	if err != nil {
		return ""
	}
	return filepath.Join("/data/data", app)
}
