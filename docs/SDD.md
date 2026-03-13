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
