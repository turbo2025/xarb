# Superpowers Quick Start

## 启用 Superpowers 工作流

### 第一步：安装（仅需一次）

如果你使用 Claude Code：

```bash
# 在 Claude Code 中运行
/plugin marketplace add obra/superpowers-marketplace
/plugin install superpowers@superpowers-marketplace
```

### 第二步：验证安装

新建一个对话，请求帮助规划某个功能：

```
Help me plan: Add Prometheus metrics to the monitoring bot
```

Claude 应该自动激活 Brainstorming 技能。

---

## 使用模式

### 模式 1: 完整特性开发

```
1️⃣ 你: "Help me plan: [feature]"
   → Claude 执行 Brainstorming

2️⃣ 你: "Approve the design"
   → Claude 生成实现计划

3️⃣ 你: "Start implementation"
   → Claude 创建 git worktree 并开始 subagent-driven-development

4️⃣ Claude: 为每个任务创建子代理
   → 你审查并批准每个任务

5️⃣ 完成: 代码自动合并或创建 PR
```

### 模式 2: 快速修复

```
你: "Let's debug: [issue description]"
→ Claude 执行 systematic-debugging
→ 输出根因分析
→ 提议修复方案
```

### 模式 3: 代码审查

```
你: "Review this code for [component]"
→ Claude 执行 requesting-code-review
→ 按规范和计划检查
→ 输出审查报告（CRITICAL/WARNING/INFO）
```

---

## 常用命令

### 激活 Brainstorming
```
Help me plan: [description of what you want to build]
```

### 激活 TDD
```
Write tests for: [feature/component]
```

### 激活调试
```
Let's debug: [problem description]
```

### 激活代码审查
```
Review this PR against the plan
```

### 激活任务实现
```
Implement this task from the plan
```

---

## 工作流示例

### 示例：添加 Prometheus 指标

#### Step 1: Brainstorming

```
你: Help me plan: Add Prometheus metrics for arbitrage opportunities

Claude:
Q1: Should we track absolute spread (e.g., $10) or percentage (0.5%)?
Q2: Per-exchange metrics or aggregated across all exchanges?
Q3: What alert thresholds should trigger notifications?
Q4: How long should we retain metrics in storage?

你: 
- Percentage spread, because it's normalized across pairs
- Both per-exchange and aggregated
- Alert when spread > 1%
- Retain for 30 days
```

#### Step 2: Design Approval

```
你: Approve this design

Claude: 生成设计文档 .github/designs/prometheus-metrics.md

你: Looks good, create the implementation plan
```

#### Step 3: Implementation Plan

```
Claude 生成:

# Implementation Plan: Prometheus Metrics

## Task 1: Create Prometheus Registry (2-5 min)
File: internal/infrastructure/monitoring/prometheus.go
- Define Registry struct
- Implement metrics initialization
- Verify: unit tests pass

## Task 2: Define Metrics (2-5 min)
File: internal/domain/metrics/metrics.go
- Define MetricDefinition interface
- Implement per-exchange and aggregated metrics
- Verify: go test -run TestMetricsDefinition

## Task 3: Integrate with Monitor Service (2-5 min)
File: internal/application/usecase/monitor/service.go
- Add Prometheus registry to ServiceDeps
- Emit metrics on price updates
- Verify: metrics endpoint works

## Task 4: Configuration (2-5 min)
File: configs/config.toml
- Add prometheus section
- Configure metrics retention
- Verify: config loads correctly

你: Start implementation
```

#### Step 4: Subagent-Driven Execution

```
Claude: 创建 git worktree 并派遣 4 个子代理

Subagent 1 执行 Task 1:
- RED: 写失败的测试
- GREEN: 实现 Registry
- REFACTOR: 优化代码
- 子代理: "Task 1 complete, awaiting review"

你: ✅ Approve Task 1

Subagent 2-4: 并行执行其他任务

所有任务完成:
Claude: "All tasks complete, merging to main"
```

---

## 最佳实践

### ✅ 要做的

1. **明确说出你的需求**
   ```
   ❌ "Add metrics"
   ✅ "Add Prometheus metrics for spread monitoring with per-exchange tracking"
   ```

2. **批准设计再开始编码**
   ```
   你: Help me plan...
   Claude: [生成设计]
   你: Approve this
   [然后才开始编码]
   ```

3. **使用 Superpowers 触发词**
   ```
   ✅ "Help me plan"
   ✅ "Let's debug"
   ✅ "Write tests for"
   ❌ "Just implement it"
   ```

4. **定期审查代码**
   ```
   你: Review this PR against the plan and standards
   ```

### ❌ 不要做的

1. **跳过设计阶段直接编码**
   - 会导致重复工作和架构问题

2. **忽视测试要求**
   - Superpowers 强制 TDD，这是特性

3. **混淆多个任务**
   - 每个任务应该是原子的，2-5 分钟可完成

4. **不批准审查结果**
   - CRITICAL 问题会阻止进度

---

## 故障排查

### Claude 没有自动激活技能？

检查：
1. 装了 Superpowers 插件吗？
2. 用的是 Claude Code 吗？
3. 用了正确的触发词吗？

### 子代理执行失败？

检查：
1. 计划任务清晰吗？
2. 文件路径正确吗？
3. 有无效的验证步骤吗？

### 代码审查被挡住？

查看 CRITICAL 问题：
- 未遵循 Go 规范
- 缺少测试
- 不符合计划
- 安全问题

---

## 项目特定规则

### 必须遵循

- ✅ 所有新代码通过 Container 依赖注入
- ✅ 所有代码都有测试（80%+ 覆盖）
- ✅ 遵循 [GO_CONVENTIONS.md](.github/skills/GO_CONVENTIONS.md)
- ✅ 遵循 [BOT_CONVENTIONS.md](.github/skills/BOT_CONVENTIONS.md)
- ✅ Redis/SQLite 通过 Container 实例化

### 验证步骤模板

```bash
# 单元测试
go test -v -cover ./[package]

# 所有测试
go test -cover ./...

# 代码风格
gofmt -s -w .
golangci-lint run

# 构建验证
go build ./cmd/xarb
```

---

## 支持和更新

- 获取最新 Superpowers：`/plugin update superpowers`
- 技能库：https://github.com/obra/superpowers
- 问题报告：https://github.com/obra/superpowers/issues

---

## 下一步

1. 安装 Superpowers（如果还没有）
2. 阅读 [DEVELOPMENT.md](.github/DEVELOPMENT.md)
3. 尝试第一个特性规划：`Help me plan: [feature]`
4. 享受 Superpowers 驱动的开发！

**Happy coding! 🚀**
