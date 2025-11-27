package main

import (
    "fmt"
    "pvz3/pkg/config"
)

func main() {
    levels := []string{
        "data/levels/level-1-1.yaml",
        "data/levels/level-1-2.yaml",
        "data/levels/level-1-3.yaml",
        "data/levels/level-1-4.yaml",
    }
    
    for _, path := range levels {
        cfg, err := config.LoadLevelConfig(path)
        if err != nil {
            fmt.Printf("FAIL: %s - %v\n", path, err)
        } else {
            fmt.Printf("OK: %s - ID=%s, SceneType=%s, RowMax=%d, Flags=%d, Waves=%d\n", 
                path, cfg.ID, cfg.SceneType, cfg.RowMax, cfg.Flags, len(cfg.Waves))
            if len(cfg.Waves) > 0 {
                fmt.Printf("     Wave 1: WaveNum=%d, Type=%s\n", cfg.Waves[0].WaveNum, cfg.Waves[0].Type)
            }
        }
    }
}
