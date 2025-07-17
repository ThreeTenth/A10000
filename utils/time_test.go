package utils_test

import (
	"a10000/utils"
	"testing"
)

func TestGetUTCTimestamp(t *testing.T) {
	ts := utils.GetUTCTimestamp()
	t.Log(ts)
}

func TestGetDefTimestamp(t *testing.T) {
	ts := utils.GetDefTimestamp()
	ts1 := utils.GetTimestamp("Asia/Shanghai")
	if ts != ts1 {
		t.Fail()
	}
}
