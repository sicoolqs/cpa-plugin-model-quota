package main

// hostBridge 通过 C ABI 宿主回调执行 HTTP 请求（host.http.do）。
// 真实请求由宿主执行，代理、传输策略与请求日志沿用内核设置。

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ZRZ-0805/cpa-plugin-model-quota/internal/balance"
)

// hostHTTPRequest 是 host.http.do 的请求体。Headers 值为数组以匹配 http.Header。
// OperationID 由 host.http.operation_open 申请，用于超时后经 host.http.cancel 中止在途请求。
type hostHTTPRequest struct {
	Method      string              `json:"method"`
	URL         string              `json:"url"`
	Headers     map[string][]string `json:"headers,omitempty"`
	OperationID string              `json:"operation_id,omitempty"`
}

// hostHTTPOperationOpenRequest 是 host.http.operation_open 的请求体。
// 共享库插件可省略 host_callback_id，宿主依据调用上下文归属发起插件。
type hostHTTPOperationOpenRequest struct {
	HostCallbackID string `json:"host_callback_id,omitempty"`
}

// hostHTTPOperationOpenResponse 是 host.http.operation_open 的返回结果。
type hostHTTPOperationOpenResponse struct {
	OperationID string `json:"operation_id"`
}

// hostHTTPCancelRequest 是 host.http.cancel 的请求体。
type hostHTTPCancelRequest struct {
	HostCallbackID string `json:"host_callback_id,omitempty"`
	OperationID    string `json:"operation_id"`
}

// hostHTTPResponse 是 host.http.do 的返回体。Body 为原始字节，经 JSON base64。
type hostHTTPResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers,omitempty"`
	Body       []byte              `json:"Body,omitempty"`
}

// hostHTTPDo 调用宿主执行一次 HTTP 请求并解析信封与响应体。
// timeout > 0 时先申请可取消操作，超时后经 host.http.cancel 中止在途请求；
// 宿主不支持 operation_open（旧版本）时退化为不可取消的普通请求。
// 返回的 body 已解码；非 2xx 状态返回携带状态码的错误，由上层归类。
func hostHTTPDo(callHost func(string, []byte) ([]byte, error), request hostHTTPRequest, timeout time.Duration) ([]byte, error) {
	operationID := openHostHTTPOperation(callHost)
	if operationID != "" {
		request.OperationID = operationID
	}
	type httpResult struct {
		body []byte
		err  error
	}
	resultCh := make(chan httpResult, 1)
	go func() {
		body, errSend := sendHostHTTP(callHost, request)
		resultCh <- httpResult{body: body, err: errSend}
	}()
	if timeout <= 0 {
		result := <-resultCh
		return result.body, result.err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-resultCh:
		return result.body, result.err
	case <-timer.C:
		// 中止在途请求（若宿主尚未认领该操作，取消后请求也会被拒绝）。
		cancelHostHTTPOperation(callHost, operationID)
		return nil, fmt.Errorf("上游请求超时（%s）", timeout)
	}
}

// sendHostHTTP 发送 host.http.do 请求并解析信封与响应。
func sendHostHTTP(callHost func(string, []byte) ([]byte, error), request hostHTTPRequest) ([]byte, error) {
	requestBytes, errMarshal := json.Marshal(request)
	if errMarshal != nil {
		return nil, fmt.Errorf("序列化 host.http.do 请求失败: %w", errMarshal)
	}
	resultBytes, errCall := callHost("host.http.do", requestBytes)
	if errCall != nil {
		return nil, fmt.Errorf("宿主 HTTP 回调失败: %w", errCall)
	}
	var envelope struct {
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result,omitempty"`
		Error  *struct {
			Code       string `json:"code"`
			Message    string `json:"message"`
			HTTPStatus int    `json:"http_status,omitempty"`
		} `json:"error,omitempty"`
	}
	if errUnmarshal := json.Unmarshal(resultBytes, &envelope); errUnmarshal != nil {
		return nil, fmt.Errorf("解析宿主回调信封失败: %w", errUnmarshal)
	}
	if !envelope.OK || envelope.Error != nil {
		message := "宿主 HTTP 回调出错"
		if envelope.Error != nil {
			message = fmt.Sprintf("%s: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return nil, fmt.Errorf("%s", message)
	}
	var response hostHTTPResponse
	if errUnmarshal := json.Unmarshal(envelope.Result, &response); errUnmarshal != nil {
		return nil, fmt.Errorf("解析宿主 HTTP 响应失败: %w", errUnmarshal)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, &balance.UpstreamError{
			StatusCode: response.StatusCode,
			Message:    fmt.Sprintf("上游返回 HTTP %d: %s", response.StatusCode, truncate(string(response.Body), 200)),
		}
	}
	return response.Body, nil
}

// openHostHTTPOperation 申请一个可取消的操作 ID。
// 宿主不支持该回调（旧版本内核）或申请失败时返回空串，退化为不可取消的普通请求。
func openHostHTTPOperation(callHost func(string, []byte) ([]byte, error)) string {
	requestBytes, errMarshal := json.Marshal(hostHTTPOperationOpenRequest{})
	if errMarshal != nil {
		return ""
	}
	resultBytes, errCall := callHost("host.http.operation_open", requestBytes)
	if errCall != nil {
		return ""
	}
	var envelope struct {
		OK     bool                           `json:"ok"`
		Result *hostHTTPOperationOpenResponse `json:"result,omitempty"`
	}
	if errUnmarshal := json.Unmarshal(resultBytes, &envelope); errUnmarshal != nil || !envelope.OK || envelope.Result == nil {
		return ""
	}
	return strings.TrimSpace(envelope.Result.OperationID)
}

// cancelHostHTTPOperation 中止一个已申请的操作。失败时静默忽略（请求侧自有兜底）。
func cancelHostHTTPOperation(callHost func(string, []byte) ([]byte, error), operationID string) {
	if strings.TrimSpace(operationID) == "" {
		return
	}
	requestBytes, errMarshal := json.Marshal(hostHTTPCancelRequest{OperationID: operationID})
	if errMarshal != nil {
		return
	}
	_, _ = callHost("host.http.cancel", requestBytes)
}

// truncate 截断过长文本用于错误消息。
func truncate(text string, max int) string {
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	return string(runes[:max]) + "…"
}

// compactJSON 去除 JSON 无关空白（调试与日志用）。
func compactJSON(raw []byte) string {
	var buffer bytes.Buffer
	if errCompact := json.Compact(&buffer, raw); errCompact != nil {
		return string(raw)
	}
	return buffer.String()
}
