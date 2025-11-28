package config

import (
	"testing"
	"time"
)

// ===========================================================================
// 时间戳修复测试：验证空字符串或 NULL 的 created_at/updated_at 不会导致 sql.Scan 错误
// Bug 背景：GetExchanges 将 created_at/updated_at 扫描到 time.Time 字段，
// 如果数据库中存储的是空字符串 '' 而非有效时间戳，会导致：
//   "sql: Scan error on column index 12, name \"created_at\": unsupported Scan, storing driver.Value type string into type *time.Time"
// ===========================================================================

// TestInitDefaultData_RepairsEmptyTimestamps 测试 initDefaultData 能修复空时间戳
// 关键步骤：
// 1. 创建数据库并初始化默认数据
// 2. 手动将某条 exchange 记录的 created_at 设为空字符串（模拟旧版数据）
// 3. 再次调用 initDefaultData（模拟服务重启）
// 4. 验证空时间戳已被修复，GetExchanges 不会返回错误
func TestInitDefaultData_RepairsEmptyTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 步骤 1: 验证初始数据已创建
	exchanges, err := db.GetExchanges("default")
	if err != nil {
		t.Fatalf("初始 GetExchanges 失败: %v", err)
	}
	if len(exchanges) == 0 {
		t.Fatal("初始化后没有默认交易所数据")
	}

	// 步骤 2: 手动将某条记录的 created_at 设为空字符串（模拟旧版数据 / 损坏数据）
	_, err = db.db.Exec(`
		UPDATE exchanges SET created_at = '', updated_at = '' WHERE id = 'binance' AND user_id = 'default'
	`)
	if err != nil {
		t.Fatalf("模拟空时间戳失败: %v", err)
	}

	// 步骤 3: 再次调用 initDefaultData（模拟服务重启后的初始化逻辑）
	err = db.initDefaultData()
	if err != nil {
		t.Fatalf("initDefaultData 失败: %v", err)
	}

	// 步骤 4: 验证 GetExchanges 不再失败
	exchanges, err = db.GetExchanges("default")
	if err != nil {
		t.Fatalf("修复后 GetExchanges 失败: %v", err)
	}

	// 步骤 5: 验证时间戳已被修复（不是零值）
	for _, ex := range exchanges {
		if ex.ID == "binance" {
			if ex.CreatedAt.IsZero() {
				t.Errorf("binance 的 CreatedAt 仍然是零值，修复失败")
			}
			if ex.UpdatedAt.IsZero() {
				t.Errorf("binance 的 UpdatedAt 仍然是零值，修复失败")
			}
			t.Logf("✅ binance 时间戳已修复: created_at=%v, updated_at=%v", ex.CreatedAt, ex.UpdatedAt)
			break
		}
	}
}

// TestInitDefaultData_RepairsNullTimestamps 测试 initDefaultData 能修复 NULL 时间戳
func TestInitDefaultData_RepairsNullTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 手动将某条记录的 created_at 设为 NULL
	_, err := db.db.Exec(`
		UPDATE exchanges SET created_at = NULL, updated_at = NULL WHERE id = 'hyperliquid' AND user_id = 'default'
	`)
	if err != nil {
		t.Fatalf("模拟 NULL 时间戳失败: %v", err)
	}

	// 再次调用 initDefaultData
	err = db.initDefaultData()
	if err != nil {
		t.Fatalf("initDefaultData 失败: %v", err)
	}

	// 验证 GetExchanges 不再失败
	exchanges, err := db.GetExchanges("default")
	if err != nil {
		t.Fatalf("修复后 GetExchanges 失败: %v", err)
	}

	// 验证时间戳已被修复
	for _, ex := range exchanges {
		if ex.ID == "hyperliquid" {
			if ex.CreatedAt.IsZero() {
				t.Errorf("hyperliquid 的 CreatedAt 仍然是零值，修复失败")
			}
			if ex.UpdatedAt.IsZero() {
				t.Errorf("hyperliquid 的 UpdatedAt 仍然是零值，修复失败")
			}
			t.Logf("✅ hyperliquid 时间戳已修复: created_at=%v, updated_at=%v", ex.CreatedAt, ex.UpdatedAt)
			break
		}
	}
}

// TestGetExchanges_DoesNotFailWhenTimestampsEmptyAfterRepair 测试修复后 GetExchanges 正常工作
// 这个测试验证完整的修复流程：
// 1. 初始化数据库
// 2. 注入空时间戳
// 3. 重新创建数据库（模拟服务重启）
// 4. 验证 GetExchanges 返回正确数据
func TestGetExchanges_DoesNotFailWhenTimestampsEmptyAfterRepair(t *testing.T) {
	// 创建临时数据库文件
	tmpFile := t.TempDir() + "/test_repair.db"

	// 第一次创建数据库
	db1, err := NewDatabase(tmpFile)
	if err != nil {
		t.Fatalf("第一次创建数据库失败: %v", err)
	}

	// 验证有默认数据
	exchanges, err := db1.GetExchanges("default")
	if err != nil {
		t.Fatalf("第一次 GetExchanges 失败: %v", err)
	}
	if len(exchanges) == 0 {
		t.Fatal("没有默认交易所数据")
	}
	initialCount := len(exchanges)

	// 注入空时间戳（模拟损坏数据）
	_, err = db1.db.Exec(`UPDATE exchanges SET created_at = '', updated_at = '' WHERE user_id = 'default'`)
	if err != nil {
		t.Fatalf("注入空时间戳失败: %v", err)
	}

	// 关闭第一个数据库连接
	db1.Close()

	// 第二次打开数据库（模拟服务重启，会触发 initDefaultData 中的修复逻辑）
	db2, err := NewDatabase(tmpFile)
	if err != nil {
		t.Fatalf("第二次创建数据库失败: %v", err)
	}
	defer db2.Close()

	// 验证 GetExchanges 正常工作
	exchanges, err = db2.GetExchanges("default")
	if err != nil {
		t.Fatalf("修复后 GetExchanges 失败: %v", err)
	}

	// 验证数据完整性
	if len(exchanges) != initialCount {
		t.Errorf("交易所数量不匹配: 期望 %d，实际 %d", initialCount, len(exchanges))
	}

	// 验证所有时间戳都有效
	for _, ex := range exchanges {
		if ex.CreatedAt.IsZero() {
			t.Errorf("交易所 %s 的 CreatedAt 是零值", ex.ID)
		}
		if ex.UpdatedAt.IsZero() {
			t.Errorf("交易所 %s 的 UpdatedAt 是零值", ex.ID)
		}
	}

	t.Logf("✅ 所有 %d 个交易所的时间戳都已正确修复", len(exchanges))
}

