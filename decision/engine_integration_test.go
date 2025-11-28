package decision

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/mockllm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type llmEnvConfig struct {
	provider string
	apiKey   string
	apiURL   string
	model    string
}

func TestRealLLMDecisionLifecycle(t *testing.T) {
	// 初始化 WSMonitor 避免 nil pointer
	setupTestEnvironment(t)
	defer cleanupTestEnvironment()

	client := newRealMCPClient(t)
	ctx := buildRealTestContext()

	decisionResult, err := GetFullDecisionWithCustomPrompt(ctx, client, "", false, "", 3.0)
	require.NoError(t, err)
	require.NotNil(t, decisionResult)
	require.NotEmpty(t, decisionResult.SystemPrompt)
	require.NotEmpty(t, decisionResult.UserPrompt)

	if len(decisionResult.Decisions) == 0 {
		t.Fatalf("expected at least one decision from real LLM response")
	}

	firstDecision := decisionResult.Decisions[0]
	assert.NotEmpty(t, firstDecision.Symbol)
	assert.NotEmpty(t, firstDecision.Action)

	logDir := t.TempDir()
	decisionLogger := logger.NewDecisionLogger(logDir)
	record := &logger.DecisionRecord{
		Timestamp:    time.Now(),
		SystemPrompt: decisionResult.SystemPrompt,
		InputPrompt:  decisionResult.UserPrompt,
		CoTTrace:     decisionResult.CoTTrace,
		Decisions: []logger.DecisionAction{
			{Action: firstDecision.Action, Symbol: firstDecision.Symbol, Success: true},
		},
		Success: true,
	}
	require.NoError(t, decisionLogger.LogDecision(record))

	files, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	data, err := os.ReadFile(filepath.Join(logDir, files[0].Name()))
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"action\":")
	assert.Contains(t, string(data), "\"symbol\":")
}

func TestRealLLMDecisionFallbackToWait(t *testing.T) {
	// 初始化 WSMonitor 避免 nil pointer
	setupTestEnvironment(t)
	defer cleanupTestEnvironment()

	client := newFailingMCPClient(t)
	ctx := buildRealTestContext()

	buf := &bytes.Buffer{}
	originalWriter := log.Writer()
	log.SetOutput(io.MultiWriter(originalWriter, buf))
	t.Cleanup(func() {
		log.SetOutput(originalWriter)
	})

	decisionResult, err := GetFullDecisionWithCustomPrompt(ctx, client, "", false, "", 3.0)
	require.NoError(t, err)
	require.NotNil(t, decisionResult)
	require.Len(t, decisionResult.Decisions, 1)

	d := decisionResult.Decisions[0]
	assert.Equal(t, "ALL", d.Symbol)
	assert.Equal(t, "wait", d.Action)
	assert.Contains(t, d.Reasoning, "LLM")
	assert.Contains(t, buf.String(), "降级为安全等待决策")

	logDir := t.TempDir()
	decisionLogger := logger.NewDecisionLogger(logDir)
	record := &logger.DecisionRecord{
		Timestamp:    time.Now(),
		SystemPrompt: decisionResult.SystemPrompt,
		InputPrompt:  decisionResult.UserPrompt,
		CoTTrace:     decisionResult.CoTTrace,
		Decisions: []logger.DecisionAction{
			{Action: d.Action, Symbol: d.Symbol, Success: true},
		},
		Success: true,
	}
	require.NoError(t, decisionLogger.LogDecision(record))

	files, err := os.ReadDir(logDir)
	require.NoError(t, err)
	require.NotEmpty(t, files)

	data, err := os.ReadFile(filepath.Join(logDir, files[0].Name()))
	require.NoError(t, err)
	assert.Contains(t, string(data), "\"action\": \"wait\"")
}

func buildRealTestContext() *Context {
	now := time.Now()
	return &Context{
		CurrentTime:     now.Format(time.RFC3339),
		RuntimeMinutes:  12,
		CallCount:       1,
		BTCETHLeverage:  10,
		AltcoinLeverage: 5,
		Account: AccountInfo{
			TotalEquity:      5000,
			AvailableBalance: 3500,
			UnrealizedPnL:    120,
			TotalPnLPct:      2.4,
			MarginUsedPct:    18,
			PositionCount:    1,
		},
		Positions: []PositionInfo{
			{
				Symbol:           "ETHUSDT",
				Side:             "long",
				EntryPrice:       3200,
				MarkPrice:        3250,
				Quantity:         0.5,
				Leverage:         3,
				UnrealizedPnL:    25,
				UnrealizedPnLPct: 4.6,
				PeakPnLPct:       6.0,
				LiquidationPrice: 2500,
				MarginUsed:       100,
				UpdateTime:       now.Add(-30 * time.Minute).UnixMilli(),
			},
		},
		CandidateCoins: []CandidateCoin{
			{Symbol: "BTCUSDT", Sources: []string{"ai500"}},
			{Symbol: "ETHUSDT", Sources: []string{"oi_top"}},
			{Symbol: "SOLUSDT", Sources: []string{"ai500"}},
		},
		IndicatorConfig: market.GetDefaultIndicatorConfig(),
	}
}

