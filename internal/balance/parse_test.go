package balance

import (
	"testing"
	"time"
)

type fakeHostClient struct {
	body []byte
	err  error
}

func (f fakeHostClient) DoHTTP(url string, headers map[string][]string, timeout time.Duration) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.body, nil
}

func TestParseGLM(t *testing.T) {
	body := []byte(`{
		"code": 200,
		"msg": "操作成功",
		"data": {
			"limits": [
				{"type": "TIME_LIMIT", "percentage": 7, "usage": 1000, "currentValue": 72, "remaining": 928},
				{"type": "TOKENS_LIMIT", "percentage": 44, "nextResetTime": 1893456000000},
				{"type": "TOKENS_LIMIT", "percentage": 53, "nextResetTime": 1896048000000}
			],
			"level": "pro"
		},
		"success": true
	}`)
	entries, errParse := ParseGLM(body)
	if errParse != nil {
		t.Fatalf("ParseGLM 失败: %v", errParse)
	}
	if len(entries) != 3 {
		t.Fatalf("期望 3 条条目，得到 %d", len(entries))
	}
	fiveHour := entries[0]
	if fiveHour.Label != "5小时额度" {
		t.Errorf("第一条应为 5小时额度，得到 %q", fiveHour.Label)
	}
	if fiveHour.UsedPercent == nil || *fiveHour.UsedPercent != 44 {
		t.Errorf("5小时额度已用应为 44%%，得到 %v", fiveHour.UsedPercent)
	}
	if fiveHour.Remaining != "56%" {
		t.Errorf("5小时额度剩余应为 56%%，得到 %q", fiveHour.Remaining)
	}
	if fiveHour.ResetAtMS != 1893456000000 {
		t.Errorf("5小时额度重置时间应为 1893456000000，得到 %d", fiveHour.ResetAtMS)
	}
	if entries[1].Label != "每周额度" {
		t.Errorf("第二条应为 每周额度，得到 %q", entries[1].Label)
	}
	if entries[2].Label != "MCP 每月调用" {
		t.Errorf("第三条应为 MCP 每月调用，得到 %q", entries[2].Label)
	}
	if entries[2].Remaining != "72 / 1000 次" {
		t.Errorf("MCP 条目展示不符，得到 %q", entries[2].Remaining)
	}
}

func TestNormalizeResetMS(t *testing.T) {
	cases := []struct {
		in, want int64
	}{
		{0, 0},
		{-5, 0},
		{1893456000, 1893456000000},    // 秒 → 毫秒
		{1893456000000, 1893456000000}, // 已是毫秒
	}
	for _, c := range cases {
		if got := normalizeResetMS(c.in); got != c.want {
			t.Errorf("normalizeResetMS(%d) = %d，期望 %d", c.in, got, c.want)
		}
	}
}

func TestParseGLMNoTokenLimits(t *testing.T) {
	body := []byte(`{"code": 200, "msg": "ok", "data": {"limits": [], "level": ""}}`)
	if _, errParse := ParseGLM(body); errParse == nil {
		t.Fatal("空 limits 应返回错误")
	}
}

func TestParseGLMErrorCode(t *testing.T) {
	body := []byte(`{"code": 401, "msg": "unauthorized", "data": {}}`)
	if _, errParse := ParseGLM(body); errParse == nil {
		t.Fatal("错误码应返回错误")
	}
}

func TestParseDeepSeek(t *testing.T) {
	body := []byte(`{
		"is_available": true,
		"balance_infos": [
			{"currency": "CNY", "total_balance": "110.50", "granted_balance": "0.00", "topped_up_balance": "110.50"}
		]
	}`)
	entries, errParse := ParseDeepSeek(body)
	if errParse != nil {
		t.Fatalf("ParseDeepSeek 失败: %v", errParse)
	}
	if len(entries) != 1 {
		t.Fatalf("期望 1 条条目，得到 %d", len(entries))
	}
	if entries[0].Remaining != "110.50" {
		t.Errorf("余额应为 110.50，得到 %q", entries[0].Remaining)
	}
	if entries[0].Detail == "" {
		t.Error("Detail 应包含币种信息")
	}
}

func TestParseOpenCode(t *testing.T) {
	body := []byte(`{
		"usage": {
			"rolling": {"status": "ok", "percent": 0, "resetsAt": "2026-10-19T07:55:51.000Z"},
			"weekly": {"status": "ok", "percent": 12, "resetsAt": "2026-10-26T00:00:00.000Z"},
			"monthly": {"status": "ok", "percent": 42, "resetsAt": "2026-10-21T03:36:57.000Z"}
		}
	}`)
	entries, errParse := ParseOpenCode(body)
	if errParse != nil {
		t.Fatalf("ParseOpenCode 失败: %v", errParse)
	}
	if len(entries) != 3 {
		t.Fatalf("期望 3 条条目，得到 %d", len(entries))
	}
	if entries[0].Provider != "opencode-go" {
		t.Errorf("Provider 应为 opencode-go，得到 %q", entries[0].Provider)
	}
	if entries[0].Label != "5小时额度" {
		t.Errorf("第一条应为 5小时额度，得到 %q", entries[0].Label)
	}
	if entries[0].UsedPercent == nil || *entries[0].UsedPercent != 0 {
		t.Errorf("5小时额度已用应为 0%%，得到 %v", entries[0].UsedPercent)
	}
	if entries[0].Remaining != "100%" {
		t.Errorf("5小时额度剩余应为 100%%，得到 %q", entries[0].Remaining)
	}
	if want := int64(1792396551000); entries[0].ResetAtMS != want {
		t.Errorf("5小时额度重置时间应为 %d，得到 %d", want, entries[0].ResetAtMS)
	}
	if entries[1].Label != "每周额度" {
		t.Errorf("第二条应为 每周额度，得到 %q", entries[1].Label)
	}
	if entries[2].Label != "每月额度" {
		t.Errorf("第三条应为 每月额度，得到 %q", entries[2].Label)
	}
	if entries[2].UsedPercent == nil || *entries[2].UsedPercent != 42 {
		t.Errorf("每月额度已用应为 42%%，得到 %v", entries[2].UsedPercent)
	}
	if entries[2].Detail != "" {
		t.Errorf("ok 状态不应带 Detail，得到 %q", entries[2].Detail)
	}
}

