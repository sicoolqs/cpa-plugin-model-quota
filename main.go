// model-quota 是 CLIProxyAPI 的原生动态库插件：
// 实时查询智谱 GLM Coding Plan 用量与 DeepSeek 账户余额，并通过插件自带
// Management API 资源页面展示。请求统一经 host.http.do 宿主回调发出。
package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);

static const cliproxy_host_api* stored_host;

static void store_host_api(const cliproxy_host_api* host) {
	stored_host = host;
}

static int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
	if (stored_host == NULL || stored_host->call == NULL) {
		return 1;
	}
	return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}

static void free_host_buffer(void* ptr, size_t len) {
	if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) {
		stored_host->free_buffer(ptr, len);
	}
}
*/
import "C"

import (
	"embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/ZRZ-0805/cpa-plugin-model-quota/internal/balance"
	"github.com/ZRZ-0805/cpa-plugin-model-quota/internal/config"
)

//go:embed resources/index.html
var resourceFS embed.FS

const (
	pluginID      = "model-quota"
	pluginVersion = "0.3.0"
	// panelPath 作为静态资源路由注册 (/v0/resource/plugins/model-quota/panel)：仅提供无状态静态 UI。
	// queryPath 作为受保护的管理路由注册 (/v0/management/plugins/model-quota/query)：强制要求管理密钥鉴权。
	queryPath = "/plugins/model-quota/query"
	panelPath = "/panel"
)

// store 保存最近一次配置与缓存结果，management.handle 间共享。
type store struct {
	mu       sync.RWMutex
	cfg      config.Config
	lastHost func(string, []byte) ([]byte, error)
}

var pluginStore = &store{}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	C.store_host_api(host)
	plugin.abi_version = C.uint32_t(1)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, errHandle := handleMethod(C.GoString(method), requestBytes)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, len C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
	_ = len
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case "plugin.register", "plugin.reconfigure":
		return handleRegister(request)
	case "management.register":
		return okEnvelope(map[string]any{
			"routes": []map[string]any{
				{
					"Method":      "GET",
					"Path":        queryPath,
					"Description": "查询各模型额度与余额（受管理密钥保护）",
				},
			},
			"resources": []map[string]any{
				{
					"Path":        panelPath,
					"Menu":        "额度查询",
					"Description": "大模型额度与余额监控面板（静态UI）",
				},
			},
		})
	case "management.handle":
		return handleManagement(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func handleRegister(request []byte) ([]byte, error) {
	var lifecycle struct {
		ConfigYAML []byte `json:"config_yaml"`
	}
	if len(request) > 0 {
		if errUnmarshal := json.Unmarshal(request, &lifecycle); errUnmarshal != nil {
			return errorEnvelope("invalid_request", "无法解析注册请求: "+errUnmarshal.Error()), nil
		}
	}
	cfg, errParse := config.Parse(lifecycle.ConfigYAML)
	if errParse != nil {
		return errorEnvelope("invalid_config", "插件配置解析失败: "+errParse.Error()), nil
	}
	pluginStore.mu.Lock()
	pluginStore.cfg = cfg
	pluginStore.mu.Unlock()
	return okEnvelope(registrationPayload())
}

func registrationPayload() map[string]any {
	return map[string]any{
		"schema_version": 6,
		"metadata": map[string]any{
			"Name":             "Model Quota (大模型额度与余额查询)",
			"Version":          pluginVersion,
			"Author":           "ZRZ-0805",
			"GitHubRepository": "https://github.com/ZRZ-0805/cpa-plugin-model-quota",
			"Logo":             "https://raw.githubusercontent.com/ZRZ-0805/cpa-plugin-model-quota/main/logo.svg",
			"Description":      "实时查询智谱 GLM Coding Plan 用量与 DeepSeek 账户余额，支持同供应商聚合展示、重置倒计时与官方 LOGO。 / Real-time quota and balance tracking for GLM Coding Plan and DeepSeek.",
			"ConfigFields": []map[string]any{
				{
					"Name":        "glm_api_key",
					"Type":        "string",
					"Description": "智谱开放平台 API Key（Coding Plan 套餐用户）",
				},
				{
					"Name":        "glm_region",
					"Type":        "enum",
					"EnumValues":  []string{"cn", "intl"},
					"Default":     "cn",
					"Description": "智谱站点：cn=国内 open.bigmodel.cn，intl=国际 api.z.ai",
				},
				{
					"Name":        "deepseek_api_key",
					"Type":        "string",
					"Description": "DeepSeek 开放平台 API Key",
				},
				{
					"Name":        "timeout_seconds",
					"Type":        "integer",
					"Default":     15,
					"Description": "单次上游查询超时秒数",
				},
			},
		},
		"capabilities": map[string]any{
			"management_api": true,
		},
	}
}

func handleManagement(request []byte) ([]byte, error) {
	var managementRequest struct {
		Method string `json:"Method"`
		Path   string `json:"Path"`
	}
	if errUnmarshal := json.Unmarshal(request, &managementRequest); errUnmarshal != nil {
		return errorEnvelope("invalid_request", "无法解析管理请求: "+errUnmarshal.Error()), nil
	}

	path := matchPath(managementRequest.Path)
	switch {
	case path == queryPath:
		return handleQuery()
	case path == panelPath:
		return handleResourcePage()
	default:
		return managementJSON(http.StatusNotFound, map[string]any{
			"error": fmt.Sprintf("未知路径: %s", managementRequest.Path),
		}), nil
	}
}

// matchPath 把完整管理/资源路径或已归一化后缀统一为注册后缀。
func matchPath(path string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(path), "/")
	for _, prefix := range []string{
		"/v0/resource/plugins/" + pluginID,
		"/v0/management",
	} {
		if strings.HasPrefix(trimmed, prefix+"/") {
			return strings.TrimPrefix(trimmed, prefix)
		}
	}
	return trimmed
}

