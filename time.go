package main

import (
	"math/rand"
	"time"

	log "github.com/tengfei-xy/go-log"
)

// 随机挂起 x 秒，并添加随机波动
func sleep(i int) {
	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())
	
	// 计算随机波动值 (±20%)
	variation := float64(i) * 0.2
	randomVariation := rand.Float64()*variation*2 - variation
	
	// 最终延迟时间（秒）
	finalDelay := float64(i) + randomVariation
	
	// 确保延迟时间为正数
	if finalDelay < 1 {
		finalDelay = 1
	}
	
	// 转换为毫秒并加入额外的微小随机延迟（0-500毫秒）
	delayMs := int(finalDelay*1000) + rand.Intn(500)
	
	log.Infof("模拟人类行为，挂起%.2f秒", float64(delayMs)/1000)
	time.Sleep(time.Duration(delayMs) * time.Millisecond)
}
