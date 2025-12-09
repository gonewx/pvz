//go:build !android

package utils

// EnsureStorageDir 确保存储目录存在（非 Android 平台的空实现）
// gdata 在非 Android 平台上会自动创建存储目录，无需额外处理
func EnsureStorageDir() error {
	return nil
}

// GetStoragePath 获取存储路径（非 Android 平台返回空字符串）
func GetStoragePath() string {
	return ""
}