func handleQuery() ([]byte, error) {
	pluginStore.mu.RLock()
	cfg := pluginStore.cfg
	pluginStore.mu.RUnlock()

	timeout := time.Duration(cfg.Timeout()) * time.Second
	client := hostClient{}
	report := balance.Report{
		Entries:     []balance.Entry{},
		Errors:      map[string]string{},
		FetchedAtMS: time.Now().UnixMilli(),
	}
	if cfg.GLMAPIKey != "" {
		entries, errQuery := balance.QueryGLM(client, cfg.GLMEndpoint(), cfg.GLMAPIKey, timeout)
		if errQuery != nil {
			report.Errors["glm"] = errQuery.Error()
		} else {
			report.Entries = append(report.Entries, entries...)
		}
	}
	if cfg.DeepSeekAPIKey != "" {
		entries, errQuery := balance.QueryDeepSeek(client, cfg.DeepSeekEndpoint(), cfg.DeepSeekAPIKey, timeout)
		if errQuery != nil {
			report.Errors["deepseek"] = errQuery.Error()
		} else {
			report.Entries = append(report.Entries, entries...)
		}
	}
	if cfg.GLMAPIKey == "" && cfg.DeepSeekAPIKey == "" {
		report.Errors["config"] = "尚未配置 API Key：请在管理控制台的「插件」->「配置」中填写 glm_api_key 或 deepseek_api_key / Please configure API keys in plugin settings."
	}
	body, errMarshal := balance.MarshalResult(report)
	if errMarshal != nil {
		return errorEnvelope("marshal_error", errMarshal.Error()), nil
	}
	return managementJSON(http.StatusOK, json.RawMessage(body)), nil
}

func handleResourcePage() ([]byte, error) {
	pageBytes, errRead := resourceFS.ReadFile("resources/index.html")
	if errRead != nil {
		return errorEnvelope("resource_error", "读取页面资源失败: "+errRead.Error()), nil
	}
	raw, errMarshal := okEnvelope(map[string]any{
		"StatusCode": http.StatusOK,
		"Headers":    map[string][]string{"Content-Type": {"text/html; charset=utf-8"}},
		"Body":       pageBytes,
	})
	if errMarshal != nil {
		return errorEnvelope("marshal_error", errMarshal.Error()), nil
	}
	return raw, nil
}

// managementJSON 构造 management.handle 的响应信封结果。
func managementJSON(statusCode int, body any) []byte {
	raw, errMarshal := json.Marshal(body)
	if errMarshal != nil {
		raw = []byte(`{"error":"marshal failed"}`)
	}
	result, _ := json.Marshal(map[string]any{
		"StatusCode": statusCode,
		"Headers":    map[string][]string{"Content-Type": {"application/json; charset=utf-8"}},
		"Body":       raw,
	})
	return mustEnvelope(result)
}

func mustEnvelope(result json.RawMessage) []byte {
	raw, errMarshal := json.Marshal(map[string]any{"ok": true, "result": result})
	if errMarshal != nil {
		return []byte(`{"ok":false,"error":{"code":"marshal_error","message":"marshal failed"}}`)
	}
	return raw
}

func okEnvelope(result any) ([]byte, error) {
	raw, errMarshal := json.Marshal(result)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(map[string]any{"ok": true, "result": json.RawMessage(raw)})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(map[string]any{
		"ok":    false,
		"error": map[string]any{"code": code, "message": message},
	})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}

// callHostMethod 是 C ABI 宿主回调的 Go 侧入口：发送请求并解析返回信封。
func callHostMethod(method string, payload []byte) ([]byte, error) {
	cMethod := C.CString(method)
	defer C.free(unsafe.Pointer(cMethod))
	var requestPtr *C.uint8_t
	if len(payload) > 0 {
		requestPtr = (*C.uint8_t)(C.CBytes(payload))
		defer C.free(unsafe.Pointer(requestPtr))
	}
	var response C.cliproxy_buffer
	if code := C.call_host_api(cMethod, requestPtr, C.size_t(len(payload)), &response); code != 0 {
		return nil, fmt.Errorf("宿主回调返回错误码 %d", code)
	}
	if response.ptr == nil || response.len == 0 {
		return nil, fmt.Errorf("宿主回调返回空响应")
	}
	defer C.free_host_buffer(response.ptr, response.len)
	return C.GoBytes(response.ptr, C.int(response.len)), nil
}
