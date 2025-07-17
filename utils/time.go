package utils

import (
	"fmt"
	"time"
)

// GetUTCTimestamp 获取当前时间的时间戳（UTC），单位毫秒
func GetUTCTimestamp() int64 {
	currentTime := time.Now().UTC()

	// 获取时间戳 (秒)
	// utcTimestamp := currentTime.Unix()
	// fmt.Println("时间戳 (秒):", utcTimestamp)

	// 获取时间戳 (毫秒)
	utcTimestampMilli := currentTime.UnixMilli()
	fmt.Println("时间戳 (毫秒):", utcTimestampMilli)

	// 获取时间戳 (纳秒)
	// utcTimestampNano := currentTime.UnixNano()
	// fmt.Println("时间戳 (纳秒):", utcTimestampNano)

	return utcTimestampMilli
}

func GetDefTimestamp() int64 {
	return GetTimestamp("Asia/Shanghai")
}

// GetTimestamp 获得指定时区时间的时间戳，单位毫秒
func GetTimestamp(locationName string) int64 {
	// 获取当前时间
	currentTime := time.Now()
	// fmt.Println("当前时间 (本地时区):", currentTime)

	// 指定时区
	location, _ := time.LoadLocation(locationName) // 使用 LoadLocation 加载时区
	localTime := currentTime.In(location)
	// fmt.Println("当前时间:", locationName, beijingTime)

	// 获取时间戳 (秒)
	// localTimestamp := localTime.Unix()
	// fmt.Println("时间戳 (秒):", localTimestamp)

	// 获取时间戳 (毫秒)
	localTimestampMilli := localTime.UnixMilli()
	fmt.Println("时间戳 (毫秒):", localTimestampMilli)

	// 获取时间戳 (纳秒)
	// localTimestampNano := localTime.UnixNano()
	// fmt.Println("时间戳 (纳秒):", localTimestampNano)

	return localTimestampMilli
}
