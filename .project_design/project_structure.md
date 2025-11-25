# NOFX 项目完整文件夹结构

## 项目根目录结构

```
nofx/
├── .project_design/                    # 项目设计文档
│   ├── README.md                       # 项目总体设计说明
│   ├── system_erd.md                   # 系统完整 ERD 图
│   ├── main_framework.md               # 主函数设计
│   ├── project_structure.md            # 项目文件夹结构
│   └── modules/                        # 各模块详细设计
│       ├── trader_manager/             # 交易员管理器模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── auto_trader/                # 自动交易器模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── api_server/                 # API服务器模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── database/                   # 数据库模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── ai_decision_engine/         # AI决策引擎模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── market_data/                # 市场数据模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── risk_management/            # 风险管理模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── authentication/             # 认证授权模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       ├── notification/               # 通知服务模块
│       │   ├── design.md               # 模块设计说明
│       │   ├── logic_flow.md           # 逻辑流程图
│       │   └── pseudocode.md           # 详细伪代码
│       └── backtest_system/            # 回测系统模块
│           ├── design.md               # 模块设计说明
│           ├── logic_flow.md           # 逻辑流程图
│           └── pseudocode.md           # 详细伪代码
├── main.go                             # 主程序入口
├── go.mod                              # Go模块依赖
├── go.sum                              # 依赖版本锁定
├── config.json                         # 系统配置文件
├── config.json.example                 # 配置文件模板
├── .env                                # 环境变量
├── .env.example                        # 环境变量模板
├── docker-compose.yml                  # Docker编排文件
├── Dockerfile                          # Docker镜像构建文件
├── .dockerignore                       # Docker忽略文件
├── README.md                           # 项目说明文档
├── CHANGELOG.md                        # 版本更新日志
├── LICENSE                             # 开源许可证
├── docs/                               # 项目文档
│   ├── README.md                       # 文档索引
│   ├── getting-started/                # 快速开始指南
│   │   ├── README.md                   # 快速开始说明
│   │   ├── installation.md             # 安装指南
│   │   ├── configuration.md            # 配置说明
│   │   └── first-trade.md              # 首次交易指南
│   ├── user-guide/                     # 用户指南
│   │   ├── README.md                   # 用户指南索引
│   │   ├── dashboard.md                # 仪表板使用
│   │   ├── trader-management.md        # 交易员管理
│   │   ├── ai-models.md                # AI模型配置
│   │   ├── exchanges.md                # 交易所配置
│   │   └── risk-management.md          # 风险管理
│   ├── developer-guide/                 # 开发者指南
│   │   ├── README.md                   # 开发者指南索引
│   │   ├── architecture.md             # 系统架构
│   │   ├── api-reference.md            # API参考
│   │   ├── database-schema.md          # 数据库结构
│   │   ├── contributing.md             # 贡献指南
│   │   └── testing.md                  # 测试指南
│   ├── deployment/                      # 部署指南
│   │   ├── README.md                   # 部署指南索引
│   │   ├── docker.md                   # Docker部署
│   │   ├── cloud.md                    # 云服务器部署
│   │   ├── ssl.md                      # SSL配置
│   │   └── monitoring.md               # 监控配置
│   └── troubleshooting/                 # 故障排除
│       ├── README.md                   # 故障排除索引
│       ├── common-issues.md            # 常见问题
│       ├── debugging.md                # 调试指南
│       └── performance.md              # 性能优化
├── api/                                # API服务层
│   ├── server.go                       # API服务器主文件
│   ├── middleware.go                   # 中间件定义
│   ├── handlers/                       # 请求处理器
│   │   ├── auth.go                     # 认证处理器
│   │   ├── trader.go                   # 交易员处理器
│   │   ├── models.go                   # AI模型处理器
│   │   ├── exchanges.go                # 交易所处理器
│   │   ├── market.go                   # 市场数据处理器
│   │   ├── backtest.go                 # 回测处理器
│   │   └── system.go                   # 系统配置处理器
│   ├── models/                         # API数据模型
│   │   ├── request.go                  # 请求模型
│   │   ├── response.go                 # 响应模型
│   │   ├── common.go                   # 通用模型
│   │   └── validation.go               # 数据验证
│   ├── routes.go                       # 路由定义
│   ├── utils.go                        # 工具函数
│   ├── utils_test.go                   # 工具函数测试
│   └── crypto_handler.go               # 加密处理器
├── manager/                            # 管理层
│   ├── trader_manager.go               # 交易员管理器
│   ├── user_manager.go                 # 用户管理器
│   ├── config_manager.go               # 配置管理器
│   └── performance_manager.go          # 性能管理器
├── trader/                             # 交易执行层
│   ├── interface.go                    # 交易器接口定义
│   ├── auto_trader.go                  # 自动交易器
│   ├── binance_futures.go              # 币安期货实现
│   ├── hyperliquid_trader.go           # Hyperliquid实现
│   ├── aster_trader.go                 # Aster DEX实现
│   ├── paper_trader.go                 # 模拟交易实现
│   ├── risk_manager.go                 # 风险管理器
│   ├── position_manager.go             # 持仓管理器
│   └── order_manager.go                # 订单管理器
├── ai/                                 # AI决策引擎
│   ├── engine.go                       # AI决策引擎
│   ├── clients/                        # AI模型客户端
│   │   ├── interface.go                # 客户端接口
│   │   ├── deepseek.go                 # DeepSeek客户端
│   │   ├── qwen.go                     # Qwen客户端
│   │   ├── glm.go                      # GLM客户端
│   │   └── custom.go                   # 自定义模型客户端
│   ├── prompts/                        # 提示词模板
│   │   ├── system_prompt.go            # 系统提示词
│   │   ├── trading_prompt.go           # 交易提示词
│   │   ├── analysis_prompt.go          # 分析提示词
│   │   └── custom_prompt.go            # 自定义提示词
│   ├── parser.go                       # 响应解析器
│   ├── validator.go                    # 决策验证器
│   └── performance_analyzer.go         # 性能分析器
├── market/                             # 市场数据层
│   ├── monitor.go                      # 市场监控器
│   ├── websocket_client.go             # WebSocket客户端
│   ├── data_processor.go               # 数据处理器
│   ├── indicators/                     # 技术指标
│   │   ├── interface.go                # 指标接口
│   │   ├── ema.go                      # EMA指标
│   │   ├── macd.go                     # MACD指标
│   │   ├── rsi.go                      # RSI指标
│   │   ├── atr.go                      # ATR指标
│   │   ├── bollinger_bands.go          # 布林带指标
│   │   └── volume_indicator.go         # 成交量指标
│   ├── timeframes/                     # 时间框架
│   │   ├── manager.go                  # 时间框架管理器
│   │   ├── m1.go                       # 1分钟K线
│   │   ├── m5.go                       # 5分钟K线
│   │   ├── m15.go                      # 15分钟K线
│   │   ├── h1.go                       # 1小时K线
│   │   ├── h4.go                       # 4小时K线
│   │   └── d1.go                       # 日K线
│   ├── liquidity/                      # 流动性分析
│   │   ├── analyzer.go                 # 流动性分析器
│   │   ├── order_book.go               # 订单簿分析
│   │   └── funding_rate.go             # 资金费率分析
│   └── sentiment/                       # 市场情绪
│       ├── analyzer.go                 # 情绪分析器
│       ├── social_sentiment.go         # 社交媒体情绪
│       └── fear_greed_index.go         # 恐惧贪婪指数
├── config/                             # 配置管理层
│   ├── database.go                     # 数据库操作
│   ├── config.go                       # 配置管理
│   ├── models.go                       # 配置模型
│   ├── migrations/                     # 数据库迁移
│   │   ├── migration.go                # 迁移管理器
│   │   ├── v1_init.sql                 # 初始化脚本
│   │   ├── v2_add_signals.sql          # 信号源表
│   │   └── v3_add_backtest.sql         # 回测表
│   ├── database_test.go                # 数据库测试
│   └── encryption.go                   # 加密配置
├── auth/                               # 认证授权层
│   ├── auth.go                         # 认证主逻辑
│   ├── jwt.go                          # JWT令牌管理
│   ├── middleware.go                   # 认证中间件
│   ├── rbac.go                         # 角色权限控制
│   ├── otp.go                          # 双因子认证
│   └── session.go                      # 会话管理
├── crypto/                             # 加密服务层
│   ├── crypto.go                       # 加密主接口
│   ├── encryption.go                   # 加密实现
│   ├── decryption.go                   # 解密实现
│   ├── key_management.go               # 密钥管理
│   ├── secure_storage.go               # 安全存储
│   └── encryption_test.go              # 加密测试
├── logger/                             # 日志系统
│   ├── logger.go                       # 日志主文件
│   ├── config.go                       # 日志配置
│   ├── formatters.go                   # 日志格式化
│   ├── telegram_sender.go              # Telegram通知
│   ├── decision_logger.go              # 决策日志
│   ├── telegram_hook.go                # Telegram钩子
│   └── webhook.go                      # Webhook通知
├── backtest/                           # 回测系统
│   ├── engine.go                       # 回测引擎
│   ├── data_loader.go                  # 历史数据加载器
│   ├── simulator.go                    # 交易模拟器
│   ├── performance_analyzer.go         # 性能分析器
│   ├── report_generator.go             # 报告生成器
│   ├── handlers.go                     # 回测API处理器
│   └── models.go                       # 回测数据模型
├── decision/                           # 决策记录
│   ├── recorder.go                     # 决策记录器
│   ├── storage.go                      # 存储管理器
│   ├── analyzer.go                     # 决策分析器
│   └── exporter.go                     # 导出器
├── pool/                               # 币种池管理
│   ├── pool.go                         # 币种池主文件
│   ├── manager.go                      # 币种池管理器
│   ├── filter.go                       # 币种过滤器
│   ├── api_client.go                   # API客户端
│   └── cache.go                        # 缓存管理
├── prompts/                            # 提示词管理
│   ├── manager.go                      # 提示词管理器
│   ├── templates/                      # 提示词模板
│   │   ├── base_prompt.go              # 基础提示词
│   │   ├── trading_prompt.go           # 交易提示词
│   │   ├── analysis_prompt.go          # 分析提示词
│   │   └── risk_prompt.go              // 风险提示词
│   ├── builder.go                      # 提示词构建器
│   └── validator.go                    # 提示词验证器
├── utils/                              # 工具函数
│   ├── time.go                         # 时间工具
│   ├── math.go                         # 数学工具
│   ├── validation.go                   # 验证工具
│   ├── format.go                       # 格式化工具
│   ├── http.go                         # HTTP工具
│   ├── cache.go                        # 缓存工具
│   └── retry.go                        # 重试工具
├── web/                                # 前端应用
│   ├── package.json                    # 前端依赖配置
│   ├── tsconfig.json                   # TypeScript配置
│   ├── vite.config.ts                  # Vite构建配置
│   ├── tailwind.config.js              # TailwindCSS配置
│   ├── index.html                      # 入口HTML文件
│   ├── src/                            # 源代码
│   │   ├── main.tsx                    # 主入口文件
│   │   ├── App.tsx                     # 主应用组件
│   │   ├── components/                 # 组件库
│   │   │   ├── common/                 # 通用组件
│   │   │   │   ├── Button.tsx          # 按钮组件
│   │   │   │   ├── Modal.tsx           # 模态框组件
│   │   │   │   ├── Table.tsx           # 表格组件
│   │   │   │   ├── Chart.tsx           # 图表组件
│   │   │   │   ├── Loading.tsx         # 加载组件
│   │   │   │   └── ErrorBoundary.tsx   # 错误边界组件
│   │   │   ├── trader/                 # 交易员组件
│   │   │   │   ├── TraderCard.tsx      # 交易员卡片
│   │   │   │   ├── TraderList.tsx      # 交易员列表
│   │   │   │   ├── TraderForm.tsx      # 交易员表单
│   │   │   │   ├── TraderDetails.tsx   # 交易员详情
│   │   │   │   └── TraderStats.tsx     # 交易员统计
│   │   │   ├── dashboard/              # 仪表板组件
│   │   │   │   ├── Overview.tsx        # 概览面板
│   │   │   │   ├── Performance.tsx     # 性能面板
│   │   │   │   ├── Positions.tsx       # 持仓面板
│   │   │   │   ├── Decisions.tsx       # 决策面板
│   │   │   │   └── Competition.tsx     # 竞赛面板
│   │   │   ├── config/                 # 配置组件
│   │   │   │   ├── AIModelForm.tsx     # AI模型表单
│   │   │   │   ├── ExchangeForm.tsx    # 交易所表单
│   │   │   │   ├── SystemConfig.tsx    # 系统配置
│   │   │   │   └── SignalSource.tsx    # 信号源配置
│   │   │   └── backtest/               # 回测组件
│   │   │       ├── BacktestForm.tsx    # 回测表单
│   │   │       ├── BacktestResults.tsx # 回测结果
│   │   │       ├── BacktestChart.tsx   # 回测图表
│   │   │       └── PerformanceReport.tsx # 性能报告
│   │   ├── pages/                      # 页面组件
│   │   │   ├── Dashboard.tsx           # 仪表板页面
│   │   │   ├── Traders.tsx             # 交易员页面
│   │   │   ├── Competition.tsx         # 竞赛页面
│   │   │   ├── Backtest.tsx            # 回测页面
│   │   │   ├── Config.tsx              # 配置页面
│   │   │   └── Login.tsx               # 登录页面
│   │   ├── hooks/                      # 自定义Hooks
│   │   │   ├── useAuth.ts              # 认证Hook
│   │   │   ├── useWebSocket.ts         # WebSocket Hook
│   │   │   ├── useTraders.ts           # 交易员Hook
│   │   │   ├── useMarketData.ts        # 市场数据Hook
│   │   │   └── useLocalStorage.ts      # 本地存储Hook
│   │   ├── services/                   # 服务层
│   │   │   ├── api.ts                  # API服务
│   │   │   ├── auth.ts                 # 认证服务
│   │   │   ├── websocket.ts            # WebSocket服务
│   │   │   └── storage.ts              # 存储服务
│   │   ├── types/                      # 类型定义
│   │   │   ├── trader.ts               # 交易员类型
│   │   │   ├── market.ts               # 市场数据类型
│   │   │   ├── config.ts               # 配置类型
│   │   │   ├── api.ts                  # API类型
│   │   │   └── common.ts               # 通用类型
│   │   ├── utils/                      # 工具函数
│   │   │   ├── formatters.ts           # 格式化工具
│   │   │   ├── validators.ts           # 验证工具
│   │   │   ├── constants.ts            # 常量定义
│   │   │   └── helpers.ts              # 辅助函数
│   │   └── styles/                     # 样式文件
│   │       ├── globals.css             # 全局样式
│   │       └── components.css          # 组件样式
│   ├── public/                          # 静态资源
│   │   ├── favicon.ico                 # 网站图标
│   │   └── manifest.json               # PWA配置
│   └── dist/                            # 构建输出目录
├── scripts/                            # 脚本文件
│   ├── build.sh                        # 构建脚本
│   ├── deploy.sh                       # 部署脚本
│   ├── setup.sh                        # 环境设置脚本
│   ├── backup.sh                       # 备份脚本
│   ├── migrate.sh                      # 数据迁移脚本
│   ├── test.sh                         # 测试脚本
│   └── start.sh                        # 启动脚本
├── tests/                              # 测试文件
│   ├── integration/                    # 集成测试
│   ├── unit/                           # 单元测试
│   ├── e2e/                            # 端到端测试
│   ├── fixtures/                       # 测试数据
│   └── mocks/                          # Mock数据
├── logs/                               # 日志目录
│   ├── app.log                         # 应用日志
│   ├── error.log                       # 错误日志
│   ├── access.log                      # 访问日志
│   ├── trading.log                     # 交易日志
│   └── decision_logs/                  # 决策日志目录
├── data/                               # 数据目录
│   ├── backtest/                       # 回测数据
│   ├── historical/                     # 历史数据
│   ├── exports/                        # 导出数据
│   └── backups/                        # 备份数据
├── secrets/                            # 密钥目录
│   ├── rsa_key                         # RSA私钥
│   ├── rsa_key.pub                     # RSA公钥
│   └── jwt_secret                      # JWT密钥
├── docker/                             # Docker相关
│   ├── Dockerfile.api                  # API服务镜像
│   ├── Dockerfile.web                  # Web服务镜像
│   ├── docker-compose.dev.yml          # 开发环境编排
│   ├── docker-compose.prod.yml         # 生产环境编排
│   └── nginx.conf                      # Nginx配置
├── .github/                            # GitHub配置
│   ├── workflows/                      # GitHub Actions
│   │   ├── ci.yml                      # 持续集成
│   │   ├── cd.yml                      # 持续部署
│   │   └── security.yml                # 安全扫描
│   ├── ISSUE_TEMPLATE/                 # Issue模板
│   └── PULL_REQUEST_TEMPLATE.md        # PR模板
├── .gitignore                          # Git忽略文件
├── .gitattributes                      # Git属性文件
├── .editorconfig                       # 编辑器配置
├── .prettierrc                         # 代码格式化配置
├── .eslintrc.js                        # ESLint配置
├── Makefile                            # Make构建文件
├── CHANGELOG.zh-CN.md                  # 中文更新日志
├── CONTRIBUTING.md                     # 贡献指南
├── SECURITY.md                         # 安全政策
├── LICENSE                             # 开源许可证
├── CODE_OF_CONDUCT.md                  # 行为准则
├── FUNDING.yml                         # 资助配置
├── SUPPORT.md                          # 支持文档
├── CLAUDE.md                           # Claude助手配置
├── QWEN.md                             # Qwen助手配置
├── AGENTS.md                           # AI代理配置
├── OPEN_ISSUES.md                      # 开放问题列表
├── ROADMAP.md                          # 路线图
├── PERFORMANCE.md                      # 性能说明
├── SECURITY_AUDIT.md                   # 安全审计报告
├── API_DOCS.md                         # API文档
├── DEPENDENCIES.md                     # 依赖说明
├── CHANGELOG.md                        # 更新日志
├── config.multi-timeframe.example.json # 多时间框架配置示例
├── beta_codes.txt                      # 内测码文件
├── nofx.db                             # SQLite数据库文件
├── nofx.db-shm                         # SQLite共享内存文件
├── nofx.db-wal                         # SQLite预写日志文件
├── decision_logs/                      # 决策日志目录
│   ├── default/                        # 默认用户决策日志
│   └── user@example.com/               # 用户决策日志
├── database_backups/                   # 数据库备份目录
├── temp/                               # 临时文件目录
├── uploads/                            # 上传文件目录
├── screenshots/                        # 截图目录
└── vendor/                             # 依赖vendor目录
```

