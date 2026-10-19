package main

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestHostHTTPDoSuccessCarriesOperationID(t *testing.T) {
	var requestPayload string
	callHost := func(method string, payload []byte) ([]byte, error) {
		switch method {
		case "host.http.operation_open":
			return []byte(`{"ok":true,"result":{"operation_id":"op-1"}}`), nil
		case "host.http.do":
			requestPayload = string(payload)
			return []byte(`{"ok":true,"result":{"StatusCode":200,"Headers":{},"Body":"aGk="}}`), nil
		default:
			return nil, fmt.Errorf("unexpected method %s", method)
		}
	}
	body, err := hostHTTPDo(callHost, hostHTTPRequest{Method: "GET", URL: "https://example.com"}, time.Second)
	if err != nil {
		t.Fatalf("请求不应失败: %v", err)
	}
	if string(body) != "hi" {
		t.Errorf("响应体应为 hi，得到 %q", string(body))
	}
	if !strings.Contains(requestPayload, `"operation_id":"op-1"`) {
		t.Errorf("请求应携带 operation_id，得到 %s", requestPayload)
	}
}

func TestHostHTTPDoTimeoutCancelsOperation(t *testing.T) {
	var cancelled int32
	callHost := func(method string, payload []byte) ([]byte, error) {
		switch method {
		case "host.http.operation_open":
			return []byte(`{"ok":true,"result":{"operation_id":"op-1"}}`), nil
		case "host.http.do":
			time.Sleep(300 * time.Millisecond)
			return []byte(`{"ok":true,"result":{"StatusCode":200,"Headers":{},"Body":"aGk="}}`), nil
		case "host.http.cancel":
			atomic.AddInt32(&cancelled, 1)
			return []byte(`{"ok":true,"result":{}}`), nil
		default:
			return nil, fmt.Errorf("unexpected method %s", method)
		}
	}
	start := time.Now()
	_, err := hostHTTPDo(callHost, hostHTTPRequest{Method: "GET", URL: "https://example.com"}, 30*time.Millisecond)
	if err == nil {
		t.Fatal("超时应返回错误")
	}
	if !strings.Contains(err.Error(), "超时") {
		t.Errorf("错误应说明超时，得到 %q", err.Error())
	}
	if atomic.LoadInt32(&cancelled) == 0 {
		t.Error("超时后应调用 host.http.cancel")
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Errorf("超时未及时返回: %s", elapsed)
	}
}

func TestHostHTTPDoFallbackWithoutOperationSupport(t *testing.T) {
	callHost := func(method string, payload []byte) ([]byte, error) {
		switch method {
		case "host.http.operation_open":
			return nil, fmt.Errorf("unsupported host callback")
		case "host.http.do":
			if strings.Contains(string(payload), "operation_id") {
				t.Error("降级场景不应携带 operation_id")
			}
			return []byte(`{"ok":true,"result":{"StatusCode":200,"Headers":{},"Body":"aGk="}}`), nil
		default:
			return nil, fmt.Errorf("unexpected method %s", method)
		}
	}
	body, err := hostHTTPDo(callHost, hostHTTPRequest{Method: "GET", URL: "https://example.com"}, time.Second)
	if err != nil {
		t.Fatalf("降级场景请求不应失败: %v", err)
	}
	if string(body) != "hi" {
		t.Errorf("响应体应为 hi，得到 %q", string(body))
	}
}

func TestHandleQuiesce(t *testing.T) {
	raw, errHandle := handleMethod("plugin.quiesce", nil)
	if errHandle != nil {
		t.Fatalf("plugin.quiesce 不应返回错误: %v", errHandle)
	}
	if !strings.Contains(string(raw), `"ok":true`) {
		t.Errorf("plugin.quiesce 应返回成功信封，得到 %s", raw)
	}
}
