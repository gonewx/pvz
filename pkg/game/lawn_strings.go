package game

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/decker502/pvz/pkg/embedded"
)

// LawnStrings 游戏文本字符串管理器
// 从 LawnStrings.txt 加载本地化文本，支持通过键快速查询
type LawnStrings struct {
	strings map[string]string // 键 -> 文本映射
}

// NewLawnStrings 从文件加载游戏文本字符串
// 参数：
//   - filePath: LawnStrings.txt 文件路径（通常为 "assets/properties/LawnStrings.txt"）
//
// 返回：
//   - *LawnStrings: 字符串管理器实例
//   - error: 如果文件读取或解析失败
//
// 文件格式：
//
//	[KEY]
//	文本内容
//
// 示例：
//
//	[ADVICE_CLICK_ON_SUN]
//	点击收集掉落的阳光！
func NewLawnStrings(filePath string) (*LawnStrings, error) {
	// 从 embedded FS 打开文件
	file, err := embedded.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open LawnStrings file %s: %w", filePath, err)
	}
	defer file.Close()

	// 初始化字符串映射表
	ls := &LawnStrings{
		strings: make(map[string]string),
	}

	// 逐行读取并解析
	scanner := bufio.NewScanner(file)
	var currentKey string
	for scanner.Scan() {
		line := scanner.Text()

		// 跳过空行
		if strings.TrimSpace(line) == "" {
			continue
		}

		// 检查是否为键定义（格式：[KEY]）
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			// 提取键名（去掉方括号）
			currentKey = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		// 如果有当前键，则将该行作为值存储
		if currentKey != "" {
			ls.strings[currentKey] = line
			currentKey = "" // 重置键，准备读取下一个键值对
		}
	}

	// 检查扫描错误
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read LawnStrings file %s: %w", filePath, err)
	}

	return ls, nil
}

// GetString 根据键获取文本字符串
// 参数：
//   - key: 文本键（如 "ADVICE_CLICK_ON_SUN"）
//
// 返回：
//   - string: 对应的文本内容，如果键不存在则返回 "[key]"（用于调试）
//
// 示例：
//
//	text := lawnStrings.GetString("ADVICE_CLICK_ON_SUN")
//	// 返回："点击收集掉落的阳光！"
func (ls *LawnStrings) GetString(key string) string {
	if text, ok := ls.strings[key]; ok {
		return text
	}
	// 键不存在时返回带方括号的键名（调试用）
	return "[" + key + "]"
}