// TestNewDatabase_InitializesValidTimestamps 测试新数据库初始化时时间戳有效
func TestNewDatabase_InitializesValidTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	exchanges, err := db.GetExchanges("default")
	if err != nil {
		t.Fatalf("GetExchanges 失败: %v", err)
	}

	if len(exchanges) == 0 {
		t.Fatal("没有默认交易所数据")
	}

	now := time.Now()
	for _, ex := range exchanges {
		// 验证 CreatedAt 不是零值
		if ex.CreatedAt.IsZero() {
			t.Errorf("交易所 %s 的 CreatedAt 是零值", ex.ID)
		}

		// 验证 UpdatedAt 不是零值
		if ex.UpdatedAt.IsZero() {
			t.Errorf("交易所 %s 的 UpdatedAt 是零值", ex.ID)
		}

		// 验证时间戳是合理的（在测试执行前后几分钟内）
		if ex.CreatedAt.After(now.Add(time.Minute)) {
			t.Errorf("交易所 %s 的 CreatedAt 在未来: %v", ex.ID, ex.CreatedAt)
		}
		if ex.CreatedAt.Before(now.Add(-24 * time.Hour)) {
			t.Errorf("交易所 %s 的 CreatedAt 太旧: %v", ex.ID, ex.CreatedAt)
		}
	}

	t.Logf("✅ 所有 %d 个交易所的时间戳都有效", len(exchanges))
}

// TestGetExchanges_HandlesMultipleEmptyTimestampRecords 测试多条记录都有空时间戳的情况
func TestGetExchanges_HandlesMultipleEmptyTimestampRecords(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 将所有记录的时间戳都设为空
	_, err := db.db.Exec(`UPDATE exchanges SET created_at = '', updated_at = '' WHERE user_id = 'default'`)
	if err != nil {
		t.Fatalf("设置空时间戳失败: %v", err)
	}

	// 调用 initDefaultData 修复
	err = db.initDefaultData()
	if err != nil {
		t.Fatalf("initDefaultData 失败: %v", err)
	}

	// 验证所有记录都被修复
	exchanges, err := db.GetExchanges("default")
	if err != nil {
		t.Fatalf("GetExchanges 失败: %v", err)
	}

	for _, ex := range exchanges {
		if ex.CreatedAt.IsZero() {
			t.Errorf("交易所 %s 的 CreatedAt 未被修复", ex.ID)
		}
		if ex.UpdatedAt.IsZero() {
			t.Errorf("交易所 %s 的 UpdatedAt 未被修复", ex.ID)
		}
	}

	t.Logf("✅ 批量修复成功：%d 个交易所的时间戳都已正确", len(exchanges))
}

// TestGetExchanges_PreservesValidTimestamps 测试修复逻辑不会覆盖已有的有效时间戳
func TestGetExchanges_PreservesValidTimestamps(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// 获取原始时间戳
	var originalCreatedAt, originalUpdatedAt string
	err := db.db.QueryRow(`
		SELECT created_at, updated_at FROM exchanges WHERE id = 'binance' AND user_id = 'default'
	`).Scan(&originalCreatedAt, &originalUpdatedAt)
	if err != nil {
		t.Fatalf("查询原始时间戳失败: %v", err)
	}

	t.Logf("原始时间戳: created_at=%s, updated_at=%s", originalCreatedAt, originalUpdatedAt)

	// 等待一小段时间，确保如果被覆盖会有时间差
	time.Sleep(100 * time.Millisecond)

	// 再次调用 initDefaultData
	err = db.initDefaultData()
	if err != nil {
		t.Fatalf("initDefaultData 失败: %v", err)
	}

	// 验证时间戳没有被改变
	var newCreatedAt, newUpdatedAt string
	err = db.db.QueryRow(`
		SELECT created_at, updated_at FROM exchanges WHERE id = 'binance' AND user_id = 'default'
	`).Scan(&newCreatedAt, &newUpdatedAt)
	if err != nil {
		t.Fatalf("查询新时间戳失败: %v", err)
	}

	t.Logf("新时间戳: created_at=%s, updated_at=%s", newCreatedAt, newUpdatedAt)

	if originalCreatedAt != newCreatedAt {
		t.Errorf("有效的 created_at 被覆盖了: %s -> %s", originalCreatedAt, newCreatedAt)
	}
	if originalUpdatedAt != newUpdatedAt {
		t.Errorf("有效的 updated_at 被覆盖了: %s -> %s", originalUpdatedAt, newUpdatedAt)
	}

	t.Logf("✅ 有效时间戳被正确保留")
}
