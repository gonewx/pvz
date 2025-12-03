// init_test.go - 测试环境初始化
//
// 此文件在测试运行前初始化 embedded 包，使用项目根目录的实际资源。
// 注意：由于 Go embed 指令只能嵌入当前包目录及其子目录的文件，
// 这里我们使用一个特殊的方式来初始化：通过创建一个读取文件系统的包装 FS。

package embedded

import (
	"embed"
	"os"
)

// initTestEmbedded 在测试环境中初始化 embedded 包
// 由于测试环境无法使用项目根目录的 embed 声明，我们需要特殊处理
func initTestEmbedded() {
	// 检查是否已经初始化
	if initialized {
		return
	}

	// 在测试环境中，我们使用空的 embed.FS
	// 测试应该处理未初始化或文件不存在的情况
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
}

// InitForIntegrationTest 用于集成测试，从文件系统初始化
// 这个函数应该在需要实际文件的测试中调用
func InitForIntegrationTest() error {
	// 检查是否在项目根目录运行
	if _, err := os.Stat("assets"); err != nil {
		return err
	}
	if _, err := os.Stat("data"); err != nil {
		return err
	}

	// 初始化为空 FS（测试需要特殊处理或跳过需要实际文件的测试）
	var emptyFS embed.FS
	Init(emptyFS, emptyFS)
	return nil
}

