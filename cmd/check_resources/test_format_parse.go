package main

import (
	"fmt"
	particlePkg "github.com/decker502/pvz/internal/particle"
)

func main() {
	// 测试不同格式的解析
	testCases := []string{
		"1,95 0",
		"1,79.99999 0",
		".9,79.99999 0",
		"0 1,9.999999 1,45 0,55",
	}
	
	for _, tc := range testCases {
		fmt.Printf("输入: '%s'\n", tc)
		min, max, keys, interp := particlePkg.ParseValue(tc)
		fmt.Printf("  min=%.2f, max=%.2f, interp='%s'\n", min, max, interp)
		fmt.Printf("  关键帧数: %d\n", len(keys))
		for i, kf := range keys {
			fmt.Printf("    [%d] time=%.4f (%.2f%%), value=%.4f\n", i, kf.Time, kf.Time*100, kf.Value)
		}
		fmt.Println()
	}
}
