# gopkg

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/imxw/gopkg.svg)](https://pkg.go.dev/github.com/imxw/gopkg)

Go 后端服务公共库：统一错误协议、HTTP 信封与引擎、结构化日志、JWT 生命周期、pgx 类型转换与 panic 安全 goroutine。为 Gin + pgx + sqlc 形态的服务而建，各包零相互纠缠（除 ginx→contextx/errorx/logger 的组合出口），可按需单包引用。

## 安装

```bash
go get github.com/imxw/gopkg@latest
```

要求 Go ≥ 1.26（库内使用 `errors.AsType`）。

## 包总览

| 包 | 用途 | 入口 |
|----|------|------|
| `errorx` | 业务错误协议：业务码 + HTTP 状态 + 稳定 NAME + 不可变副本语义 | `errorx.New` / 哨兵 `ErrNotFound` 等 |
| `ginx` | HTTP 信封（`{code,msg,data,error,requestId,details}`）、路由引擎（中间件链内建）、参数绑定与分页 | `ginx.Error` / `ginx.NewEngine` / `ginx.BindJSON` |
| `contextx` | 请求上下文身份存取（requestID/userID/userName/orgID），int 私有键防碰撞 | `contextx.UserID(ctx)` |
| `logger` | zap 全局日志：结构化快捷函数、Ctx* 上下文关联、轮转配置、Init 一次性语义 | `logger.Init` / `logger.CtxInfow` |
| `jwt` | Token 生命周期：双 token 对（org 绑定）、黑名单吊销、刷新轮换（原子反重放） | `jwt.NewTokenManager` |
| `jwt/redisblacklist` | Redis 黑名单实现（单机/Cluster/哨兵通用），SET NX 原子占位 | `redisblacklist.New` |
| `gosafe` | panic 安全 goroutine（带命名标签落日志） | `gosafe.Go` / `gosafe.GoNamed` |
| `pgxutil` | pgtype ↔ Go 值转换（nil 安全、往返全覆盖）+ pgx 错误 → 业务错误翻译 | `pgxutil.ToPgText` / `pgxutil.MapPgxError` |
| `validatorx` | 中文校验消息 + gin binding 适配（import 即替换全局 validator，见包内 WARNING） | `validatorx.ReplaceGinValidator` |
| `statemachine` | 内存态转换合法性查询器（纯查询，无 hook/持久化） | `statemachine.NewEngine` |

## 快速上手

### 错误协议（errorx + ginx）

```go
// domain 层：包级哨兵（码段自行规划，避免撞号）
var ErrAppNotFound = errorx.New(5601, http.StatusNotFound, "APP_NOT_FOUND", "app not found")

// repo 层：pgx 错误 → 业务错误（一行替换 ErrNoRows/约束违反三步翻译）
app, err := repo.GetApp(ctx, id)
if err != nil {
    return nil, pgxutil.MapPgxError(err,
        fmt.Errorf("get app %d: %w", id, domain.ErrAppNotFound),
        pgxutil.WithUniqueViolationOn("apps_name_key", domain.ErrAppNameConflict))
}

// handler 层：统一信封出口（5xx 自动落 ERROR 日志，requestId 自动关联）
if err != nil {
    ginx.Error(c, err)
    return
}
// 结构化错误载荷（校验报告等）走 details 专列
ginx.ErrorWithDetails(c, err, map[string]any{"errors": report.Errors})
```

### HTTP 引擎（ginx）

```go
r := ginx.NewEngine(ginx.DefaultEngineConfig())
// 链序内建：SecurityHeaders → RequestID → Recovery → AccessLog → BodyLimit → CORS
r.POST("/apps", createApp)
```

RequestID 中间件接受合法入站 `X-Request-Id`（8-128 位字母数字/-/_）跨服务串联，并回写响应头；上下文读取统一走 `ginx.RequestIDFrom(c)`。

### JWT 生命周期（jwt + redisblacklist）

```go
bl := redisblacklist.New(redisClient, "myapp:token_blacklist") // redis.UniversalClient 通用
tm, err := jwt.NewTokenManager(jwt.Config{
    Secret:        secretAtLeast32Bytes, // 签名强制校验下限
    Issuer:        "myapp",
    TokenExpire:   2 * time.Hour,
    RefreshExpire: 7 * 24 * time.Hour,
}, bl)

pair, _ := tm.GenerateTokenPairInOrg(ctx, userID, userName, orgID)
pair2, err := tm.RefreshAccessToken(ctx, pair.RefreshToken)
// err == jwt.ErrTokenInBlacklist ⇒ 同一 refresh token 重放（轮换原子占位，并发安全）
```

### 日志（logger）

```go
logger.Init() // 未显式设置级别时按环境兜底（development→debug / test→error / production→info）
logger.Init(logger.WithLevel(zapcore.InfoLevel)) // 显式钉级别：不被环境适配覆盖
logger.CtxInfow(ctx, "app created", logger.StringField("app_id", id))
defer logger.Sync()
```

## 版本

版本随语义化发布：修复 → patch；新增 API → minor。当前推荐 `v0.10.0`。

## License

[MIT](LICENSE)
