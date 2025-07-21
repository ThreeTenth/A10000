package utils

import (
	"fmt"
	"math/rand"
	"time"
)

func GenerateTransactionID() string {
	// 生成一个唯一的交易 ID
	// 这里可以使用 UUID 或其他方式生成唯一 ID
	// 简单示例：使用当前时间戳和随机数
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), rand.Int63())
}
