# NOFX 系统实体关系图 (ERD)

```mermaid
erDiagram
    USERS {
        string id PK "用户唯一标识"
        string email "用户邮箱"
        string password_hash "密码哈希"
        string jwt_token "JWT令牌"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
        boolean is_active "是否激活"
        boolean is_admin "是否管理员"
        string beta_code "内测码"
    }

    AI_MODELS {
        string id PK "AI模型唯一标识"
        string user_id FK "所属用户ID"
        string provider "供应商(deepseek/qwen/glm/custom)"
        string model_name "模型名称"
        string api_key "API密钥(加密存储)"
        string custom_api_url "自定义API URL"
        string custom_model_name "自定义模型名称"
        boolean enabled "是否启用"
        json config "模型配置参数"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
    }

    EXCHANGES {
        string id PK "交易所配置唯一标识"
        string user_id FK "所属用户ID"
        string exchange_type "交易所类型(binance/hyperliquid/aster)"
        string api_key "API密钥(加密存储)"
        string secret_key "密钥(加密存储)"
        string hyperliquid_wallet_addr "Hyperliquid钱包地址"
        string aster_user "Aster用户地址"
        string aster_signer "Aster签名地址"
        boolean testnet "是否测试网"
        boolean enabled "是否启用"
        json config "交易所配置参数"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
    }

    TRADERS {
        string id PK "交易员唯一标识"
        string user_id FK "所属用户ID"
        string name "交易员名称"
        string ai_model_id FK "AI模型ID"
        string exchange_id FK "交易所配置ID"
        float initial_balance "初始资金"
        int scan_interval_minutes "扫描间隔(分钟)"
        int btc_eth_leverage "BTC/ETH最大杠杆"
        int altcoin_leverage "山寨币最大杠杆"
        boolean is_cross_margin "是否全仓模式"
        boolean is_running "是否运行中"
        string trading_symbols "交易币种列表(逗号分隔)"
        boolean use_coin_pool "是否使用币种池"
        string custom_prompt "自定义提示词"
        boolean override_base_prompt "是否覆盖基础提示词"
        string system_prompt_template "系统提示词模板"
        string indicator_config "技术指标配置(JSON)"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
    }

    TRADING_DECISIONS {
        string id PK "决策记录唯一标识"
        string trader_id FK "交易员ID"
        string decision_type "决策类型(wait/hold/open_long/open_short/close_long/close_short)"
        string symbol "交易币种"
        float quantity "数量"
        int leverage "杠杆倍数"
        float entry_price "入场价格"
        float stop_loss_price "止损价格"
        float take_profit_price "止盈价格"
        text reasoning "决策推理过程"
        text market_data "市场数据快照"
        text ai_response "AI响应内容"
        string status "执行状态(pending/success/failed)"
        string error_message "错误信息"
        timestamp created_at "创建时间"
        timestamp executed_at "执行时间"
    }

    POSITIONS {
        string id PK "持仓记录唯一标识"
        string trader_id FK "交易员ID"
        string symbol "交易币种"
        string position_side "持仓方向(long/short)"
        float quantity "持仓数量"
        float entry_price "开仓价格"
        float current_price "当前价格"
        float unrealized_pnl "未实现盈亏"
        float realized_pnl "已实现盈亏"
        float margin_used "已用保证金"
        int leverage "杠杆倍数"
        timestamp opened_at "开仓时间"
        timestamp closed_at "平仓时间"
        string status "持仓状态(open/closed)"
    }

    PERFORMANCE_STATS {
        string id PK "性能统计唯一标识"
        string trader_id FK "交易员ID"
        date stat_date "统计日期"
        float total_equity "总权益"
        float total_pnl "总盈亏"
        float total_pnl_pct "总盈亏百分比"
        int total_trades "总交易次数"
        int winning_trades "盈利交易次数"
        int losing_trades "亏损交易次数"
        float win_rate "胜率"
        float avg_profit "平均盈利"
        float avg_loss "平均亏损"
        float profit_factor "盈利因子"
        float sharpe_ratio "夏普比率"
        float max_drawdown "最大回撤"
        float daily_return "日收益率"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
    }

    BACKTEST_SESSIONS {
        string id PK "回话会话唯一标识"
        string user_id FK "所属用户ID"
        string name "回话会话名称"
        string ai_model_id FK "AI模型ID"
        string exchange_id FK "交易所配置ID"
        date start_date "回话开始日期"
        date end_date "回话结束日期"
        float initial_balance "初始资金"
        json config "回话配置参数"
        string status "会话状态(running/completed/failed)"
        timestamp created_at "创建时间"
        timestamp completed_at "完成时间"
    }

    BACKTEST_RESULTS {
        string id PK "回话结果唯一标识"
        string session_id FK "回话会话ID"
        string symbol "交易币种"
        string decision_type "决策类型"
        float quantity "数量"
        float entry_price "入场价格"
        float exit_price "出场价格"
        float pnl "盈亏金额"
        float pnl_pct "盈亏百分比"
        timestamp decision_time "决策时间"
        timestamp execution_time "执行时间"
        text reasoning "决策推理"
    }

    SIGNAL_SOURCES {
        string id PK "信号源唯一标识"
        string user_id FK "所属用户ID"
        string name "信号源名称"
        string coin_pool_url "币种池API URL"
        string oi_top_url "持仓量TOP API URL"
        boolean enabled "是否启用"
        json config "信号源配置"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
    }

    SYSTEM_CONFIGS {
        string key PK "配置键名"
        string value "配置值"
        string description "配置描述"
        string config_type "配置类型(system/trading/ui)"
        timestamp created_at "创建时间"
        timestamp updated_at "更新时间"
    }

    DECISION_LOGS {
        string id PK "决策日志唯一标识"
        string trader_id FK "交易员ID"
        string log_file_path "日志文件路径"
        text decision_content "决策完整内容"
        text ai_response "AI响应内容"
        text market_snapshot "市场快照数据"
        text performance_feedback "性能反馈数据"
        timestamp created_at "创建时间"
    }

    // 关系定义
    USERS ||--o{ AI_MODELS : "拥有"
    USERS ||--o{ EXCHANGES : "拥有"
    USERS ||--o{ TRADERS : "拥有"
    USERS ||--o{ BACKTEST_SESSIONS : "拥有"
    USERS ||--o{ SIGNAL_SOURCES : "拥有"

    AI_MODELS ||--o{ TRADERS : "配置给"
    AI_MODELS ||--o{ BACKTEST_SESSIONS : "用于"

    EXCHANGES ||--o{ TRADERS : "配置给"
    EXCHANGES ||--o{ BACKTEST_SESSIONS : "用于"

    TRADERS ||--o{ TRADING_DECISIONS : "生成"
    TRADERS ||--o{ POSITIONS : "持有"
    TRADERS ||--o{ PERFORMANCE_STATS : "产生"
    TRADERS ||--o{ DECISION_LOGS : "记录"

    BACKTEST_SESSIONS ||--o{ BACKTEST_RESULTS : "包含"

    SIGNAL_SOURCES ||--o{ TRADERS : "提供给"
```

