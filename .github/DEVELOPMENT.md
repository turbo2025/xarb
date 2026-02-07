# Superpowers-Driven Development

本文档说明如何使用 Superpowers 方法论开发 xarb 项目。

## 工作流

### 1. 功能规划（Brainstorming）

**触发方式**：
```
Help me plan: [feature description]
```

**期望结果**：
- ✅ 通过 Socratic 问答精化需求
- ✅ 生成设计文档 `.github/designs/[feature].md`
- ✅ 分块展示设计供批准
- ✅ 验证关键决策

**示例**：
```
Help me plan: Add Prometheus metrics to monitor bot

Claude will ask:
- What metrics matter most for trading signals?
- Should we track per-exchange or aggregated?
- Alert thresholds?
- Retention policy?
```

### 2. 创建工作分支（Git Worktrees）

**自动激活**：
```bash
# 在 git worktree 上工作，保持 main 干净
git worktree add ../xarb-feature feature-branch
cd ../xarb-feature
```

**特点**：
- 隔离工作环境
- 并行多个特性开发
- 自动验证测试基线

### 3. 任务分解（Writing Plans）

**期望输出**：
```markdown
# Implementation Plan: Prometheus Metrics

## Task 1: Setup Prometheus client
- File: internal/infrastructure/monitoring/prometheus.go
- Expected: PrometheusRegistry type
- Verify: go test -run TestPrometheusInit

## Task 2: Define metrics
- File: internal/domain/metrics/metrics.go
- Expected: MetricsDefinition interface
- Verify: 覆盖所有交易对

## Task 3: Integration with monitor service
- File: internal/application/usecase/monitor/service.go
- Expected: 集成 Prometheus 埋点
- Verify: metrics exposed on /metrics endpoint
```

**关键点**：
- 2-5 分钟任务粒度
- 完整文件路径
- 精确的验证步骤
- 明确的验收标准

### 4. 实现执行（Subagent-Driven Development）

**流程**：
1. 为每个任务创建独立子代理
2. 两阶段审查：
   - 第1阶段：规格合规性（代码符合计划吗？）
   - 第2阶段：代码质量（是否遵循规范？）
3. 持续向前（关键问题阻止进度）

**你的责任**：
- ✅ 批准每个任务
- ✅ 审查代码质量
- ✅ 反馈任何偏差

### 5. 测试驱动开发（TDD）

**强制循环**：
```go
// 1. RED: 写失败的测试
func TestPrometheusMetricsExport(t *testing.T) {
    registry := NewPrometheusRegistry()
    metrics := registry.Export()
    assert.Contains(t, metrics, "price_spread")
}
// 运行: 失败 ❌

// 2. GREEN: 写最小实现
func (r *PrometheusRegistry) Export() map[string]interface{} {
    return map[string]interface{}{
        "price_spread": r.spreadMetric,
    }
}
// 运行: 成功 ✅

// 3. REFACTOR: 改进代码
// 提取到 metrics.go，添加注释，优化结构
```

**规则**：
- ✅ 先写测试，后写代码（无例外）
- ✅ 测试失败后才写实现
- ✅ 删除测试前写的代码
- ✅ 目标覆盖率：80%+

### 6. 代码审查（Code Review）

**自动检查**：
- ✅ 代码符合计划吗？
- ✅ 遵循 Go 规范吗？（[GO_CONVENTIONS.md](skills/GO_CONVENTIONS.md)）
- ✅ 机器人规范吗？（[BOT_CONVENTIONS.md](skills/BOT_CONVENTIONS.md)）
- ✅ 有充分的测试吗？
- ✅ 文档齐全吗？

**严重性等级**：
- 🔴 **CRITICAL**：阻止合并（比如：安全漏洞、无测试）
- 🟡 **WARNING**：需要修复（比如：代码风格）
- 🔵 **INFO**：建议改进（比如：可读性）

### 7. 完成开发分支（Finishing Branch）

**选项**：
1. 合并到 main
2. 创建 Pull Request 供评审
3. 保留分支继续开发
4. 丢弃分支

**清理**：
- 删除 git worktree
- 验证所有测试通过
- 更新文档

---

## 实战示例

### 场景：添加 Prometheus 监控

```
你：Help me plan: Add Prometheus metrics for price spread monitoring

Claude Brainstorming:
Q1: Should we track absolute spread or percentage?
Q2: Per-exchange or aggregated?
Q3: Alert thresholds?

你：Approve design

Claude Writing Plan:
Task 1: internal/infrastructure/monitoring/prometheus.go (Registry)
Task 2: internal/domain/metrics/metrics.go (Definitions)
Task 3: internal/application/usecase/monitor/service.go (Integration)
Task 4: configs/config.toml (Prometheus config)
Task 5: tests for all components

你：Approve plan, start development

Claude Subagent-Driven:
- Dispatch 5 subagents for 5 tasks
- Each writes tests first (RED)
- Implements feature (GREEN)
- Refactors code (REFACTOR)
- You review and approve
- Auto-merge when all tasks complete
```

---

## 项目特定规则

### Redis & SQLite 集成

**设计原则**：
- ✅ 通过 Container 访问依赖（[CONTAINER_PATTERN.md](skills/CONTAINER_PATTERN.md)）
- ✅ 所有数据操作都有测试
- ✅ 使用 Repository 模式隔离存储
- ✅ 错误处理显式（不忽略错误）

### 多交易所支持

**设计原则**：
- ✅ 新交易所 = 新的 `parser.go` + `ws_client.go`
- ✅ 实现 `PriceFeed` 接口
- ✅ 100% 单元测试覆盖
- ✅ 集成测试验证 WebSocket 连接

### 配置管理

**设计原则**：
- ✅ 所有配置在 `configs/config.toml`
- ✅ 敏感信息用环境变量覆盖
- ✅ 启动时验证配置完整性
- ✅ 日志输出加载的配置（非敏感部分）

---

## 检查清单

### 开始新功能前

- [ ] 需求已通过 Brainstorming 精化
- [ ] 设计文档已批准
- [ ] 计划已分解为原子任务
- [ ] 每个任务都有验证步骤

### 任务实现时

- [ ] ✅ 先写失败的测试（RED）
- [ ] ✅ 写最小实现（GREEN）
- [ ] ✅ 重构改进代码（REFACTOR）
- [ ] ✅ 遵循 Go 规范
- [ ] ✅ 通过代码审查

### 完成功能后

- [ ] 所有测试通过（coverage >= 80%）
- [ ] 文档已更新
- [ ] 无 CRITICAL 代码审查问题
- [ ] PR 已创建或已合并
- [ ] 分支已清理

---

## 常见触发词

使用这些短语自动激活相关技能：

| 短语 | 激活技能 |
|------|---------|
| "help me plan" | brainstorming + writing-plans |
| "let's debug" | systematic-debugging |
| "write tests for" | test-driven-development |
| "review this code" | requesting-code-review |
| "implement this" | subagent-driven-development |
| "finish up" | finishing-a-development-branch |

---

## 参考资源

- [Superpowers GitHub](https://github.com/obra/superpowers)
- [Go 代码规范](.github/skills/GO_CONVENTIONS.md)
- [机器人规范](.github/skills/BOT_CONVENTIONS.md)
- [容器模式](.github/skills/CONTAINER_PATTERN.md)

---

## 成功标志

✅ 你知道什么时候成功了吗？

- 新功能有 80%+ 测试覆盖
- 代码符合 Go 规范和机器人规范
- 所有代码审查都通过
- 文档和代码同步更新
- 能信心满满地部署到生产

**Happy coding with Superpowers! 🚀**
