package main

import (
	"time"

	"github.com/ZRZ-0805/cpa-plugin-model-quota/internal/balance"
)

// hostClient 把 C ABI 宿主回调（见 main.go 的 callHostMethod）封装为 balance.HostClient。
type hostClient struct{}

// DoHTTP 通过宿主回调执行 GET 请求，并把上游状态码错误转换为 balance.UpstreamError。
func (hostClient) DoHTTP(url string, headers map[string][]string, timeout time.Duration) ([]byte, error) {
	body, errDo := hostHTTPDo(callHostMethod, hostHTTPRequest{
		Method:  "GET",
		URL:     url,
		Headers: headers,
	}, timeout)
	if errDo != nil {
		// 保留上游状态码错误，供信封映射 http_status。
		var upstream *balance.UpstreamError
		if errorsAs(errDo, &upstream) {
			return nil, upstream
		}
		return nil, errDo
	}
	return body, nil
}

// errorsAs 是标准库 errors.As 的轻量替身，避免为单一类型判断引入额外依赖路径。
func errorsAs(err error, target **balance.UpstreamError) bool {
	for err != nil {
		if upstream, ok := err.(*balance.UpstreamError); ok {
			*target = upstream
			return true
		}
		unwrapper, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapper.Unwrap()
	}
	return false
}