func TestParseOpenCodeAbnormalStatus(t *testing.T) {
	body := []byte(`{
		"usage": {
			"rolling": {"status": "blocked", "percent": 100, "resetsAt": "2026-10-19T07:55:51.000Z"},
			"weekly": {"status": "ok", "percent": 0, "resetsAt": "2026-10-26T00:00:00.000Z"},
			"monthly": {"status": "ok", "percent": 42, "resetsAt": "2026-10-21T03:36:57.000Z"}
		}
	}`)
	entries, errParse := ParseOpenCode(body)
	if errParse != nil {
		t.Fatalf("ParseOpenCode 失败: %v", errParse)
	}
	if entries[0].Detail != "状态: blocked" {
		t.Errorf("blocked 状态应写入 Detail，得到 %q", entries[0].Detail)
	}
}

func TestParseOpenCodeInvalidResetsAt(t *testing.T) {
	body := []byte(`{
		"usage": {
			"rolling": {"status": "ok", "percent": 10, "resetsAt": "not-a-time"},
			"weekly": {"status": "ok", "percent": 0, "resetsAt": ""},
			"monthly": {"status": "ok", "percent": 0, "resetsAt": "2026-10-21T03:36:57.000Z"}
		}
	}`)
	entries, errParse := ParseOpenCode(body)
	if errParse != nil {
		t.Fatalf("ParseOpenCode 失败: %v", errParse)
	}
	if entries[0].ResetAtMS != 0 || entries[1].ResetAtMS != 0 {
		t.Errorf("非法 resetsAt 应归一为 0，得到 %d / %d", entries[0].ResetAtMS, entries[1].ResetAtMS)
	}
	if entries[2].ResetAtMS == 0 {
		t.Error("合法 resetsAt 不应为 0")
	}
}

func TestParseOpenCodeEmptyData(t *testing.T) {
	body := []byte(`{"usage": {"rolling": {"status": "ok", "percent": 0, "resetsAt": ""}, "weekly": {"status": "ok", "percent": 0, "resetsAt": ""}, "monthly": {"status": "ok", "percent": 0, "resetsAt": ""}}}`)
	if _, errParse := ParseOpenCode(body); errParse == nil {
		t.Fatal("空额度数据应返回错误")
	}
	if _, errParse := ParseOpenCode([]byte(`{"usage": {}}`)); errParse == nil {
		t.Fatal("缺失窗口应返回错误")
	}
	if _, errParse := ParseOpenCode([]byte(`not-json`)); errParse == nil {
		t.Fatal("非法 JSON 应返回错误")
	}
}

func TestQueryMissingKey(t *testing.T) {
	if _, errQuery := QueryGLM(fakeHostClient{}, "https://example.com", "", time.Second); errQuery == nil {
		t.Fatal("缺 glm_api_key 应返回错误")
	}
	if _, errQuery := QueryDeepSeek(fakeHostClient{}, "https://example.com", "", time.Second); errQuery == nil {
		t.Fatal("缺 deepseek_api_key 应返回错误")
	}
	if _, errQuery := QueryOpenCode(fakeHostClient{}, "https://example.com", "", time.Second); errQuery == nil {
		t.Fatal("缺 opencode_api_key 应返回错误")
	}
}

func TestQueryOpenCodePassesAuthorization(t *testing.T) {
	var captured map[string][]string
	client := stubClient{onDo: func(url string, headers map[string][]string) ([]byte, error) {
		captured = headers
		return []byte(`{"usage":{"rolling":{"status":"ok","percent":0,"resetsAt":"2026-10-19T07:55:51.000Z"},"weekly":{"status":"ok","percent":0,"resetsAt":"2026-10-26T00:00:00.000Z"},"monthly":{"status":"ok","percent":42,"resetsAt":"2026-10-21T03:36:57.000Z"}}}`), nil
	}}
	if _, errQuery := QueryOpenCode(client, "https://example.com", "test-key", time.Second); errQuery != nil {
		t.Fatalf("QueryOpenCode 失败: %v", errQuery)
	}
	if got := captured["Authorization"][0]; got != "Bearer test-key" {
		t.Errorf("OpenCode 认证头应为 Bearer 前缀，得到 %q", got)
	}
}

func TestQueryGLMPassesAuthorization(t *testing.T) {
	var captured map[string][]string
	client := stubClient{onDo: func(url string, headers map[string][]string) ([]byte, error) {
		captured = headers
		return []byte(`{"code":200,"msg":"ok","data":{"limits":[{"type":"TOKENS_LIMIT","percentage":10}],"level":"lite"}}`), nil
	}}
	if _, errQuery := QueryGLM(client, "https://example.com", "test-key", time.Second); errQuery != nil {
		t.Fatalf("QueryGLM 失败: %v", errQuery)
	}
	if got := captured["Authorization"][0]; got != "test-key" {
		t.Errorf("智谱认证头应为裸 token，得到 %q", got)
	}
}

type stubClient struct {
	onDo func(url string, headers map[string][]string) ([]byte, error)
}

func (s stubClient) DoHTTP(url string, headers map[string][]string, timeout time.Duration) ([]byte, error) {
	return s.onDo(url, headers)
}