## 实体关系说明

### 核心业务实体
1. **USERS (用户)**: 系统的用户实体，支持多用户模式
2. **AI_MODELS (AI模型)**: 用户配置的各种AI模型(DeepSeek、Qwen、GLM等)
3. **EXCHANGES (交易所)**: 用户配置的交易所API信息
4. **TRADERS (交易员)**: 核心业务实体，结合AI模型和交易所的自动交易策略

### 交易相关实体
5. **TRADING_DECISIONS (交易决策)**: AI生成的每个交易决策记录
6. **POSITIONS (持仓)**: 当前持仓和历史持仓记录
7. **PERFORMANCE_STATS (性能统计)**: 交易员的历史性能统计数据

### 回测相关实体
8. **BACKTEST_SESSIONS (回测会话)**: 回测任务配置和执行状态
9. **BACKTEST_RESULTS (回测结果)**: 详细的回测交易记录

### 系统支持实体
10. **SIGNAL_SOURCES (信号源)**: 外部数据源配置
11. **SYSTEM_CONFIGS (系统配置)**: 全局系统配置参数
12. **DECISION_LOGS (决策日志)**: 完整的AI决策过程日志

### 关系类型说明
- **一对一 (1:1)**: 一个实体对应另一个实体
- **一对多 (1:n)**: 一个实体可以关联多个另一个实体
- **多对多 (n:n)**: 通过中间表实现的多对多关系

### 数据流向
1. 用户配置AI模型和交易所 → 创建交易员
2. 交易员基于AI决策 → 生成交易决策记录
3. 执行交易决策 → 创建/更新持仓记录
4. 定期统计 → 更新性能统计数据
5. 完整的决策过程 → 记录到决策日志