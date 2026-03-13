# SDD - SQL 语义安全校验工具

## 1. 一句话摘要

在编译/CI 阶段静态校验代码中嵌入的 SQL（字符串与 ORM 调用），结合目标数据库 schema 做语义检查（字段/表存在性、可用索引、潜在全表扫描、危险语句等）。

## 2. 目标与非目标

### 目标

- 发现 SQL 语义错误（表/字段不存在、索引未命中、潜在全表扫描等）。
- 非侵入式：零改动源码调用。
- 支持 CI：JSON 可解析输出 + 终端 summary。
- 规则插件化：可注册/启用/禁用，支持自定义规则。
- 精确定位：文件路径、行列、原始 SQL 片段、修复建议。
- 并行分析，性能可接受。

### 非目标

- 运行时 SQL 执行与拦截。
- SQL 优化器/执行计划级别预测。
- 覆盖所有 SQL 方言的完整语法。

## 3. 设计原则

- 小而可测、可扩展、非侵入。
- Schema 驱动，优先从 DB/DDL 获取真实结构。
- 接口优先、可替换实现，避免全局可变状态。
- 可审计与安全：不上传源码，远程访问可配置。

## 4. 需求清单（MVP）

- AST 扫描 Go 源码中的 SQL 字符串。
- 支持 `SELECT/UPDATE/DELETE` 基础语义解析。
- Schema 加载（MySQL 或 DDL 文件）。
- 字段/表存在性校验。
- CLI `scan` 输出 JSON + summary。

## 5. 架构分层

- `cli/`：命令行入口、参数解析、配置加载。
- `core/analyzer/`：协调扫描、并行处理、诊断汇总。
- `parser/`：SQL 解析器接口与实现。
- `schema/`：Schema 加载器与缓存。
- `rules/`：规则注册/启用/禁用与规则执行。
- `output/`：JSON/终端输出。
- `report/`：诊断数据结构。

## 6. 关键数据结构

### 诊断（Diagnostic）

- `File`：文件路径
- `Line`/`Column`
- `Snippet`：原始 SQL 片段
- `Rule`：规则 ID
- `Message`：诊断内容
- `Suggestion`：修复建议
- `Severity`：`info|warn|error`
- `Code`：可机器解析的错误码

## 7. 关键接口

### Analyzer

- `Analyze(ctx, targets, cfg) ([]Diagnostic, error)`

### Parser

- `Parse(sql string) (SQLIR, error)`

### SchemaLoader

- `Load(ctx, cfg) (Schema, error)`

### Rule

- `ID() string`
- `Apply(ctx, ir, schema) ([]Diagnostic, error)`

## 8. 扩展性与插件

- 规则通过注册表加载；MVP 先支持内置规则。
- 后续支持 Go plugin 或动态配置加载。

## 9. 诊断输出

- JSON 结构稳定，支持机器解析。
- summary 输出统计与 Top 规则。

## 10. 并发策略

- worker 池扫描 Go 文件。
- 可配置并发数（默认 `GOMAXPROCS`）。

## 11. 测试策略

- 单测：AST 识别、SQL 解析、规则触发、Schema 加载。
- 集成测试：sqlite 或 dockertest（MySQL/Postgres）。
- Benchmark：解析器与分析器核心路径。

## 12. 里程碑

- M1: MVP 可扫描并产出 JSON 诊断
- M2: GORM 识别 + 索引提示
- M3: 插件化 + CI 集成增强

## 13. 阶段 2 高标准规范与要求

### 总体维度

- 技术深度：语义识别、优化器规则、数据库特性理解。
- 工程鲁棒性：缓存、容错、权限与性能保障。
- 易用性：可读诊断、可执行建议、配置开关友好。

### 13.1 语义识别层：GORM Pattern Recognition

要求：必须处理“非连续链式调用”。

规范：

- AST 深度追踪：不能仅识别简单的 `db.Where().Find()`。必须通过 `go/ast` 或 `go/types` 追踪变量赋值。例如，当 `query := db.Model(&User{})` 在函数开头，而 `query.Where(...)` 在条件分支中时，解析器需能还原完整的查询语义树。
- 方法覆盖率：首批必须完整覆盖 `Where`, `Select`, `Joins`, `Group`, `Having`, `Order`, `Limit`, `Offset`。
- 常量推导：识别 `Where("age > ?", 18)` 中的字段名。对于动态拼接的字符串字段（虽然不推荐，但常见），需提供“无法解析”的警告而非崩溃。

### 13.2 元数据层：Postgres Schema Live Fetch

要求：建立“轻量化、版本化”的数据库视图快照。

规范：

- 非侵入式查询：仅使用只读权限查询 `pg_catalog` 和 `information_schema`。禁止对生产库产生锁竞争。
- Schema 缓存版本化：缓存需包含 `MD5(schema_struct)` 或 `last_updated`。当数据库结构变更时，支持手动或根据 TTL 自动触发 Refetch。
- 多租户支持：在获取 Schema 时，必须明确 SearchPath（Schema 隔离），避免混淆不同 Schema 下的同名表。

### 13.3 核心算法层：MissingIndexRule

要求：模拟数据库优化器的索引匹配逻辑。

规范：

- 左前缀匹配准则：针对复合索引，规则必须能够判断查询条件是否满足“最左匹配原则”。如果 WHERE 只有 age 而索引是 (name, age)，应提示索引失效风险。
- 算子敏感性：识别 `LIKE '%abc'`、`NOT IN`、`!=` 等导致索引失效的算子，并标记为 High Risk。
- 冗余索引检测：如果建议增加索引 (a, b)，但表中已有 (a, b, c)，算法应能识别并抑制建议，防止索引膨胀。

### 13.4 诊断输出层：Code & Suggestion

要求：产出必须具备“机器可读”与“人类可感知”的双重属性。

规范：

- 错误码标准化：参考 golangci-lint，每个诊断结果必须有固定 ID（如 `GORM1001: MissingIndex`）。
- 修复建议（Quick Fix）：Suggestion 不仅要说“缺索引”，还要给出 SQL 语句：`CREATE INDEX idx_user_age ON users(age);`。
- 置信度分级：输出需带上 Confidence 字段（High/Medium/Low）。例如，解析不出的动态 SQL 标记为 Low，明确的 Full Table Scan 标记为 High。

### 13.5 性能与开关规范

要求：严禁阻塞 CI/CD 流程或本地 IDE 响应。

规范：

- 扫描耗时限制：单个 Package 的 AST 解析加规则检查，耗时应控制在 500ms 以内。
- 开关粒度：支持配置文件（如 `.gormlint.yaml`）和代码内注释（`//gormlint:ignore`）双重粒度的控制。

### 13.6 进阶风险提示

- 软删除：GORM 默认带 `deleted_at`。MissingIndexRule 必须自动将 `deleted_at` 纳入匹配逻辑，否则索引建议可能失效。
- JSONB 与 GIN 索引：识别 JSONB 字段查询并建议 GIN 索引。
