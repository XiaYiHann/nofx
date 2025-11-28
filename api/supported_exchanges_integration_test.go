package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nofx/config"
	"nofx/crypto"
	"nofx/manager"
	"nofx/testhelpers"
)

// ===========================================================================
// API 集成测试：验证 /api/supported-exchanges 端点在数据库时间戳损坏时的行为
// Bug 背景：当 exchanges 表中的 created_at/updated_at 是空字符串时，
// GetExchanges 会返回 sql.Scan 错误，导致 HTTP 500 响应
// ===========================================================================

// TestSupportedExchangesEndpoint_HandlesLegacyEmptyTimestamps 测试端点能处理旧版空时间戳数据
// 关键步骤：
// 1. 创建测试数据库和服务器
// 2. 注入空时间戳数据（模拟旧版/损坏数据）
// 3. 触发数据库重新初始化（修复逻辑）
// 4. 调用 /api/supported-exchanges 端点
// 5. 验证返回 HTTP 200 和有效的 JSON 数组
func TestSupportedExchangesEndpoint_HandlesLegacyEmptyTimestamps(t *testing.T) {
	t.Helper()
	testhelpers.LoadDotEnv(t)

	// 设置默认加密密钥（如果环境变量未设置）
	if os.Getenv("DATA_ENCRYPTION_KEY") == "" {
		os.Setenv("DATA_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	}

	// 创建临时数据库文件
	dbPath := filepath.Join(t.TempDir(), "test_exchanges.db")

	// 第一次创建数据库
	db1, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}

	// 注入空时间戳（模拟旧版数据）
	_, err = db1.GetDB().Exec(`UPDATE exchanges SET created_at = '', updated_at = '' WHERE user_id = 'default'`)
	if err != nil {
		t.Fatalf("注入空时间戳失败: %v", err)
	}

	// 关闭数据库
	db1.Close()

	// 第二次打开数据库（触发 initDefaultData 中的修复逻辑）
	database, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("重新打开数据库失败: %v", err)
	}
	defer database.Close()

	// 创建加密服务
	rsaKeyPath := filepath.Join(t.TempDir(), "test_rsa_key")
	cryptoService, err := crypto.NewCryptoService(rsaKeyPath)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}
	database.SetCryptoService(cryptoService)

	// 创建服务器
	traderManager := manager.NewTraderManager()
	server := NewServer(traderManager, database, cryptoService, 0, true)

	// 发送请求到 /api/supported-exchanges
	req := httptest.NewRequest(http.MethodGet, "/api/supported-exchanges", nil)
	resp := httptest.NewRecorder()
	server.router.ServeHTTP(resp, req)

	// 验证响应状态码
	if resp.Code != http.StatusOK {
		t.Errorf("期望 HTTP 200，实际 %d，响应体: %s", resp.Code, resp.Body.String())
	}

	// 验证响应是有效的 JSON 数组
	var exchanges []SafeExchangeConfig
	if err := json.Unmarshal(resp.Body.Bytes(), &exchanges); err != nil {
		t.Errorf("响应不是有效的 JSON: %v, 响应体: %s", err, resp.Body.String())
	}

	// 验证返回了交易所数据
	if len(exchanges) == 0 {
		t.Error("返回的交易所列表为空")
	}

	// 验证每个交易所都有必要字段
	for _, ex := range exchanges {
		if ex.ID == "" {
			t.Error("交易所缺少 id 字段")
		}
		if ex.Name == "" {
			t.Errorf("交易所 %s 缺少 name 字段", ex.ID)
		}
		if ex.Type == "" {
			t.Errorf("交易所 %s 缺少 type 字段", ex.ID)
		}
	}

	t.Logf("✅ /api/supported-exchanges 返回 %d 个交易所", len(exchanges))
}

// TestSupportedExchangesEndpoint_ReturnsValidJSON 测试端点返回有效 JSON
func TestSupportedExchangesEndpoint_ReturnsValidJSON(t *testing.T) {
	t.Helper()
	testhelpers.LoadDotEnv(t)

	if os.Getenv("DATA_ENCRYPTION_KEY") == "" {
		os.Setenv("DATA_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	}

	dbPath := filepath.Join(t.TempDir(), "test_valid_json.db")
	database, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	defer database.Close()

	rsaKeyPath := filepath.Join(t.TempDir(), "test_rsa_key")
	cryptoService, err := crypto.NewCryptoService(rsaKeyPath)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}
	database.SetCryptoService(cryptoService)

	traderManager := manager.NewTraderManager()
	server := NewServer(traderManager, database, cryptoService, 0, true)

	req := httptest.NewRequest(http.MethodGet, "/api/supported-exchanges", nil)
	resp := httptest.NewRecorder()
	server.router.ServeHTTP(resp, req)

	// 验证状态码
	if resp.Code != http.StatusOK {
		t.Fatalf("期望 HTTP 200，实际 %d", resp.Code)
	}

	// 验证 Content-Type
	contentType := resp.Header().Get("Content-Type")
	if contentType != "application/json; charset=utf-8" {
		t.Errorf("期望 Content-Type 为 application/json，实际 %s", contentType)
	}

	// 验证 JSON 可解析
	var exchanges []SafeExchangeConfig
	if err := json.Unmarshal(resp.Body.Bytes(), &exchanges); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v", err)
	}

	// 验证预期的交易所存在
	expectedIDs := map[string]bool{
		"binance":       false,
		"hyperliquid":   false,
		"aster":         false,
		"paper_trading": false,
	}

	for _, ex := range exchanges {
		if _, ok := expectedIDs[ex.ID]; ok {
			expectedIDs[ex.ID] = true
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("预期的交易所 %s 未在响应中找到", id)
		}
	}

	t.Logf("✅ 验证通过：返回 %d 个交易所，所有预期交易所都存在", len(exchanges))
}