func newRealMCPClient(t *testing.T) *mcp.Client {
	// Default to using a fast local mock to avoid external LLM calls during CI or
	// regular developer runs. To explicitly exercise a real LLM set
	// RUN_REAL_LLM_TESTS=true and provide LLM_API_KEY in your environment or .env.
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("RUN_REAL_LLM_TESTS"))); v != "true" {
		// Return a deterministic mock response which contains a valid decision JSON block
		assistantContent := "<reasoning>Mocked reasoning for tests</reasoning>\n\n<decision>```json\n[{\"symbol\": \"BTCUSDT\", \"action\": \"wait\", \"confidence\": 100, \"reasoning\": \"mock response\"}]\n```</decision>"
		return mockllm.NewMCPClientFromMockServer(t, assistantContent)
	}

	cfg := loadLLMConfigFromEnv(t)
	client := mcp.New()

	switch cfg.provider {
	case "deepseek", "deepseek-v2", "ds":
		client.SetDeepSeekAPIKey(cfg.apiKey, cfg.apiURL, cfg.model)
		t.Logf("🔧 [MCP] DeepSeek 配置: %s", cfg.model)
	case "qwen":
		client.SetQwenAPIKey(cfg.apiKey, cfg.apiURL, cfg.model)
		t.Logf("🔧 [MCP] Qwen 配置: %s", cfg.model)
	case "glm":
		client.SetGLMAPIKey(cfg.apiKey, cfg.apiURL, cfg.model)
		t.Logf("🔧 [MCP] GLM 配置: %s", cfg.model)
	case "openai":
		client.SetCustomAPI(cfg.apiURL, cfg.apiKey, cfg.model)
		t.Logf("🔧 [MCP] OpenAI 兼容配置: %s", cfg.model)
	default:
		if cfg.apiURL == "" {
			t.Skipf("custom provider %s requires LLM_API_URL", cfg.provider)
		}
		client.SetCustomAPI(cfg.apiURL, cfg.apiKey, cfg.model)
		t.Logf("🔧 [MCP] 自定义 API 配置: %s", cfg.provider)
	}

	// 显示配置信息（隐藏 API Key）
	maskedKey := cfg.apiKey
	if len(maskedKey) > 8 {
		maskedKey = cfg.apiKey[:4] + "..." + cfg.apiKey[len(cfg.apiKey)-4:]
	}
	t.Logf("🔧 [MCP] %s 使用自定义 BaseURL: %s", cfg.provider, cfg.apiURL)
	t.Logf("🔧 [MCP] %s 使用自定义 Model: %s", cfg.provider, cfg.model)
	t.Logf("🔧 [MCP] %s API Key: %s", cfg.provider, maskedKey)

	return client
}

func newFailingMCPClient(t *testing.T) *mcp.Client {
	// similar opt-in semantics: if RUN_REAL_LLM_TESTS=true then try to use a
	// real client and force a timeout; otherwise use a failing mock server.
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("RUN_REAL_LLM_TESTS"))); v == "true" {
		client := newRealMCPClient(t)
		client.Timeout = time.Nanosecond
		return client
	}

	return mockllm.NewFailingMCPClientFromMockServer(t)
}

func loadLLMConfigFromEnv(t *testing.T) llmEnvConfig {
	t.Helper()
	loadDotEnvIfNeeded(t)

	apiKey := strings.TrimSpace(os.Getenv("LLM_API_KEY"))
	if apiKey == "" {
		t.Skip("LLM_API_KEY 未配置，跳过真实 LLM 测试")
	}

	provider := strings.ToLower(strings.TrimSpace(os.Getenv("LLM_PROVIDER")))
	if provider == "" {
		provider = "deepseek" // 默认使用 deepseek
	}

	apiURL := strings.TrimSpace(os.Getenv("LLM_API_URL"))
	model := strings.TrimSpace(os.Getenv("LLM_MODEL"))

	// 如果没有设置 provider，但从 API URL 推断出来
	if provider == "deepseek" && apiURL != "" {
		if strings.Contains(apiURL, "bigmodel.cn") {
			provider = "glm" // 智谱 GLM
		}
	}

	t.Logf("🔧 [MCP] 配置信息: provider=%s, apiURL=%s, model=%s", provider, apiURL, model)

	return llmEnvConfig{
		provider: provider,
		apiKey:   apiKey,
		apiURL:   apiURL,
		model:    model,
	}
}

