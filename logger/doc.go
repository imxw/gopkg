// Package logger 提供生产级别的 Zap 日志封装，支持：
// 1. 多环境适配（production/development/test）；
// 2. 动态调整日志级别；
// 3. 上下文 TraceID 传递；
// 4. 日志轮转（文件输出自动切分/压缩）；
// 5. 兼容 Zap 原生 API，同时封装简化使用。
//
// 快速开始：
//
//  1. 初始化 Logger：
//     logger.Init() // 使用默认配置（开发环境）
//     // 或自定义生产环境配置：
//     logger.Init(
//     func(opts *logger.Options) {
//     opts.Env = "production"
//     opts.OutputPaths = []string{"./logs/app.log"}
//     },
//     )
//
//  2. 基础日志使用：
//     logger.Infow("app started", "pid", os.Getpid())
//     logger.CtxErrorw(ctx, "request failed", "error", err)
//
//  3. 动态调整级别：
//     logger.SetLevel(zapcore.DebugLevel)
//
//  4. 优雅退出：
//     defer logger.Sync()
package logger
