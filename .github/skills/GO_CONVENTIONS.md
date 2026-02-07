# Go 代码规范

本文档定义了项目中 Go 代码的编码规范和最佳实践。

## 目录结构

```
├── cmd/              # 命令行应用入口
├── internal/         # 内部包（不对外暴露）
│   ├── application/  # 应用层（用例、端口）
│   ├── domain/       # 领域层（业务逻辑）
│   └── infrastructure/ # 基础设施层
├── configs/          # 配置文件
├── data/             # 数据文件
└── tests/            # 测试文件
```

## 命名规范

### 包名
- 使用小写，单数形式：`exchange`, `logger`, `storage`
- 避免在包名中添加前缀或后缀：`util`, `helper`, `base`
- 避免冲突：使用更具体的名称，如 `sqlstorage` 而非 `storage`

### 文件名
- 使用小写和下划线：`ws_client.go`, `postgres_repo.go`
- 测试文件使用 `_test.go` 后缀：`service_test.go`
- 避免使用大写字母

### 函数和方法
- 使用 PascalCase 表示公开函数：`NewClient()`, `GetPrice()`
- 使用 camelCase 表示私有函数：`parseMessage()`, `validateInput()`
- 首选具体的名称：`GetExchangePrice()` 而非 `Get()`

### 变量和常量
- 使用 camelCase：`priceData`, `userID`
- 常量使用 SCREAMING_SNAKE_CASE：`MAX_RETRIES`, `DEFAULT_TIMEOUT`
- 避免使用单字母变量（循环除外）

## 代码风格

### 缩进和格式
- 使用 `gofmt` 自动格式化代码
- 一行最多 120 字符
- 导入分组：标准库、第三方库、项目内部包

### 导入顺序
```go
import (
    // 标准库
    "context"
    "fmt"
    "sync"
    
    // 第三方库
    "github.com/example/package"
    
    // 项目内部
    "xarb/internal/domain/model"
)
```

### 错误处理
- 始终检查错误，不要忽略返回值
- 返回错误而不是打印日志：
  ```go
  return fmt.Errorf("failed to connect: %w", err)
  ```
- 定义自定义错误类型用于特定错误场景

### 接口设计
- 定义精简的接口：通常 1-3 个方法
- 使用接口来解耦依赖关系
- 接口名称应该以 `er` 结尾：`Reader`, `Writer`, `Closer`

```go
type EventBus interface {
    Publish(ctx context.Context, event interface{}) error
    Subscribe(ctx context.Context, handler EventHandler) error
}
```

### 结构体
- 导出字段应该有注释
- 使用组合而非继承
- 避免过大的结构体

```go
// User 表示系统中的用户
type User struct {
    ID    string    // 用户唯一标识
    Name  string    // 用户名称
    Email string    // 用户邮箱
}
```

## 并发编程

### Goroutines 和 Channels
- 使用 context 管理生命周期：
  ```go
  func (s *Service) Start(ctx context.Context) error {
      go func() {
          <-ctx.Done()
          s.Stop()
      }()
      return nil
  }
  ```
- 避免使用全局变量进行状态共享
- 使用 sync.Mutex 保护共享资源

### Context 使用
- 将 context 作为第一个参数传递
- 使用 context 超时来防止资源泄漏
- 尊重 context 的取消信号

## 测试

### 单元测试
- 测试文件与源文件同目录
- 使用表格驱动测试：
  ```go
  func TestParseSymbol(t *testing.T) {
      tests := []struct {
          input    string
          expected string
          err      bool
      }{
          {"BTCUSDT", "BTC", false},
          {"", "", true},
      }
      
      for _, tt := range tests {
          t.Run(tt.input, func(t *testing.T) {
              // 测试逻辑
          })
      }
  }
  ```

### 覆盖率
- 目标覆盖率：80%+
- 重点测试业务逻辑和错误处理路径

## 文档

### 包文档
```go
// Package exchange 提供与加密货币交易所的连接
package exchange
```

### 公开函数文档
```go
// GetPrice 获取指定交易对的当前价格
// 返回的 Price 包含时间戳和交易对信息
func GetPrice(ctx context.Context, symbol string) (*Price, error) {
}
```

## 依赖注入

- 使用构造函数注入：
  ```go
  func NewService(logger Logger, repo Repository) *Service {
      return &Service{
          logger: logger,
          repo:   repo,
      }
  }
  ```

## 日志

- 使用项目配置的 Logger 接口
- 避免使用 fmt.Println 输出日志
- 适当记录错误和重要事件

## 性能考虑

- 避免在热路径中分配大量内存
- 使用对象池来管理频繁创建的对象
- 定期使用 pprof 进行性能分析

## 常见错误

❌ **不要做：**
- 忽略错误返回值
- 使用全局变量存储状态
- 导出不必要的类型和函数
- 在包级别初始化中执行复杂操作

✅ **要做：**
- 显式处理错误
- 使用依赖注入
- 仅导出必要的公开 API
- 在初始化函数中接收 context 参数

## 工具

推荐使用以下工具维护代码质量：

```bash
# 格式化代码
go fmt ./...

# 代码检查
golangci-lint run

# 生成测试覆盖率
go test -cover ./...

# 性能测试
go test -bench=. ./...
```

## 版本要求

- Go 1.21+