func loadDotEnvIfNeeded(t *testing.T) {
	t.Helper()
	if os.Getenv("LLM_API_KEY") != "" {
		t.Logf("✓ LLM_API_KEY 已存在于环境变量中")
		return
	}

	// 尝试多个可能的 .env 位置
	possibleEnvPaths := []string{
		".env",          // 当前目录
		"../.env",       // 父目录
		"../../.env",    // 父父目录
		"./.env",        // 显式当前目录
		"../../../.env", // 更上层目录
	}

	// 获取当前工作目录用于调试
	wd, _ := os.Getwd()
	t.Logf("🔍 当前工作目录: %s", wd)

	var loaded bool
	var loadedPath string
	for _, envPath := range possibleEnvPaths {
		// 检查文件是否存在
		if _, err := os.Stat(envPath); err != nil {
			t.Logf("❌ 路径不存在: %s", envPath)
			continue
		}

		// 尝试读取文件
		data, err := os.ReadFile(envPath)
		if err != nil {
			t.Logf("❌ 读取失败: %s - %v", envPath, err)
			continue
		}

		t.Logf("✅ 成功读取 .env 文件: %s (%d bytes)", envPath, len(data))

		// 解析并设置环境变量
		lines := strings.Split(string(data), "\n")
		var loadedVars []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "#") {
				continue
			}
			if strings.HasPrefix(trimmed, "export ") {
				trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "export "))
			}
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "'\"")
			if key != "" {
				_ = os.Setenv(key, value)
				// 安全地掩码敏感值
				var maskedValue string
				if len(value) <= 8 {
					maskedValue = strings.Repeat("*", len(value))
				} else {
					maskedValue = value[:4] + strings.Repeat("*", len(value)-8) + value[len(value)-4:]
				}
				loadedVars = append(loadedVars, fmt.Sprintf("%s=%s", key, maskedValue))
			}
		}

		if len(loadedVars) > 0 {
			t.Logf("✅ 从 %s 加载了 %d 个环境变量: %s", envPath, len(loadedVars), strings.Join(loadedVars, ", "))
			loaded = true
			loadedPath = envPath
			break
		} else {
			t.Logf("⚠️ 文件 %s 存在但没有找到有效的环境变量", envPath)
		}
	}

	if !loaded {
		// 列出当前目录的文件用于调试
		if files, err := os.ReadDir("."); err == nil {
			var fileNames []string
			for _, file := range files {
				if !file.IsDir() {
					fileNames = append(fileNames, file.Name())
				}
			}
			t.Logf("📁 当前目录文件: %s", strings.Join(fileNames, ", "))
		}

		t.Skip("❌ 无法从任何路径读取 .env 文件且环境未提供 LLM_API_KEY，跳过真实 LLM 测试")
	}

	// 验证关键环境变量是否加载成功
	apiKey := os.Getenv("LLM_API_KEY")
	if apiKey == "" {
		t.Fatalf("❌ .env 文件已加载 (%s) 但未找到 LLM_API_KEY", loadedPath)
	}

	provider := os.Getenv("LLM_PROVIDER")
	apiURL := os.Getenv("LLM_API_URL")
	model := os.Getenv("LLM_MODEL")

	t.Logf("🎯 配置验证成功: Provider=%s, Model=%s, URL=%s, Key=****%s",
		provider, model, apiURL, apiKey[len(apiKey)-4:])
}

// setupTestEnvironment 初始化测试环境，包括 WSMonitor 和提示词路径
func setupTestEnvironment(t *testing.T) {
	t.Helper()

	// 初始化 WSMonitor 避免 nil pointer
	if market.WSMonitorCli == nil {
		market.WSMonitorCli = market.NewWSMonitor(10)
		t.Logf("✓ 已初始化 WSMonitor for test")
	}

	// 确保当前工作目录正确，以便找到 prompts 目录
	promptsPath := "prompts"
	if _, err := os.Stat(promptsPath); os.IsNotExist(err) {
		// 如果当前目录下没有 prompts，尝试项目根目录
		if _, err := os.Stat("../prompts"); err == nil {
			promptsPath = "../prompts"
			t.Logf("✓ 使用项目根目录的 prompts 路径: %s", promptsPath)
		}
	}

	// 设置提示词路径并重新加载
	SetPromptsDir(promptsPath)
	ReloadPromptsIfNeeded()
}

// cleanupTestEnvironment 清理测试环境
func cleanupTestEnvironment() {
	// 重置 WSMonitor 避免影响其他测试
	market.WSMonitorCli = nil

	// 恢复默认的 promptsDir 路径
	SetPromptsDir("prompts")
}

// ReloadPromptsIfNeeded 重新加载提示词模板（如果需要）
func ReloadPromptsIfNeeded() {
	// 如果提示词管理器没有加载模板，尝试重新加载
	if globalPromptManager == nil || len(globalPromptManager.GetAllTemplateNames()) == 0 {
		if globalPromptManager == nil {
			globalPromptManager = NewPromptManager()
		}
		if err := globalPromptManager.LoadTemplates(promptsDir); err != nil {
			log.Printf("⚠️  重新加载提示词模板失败: %v", err)
		} else {
			log.Printf("✓ 已重新加载 %d 个提示词模板", len(globalPromptManager.GetAllTemplateNames()))
		}
	}
}
