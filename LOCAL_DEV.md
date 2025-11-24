# 本地开发启动指南

## 快速开始

使用 `start_local.sh` 脚本可以直接在本地运行 NOFX 系统，无需 Docker。

### 启动服务

```bash
./start_local.sh start
```

这将启动：
- **后端服务**: http://localhost:8080
- **前端服务**: http://localhost:5173 (Vite 开发服务器)

### 停止服务

```bash
./start_local.sh stop
```

### 重启服务

```bash
./start_local.sh restart
```

### 查看状态

```bash
./start_local.sh status
```

### 查看日志

```bash
# 查看所有日志
./start_local.sh logs

# 只查看后端日志
./start_local.sh logs backend

# 只查看前端日志
./start_local.sh logs frontend
```

## 服务端口

| 服务 | 端口 | URL |
|------|------|-----|
| 前端 (Vite Dev Server) | 5173 | http://localhost:5173 |
| 后端 (Go API Server) | 8080 | http://localhost:8080 |

## 日志文件

- `backend.log` - 后端服务日志
- `frontend.log` - 前端服务日志

## 前提条件

确保已安装：
- Go 1.20+
- Node.js 18+
- npm

## 与 Docker 版本的区别

### 本地开发模式 (`./start_local.sh`)

**优点**:
- ✅ 快速启动，无需构建镜像
- ✅ 代码修改立即生效（热重载）
- ✅ 方便调试
- ✅ 资源占用少

**缺点**:
- ❌ 需要本地安装 Go 和 Node.js
- ❌ 环境可能不一致

**适用场景**: 
- 日常开发调试
- 快速测试功能
- 前端开发

### Docker 模式 (`./start.sh`)

**优点**:
- ✅ 环境一致性
- ✅ 生产环境接近
- ✅ 易于部署

**缺点**:
- ❌ 构建时间长
- ❌ 修改需要重新构建
- ❌ 资源占用较多

**适用场景**:
- 生产部署
- 测试完整构建
- 多人协作

## 常见问题

### 端口被占用

如果端口被占用，可以：

```bash
# 检查占用 8080 端口的进程
lsof -i :8080

# 检查占用 5173 端口的进程  
lsof -i :5173

# 杀死进程
kill -9 <PID>
```

### 前端依赖问题

```bash
cd web
npm install
cd ..
./start_local.sh start
```

### 后端编译问题

```bash
go mod download
go mod tidy
./start_local.sh start
```

## 开发工作流建议

1. **启动本地服务**:
   ```bash
   ./start_local.sh start
   ```

2. **进行开发和测试**:
   - 前端修改会自动热重载
   - 后端修改需要重启: `./start_local.sh restart`

3. **完成后停止服务**:
   ```bash
   ./start_local.sh stop
   ```

4. **提交前测试 Docker 构建**:
   ```bash
   ./start.sh build
   ./start.sh start
   ```
