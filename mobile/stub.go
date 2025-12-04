//go:build !mobile

// stub.go - 非移动端构建时的占位文件
//
// 此文件在普通构建时编译，提供空的 Dummy 函数。
// 实际的移动端代码在 mobile.go 和 embed.go 中，
// 仅在使用 -tags mobile 时编译。
package mobile

// Dummy 是一个空导出函数，确保包在非移动端构建时也能被引用
func Dummy() {}