## 目录说明

### 核心代码目录
- **main.go**: 程序入口，负责系统初始化和启动
- **api/**: HTTP API服务层，处理所有Web请求
- **manager/**: 业务管理层，协调各个模块
- **trader/**: 交易执行层，实现具体的交易逻辑
- **ai/**: AI决策引擎，处理AI模型调用和决策生成
- **market/**: 市场数据层，获取和处理市场数据
- **config/**: 配置管理层，处理数据库和配置相关操作
- **auth/**: 认证授权层，处理用户认证和权限控制
- **crypto/**: 加密服务层，处理敏感数据加密

### 数据存储目录
- **config.db**: SQLite数据库主文件
- **logs/**: 日志文件存储目录
- **data/**: 各种数据文件存储目录
- **secrets/**: 加密密钥存储目录

### 前端应用目录
- **web/**: React前端应用完整源码
- **dist/**: 前端构建输出目录

### 部署相关目录
- **docker/**: Docker配置文件
- **scripts/**: 各种自动化脚本
- **.github/**: GitHub Actions配置

### 文档目录
- **docs/**: 完整的项目文档
- **.project_design/**: 详细的技术设计文档

### 测试目录
- **tests/**: 各类测试文件

### 临时目录
- **temp/**: 临时文件存储
- **uploads/**: 用户上传文件存储
- **decision_logs/**: AI决策日志存储

## 文件命名规范

### Go文件命名
- 使用小写字母和下划线
- 文件名与包名保持一致
- 测试文件以`_test.go`结尾

### 前端文件命名
- 组件文件使用PascalCase（如`TraderCard.tsx`）
- 工具文件使用camelCase（如`formatters.ts`）
- 样式文件使用kebab-case（如`trader-card.css`）

### 配置文件命名
- 主配置文件：`config.json`
- 模板文件：`*.example.json`
- 环境变量：`.env`和`.env.example`

## 模块依赖关系

```
main.go
├── manager/
│   ├── trader_manager.go → trader/, config/, ai/
│   ├── user_manager.go → auth/, config/
│   └── config_manager.go → config/
├── api/
│   ├── handlers/ → manager/, auth/, crypto/
│   ├── middleware/ → auth/
│   └── models/ → common types
├── trader/
│   ├── auto_trader.go → ai/, market/, config/
│   ├── risk_manager.go → market/, config/
│   └── implementations/ → exchange APIs
├── ai/
│   ├── engine.go → clients/, prompts/
│   ├── clients/ → external AI APIs
│   └── prompts/ → templates/
├── market/
│   ├── monitor.go → exchange APIs, indicators/
│   ├── indicators/ → technical analysis
│   └── websocket/ → real-time data
├── config/
│   ├── database.go → SQLite, crypto/
│   └── models.go → data structures
├── auth/
│   ├── auth.go → jwt.go, rbac.go
│   └── middleware/ → request validation
└── crypto/
    ├── encryption.go → RSA, AES
    └── secure_storage.go → file system
```

这个完整的项目结构设计提供了：

1. **清晰的模块划分**: 每个模块职责明确，依赖关系清晰
2. **标准化命名**: 统一的文件和目录命名规范
3. **完整的文档结构**: 从设计文档到用户手册的完整文档体系
4. **可扩展架构**: 支持新功能模块的快速集成
5. **开发友好**: 便于团队协作和代码维护
6. **部署就绪**: 包含完整的部署和运维支持文件