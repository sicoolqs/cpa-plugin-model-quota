package main

import (
	"encoding/json"
	"testing"
)

// 生命周期通知方法应返回成功信封，避免宿主在启用新方法后收到 unknown_method。
func TestHandleMethodLifecycle(t *testing.T) {
	for _, method := range []string{"plugin.quiesce", "plugin.shutdown"} {
		raw, errHandle := handleMethod(method, nil)
		if errHandle != nil {
			t.Fatalf("%s 返回错误: %v", method, errHandle)
		}
		var envelope struct {
			OK bool `json:"ok"`
		}
		if errUnmarshal := json.Unmarshal(raw, &envelope); errUnmarshal != nil {
			t.Fatalf("%s 信封解析失败: %v", method, errUnmarshal)
		}
		if !envelope.OK {
			t.Errorf("%s 应返回成功信封，得到 %s", method, string(raw))
		}
	}
}