// TestSupportedExchangesEndpoint_DoesNotExposeSecrets 测试端点不泄露敏感信息
func TestSupportedExchangesEndpoint_DoesNotExposeSecrets(t *testing.T) {
	t.Helper()
	testhelpers.LoadDotEnv(t)

	if os.Getenv("DATA_ENCRYPTION_KEY") == "" {
		os.Setenv("DATA_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	}

	dbPath := filepath.Join(t.TempDir(), "test_no_secrets.db")
	database, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	defer database.Close()

	rsaKeyPath := filepath.Join(t.TempDir(), "test_rsa_key")
	cryptoService, err := crypto.NewCryptoService(rsaKeyPath)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}
	database.SetCryptoService(cryptoService)

	traderManager := manager.NewTraderManager()
	server := NewServer(traderManager, database, cryptoService, 0, true)

	req := httptest.NewRequest(http.MethodGet, "/api/supported-exchanges", nil)
	resp := httptest.NewRecorder()
	server.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 HTTP 200，实际 %d", resp.Code)
	}

	// 检查响应中不包含敏感字段
	respBody := resp.Body.String()
	sensitiveFields := []string{
		"apiKey",
		"secretKey",
		"asterPrivateKey",
		"lighterPrivateKey",
		"api_key",
		"secret_key",
	}

	for _, field := range sensitiveFields {
		// 简单检查：如果字段出现且不为空
		if containsNonEmptyField(respBody, field) {
			t.Errorf("响应可能包含敏感字段 %s", field)
		}
	}

	t.Logf("✅ 验证通过：响应不包含敏感信息")
}

// TestSupportedModelsEndpoint_ReturnsValidJSON 测试 /api/supported-models 端点
func TestSupportedModelsEndpoint_ReturnsValidJSON(t *testing.T) {
	t.Helper()
	testhelpers.LoadDotEnv(t)

	if os.Getenv("DATA_ENCRYPTION_KEY") == "" {
		os.Setenv("DATA_ENCRYPTION_KEY", "0123456789abcdef0123456789abcdef")
	}

	dbPath := filepath.Join(t.TempDir(), "test_models.db")
	database, err := config.NewDatabase(dbPath)
	if err != nil {
		t.Fatalf("创建数据库失败: %v", err)
	}
	defer database.Close()

	rsaKeyPath := filepath.Join(t.TempDir(), "test_rsa_key")
	cryptoService, err := crypto.NewCryptoService(rsaKeyPath)
	if err != nil {
		t.Fatalf("创建加密服务失败: %v", err)
	}
	database.SetCryptoService(cryptoService)

	traderManager := manager.NewTraderManager()
	server := NewServer(traderManager, database, cryptoService, 0, true)

	req := httptest.NewRequest(http.MethodGet, "/api/supported-models", nil)
	resp := httptest.NewRecorder()
	server.router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("期望 HTTP 200，实际 %d, 响应: %s", resp.Code, resp.Body.String())
	}

	var models []*config.AIModelConfig
	if err := json.Unmarshal(resp.Body.Bytes(), &models); err != nil {
		t.Fatalf("响应 JSON 解析失败: %v", err)
	}

	if len(models) == 0 {
		t.Error("返回的模型列表为空")
	}

	// 验证预期的模型存在
	expectedProviders := map[string]bool{
		"deepseek": false,
		"qwen":     false,
		"glm":      false,
	}

	for _, model := range models {
		if _, ok := expectedProviders[model.Provider]; ok {
			expectedProviders[model.Provider] = true
		}
	}

	for provider, found := range expectedProviders {
		if !found {
			t.Errorf("预期的模型提供商 %s 未在响应中找到", provider)
		}
	}

	t.Logf("✅ /api/supported-models 返回 %d 个模型", len(models))
}

// containsNonEmptyField 检查 JSON 字符串中是否包含非空的指定字段
func containsNonEmptyField(jsonStr, field string) bool {
	// 简单的字符串检查，用于测试目的
	// 检查 "field":"非空值" 的模式
	patterns := []string{
		`"` + field + `":"[^"]+`,
		`"` + field + `": "[^"]+`,
	}
	for _, pattern := range patterns {
		if matched, _ := json.Marshal(pattern); matched != nil {
			// 这里简化处理，只检查字段名是否出现并且值不为空字符串
			if len(jsonStr) > 0 {
				// 使用 JSON 解析来安全检查
				var data []map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &data); err == nil {
					for _, item := range data {
						if val, ok := item[field]; ok {
							if str, isStr := val.(string); isStr && str != "" {
								return true
							}
						}
					}
				}
			}
		}
	}
	return false
}
