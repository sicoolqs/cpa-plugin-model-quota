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

func TestQueryMissingKey(t *testing.T) {
	if _, errQuery := QueryGLM(fakeHostClient{}, "https://example.com", "", time.Second); errQuery == nil {
		t.Fatal("缺 glm_api_key 应返回错误")
	}
	if _, errQuery := QueryDeepSeek(fakeHostClient{}, "https://example.com", "", time.Second); errQuery == nil {
		t.Fatal("缺 deepseek_api_key 应返回错误")
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
