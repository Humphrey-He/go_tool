# SQL 语义安全校验工具（SQL Semantic Lint）

一个面向 Go 项目的编译期 SQL 语义检查工具：在编译/CI 阶段静态校验代码中嵌入的 SQL（字符串与 ORM 调用），结合目标数据库 schema 做语义检查（字段/表存在性、可用索引、潜在全表扫描、危险语句等）。

## 项目说明

### 目标

- 在编译期发现 SQL 语义问题，而不是在运行期或线上发现。
- 支持常见 SQL 语句和 ORM 语句的字段/索引校验。
- 集成 `go vet` 与 `golangci-lint`，作为开发者工具链的一部分。

### 核心思路

- 使用 Go AST 解析器扫描源码，提取 SQL 字符串或 ORM 方法链。
- 解析 SQL 抽象语法树，抽取表名、字段、条件、排序等语义要素。
- 根据 Schema 元信息进行校验：字段是否存在、索引是否命中、潜在慢查询风险提示。

### 价值

- 在提交代码前减少逻辑性 SQL 错误。
- 降低漏写索引导致的性能问题。
- 形成可复用的工程化能力，为后续自动化优化提供基础。

## 功能清单

- 代码扫描
- SQL 字符串识别（静态/格式化字符串/占位符拼接）
- ORM 调用识别（GORM、sqlx、database/sql）
- SQL 语义解析（`SELECT/UPDATE/DELETE/INSERT`）
- Schema 驱动校验（表/字段/主键/唯一索引/普通索引/外键/列类型）
- 索引命中检查（基于 `WHERE` 条件的索引覆盖建议）
- 性能风险提示（潜在全表扫描、危险语句、复杂 `LIKE`）
- 规则引擎（规则注册/启用/禁用，自定义规则插件）
- 输出与集成（JSON 诊断 + SARIF + 终端 summary，CI 支持）
- CLI（`init`/`scan`/`serve` 可选，支持 mono-repo）

## 设计原则

- 小而可测、可扩展、非侵入（零改动源码调用）
- 精确定位（文件:行:列 + 原始 SQL 片段 + 修复建议）
- 规则插件化（Go plugin 或动态配置）
- 多输入源支持（字符串/格式化/拼接/ORM）
- Schema 驱动（可从 live DB 或 DDL 加载）
- 并行分析（可配置 worker 池）
- CI 友好（机器可读输出与稳定退出码）
- 安全与隐私（不上传源码，远程拉取 schema 可审计）

## SDD 文档

- 详见 [docs/SDD.md](E:\awesomeProject\go_tool\docs\SDD.md)

## 阶段 2 规范与要求（摘要）

- GORM 语义识别必须支持非连续链式调用与 AST 深度追踪
- 覆盖 Where/Select/Joins/Group/Having/Order/Limit/Offset
- Postgres schema live fetch 仅查询 `pg_catalog`/`information_schema`，含 version/TTL/cache 与 search_path
- MissingIndexRule 遵循最左前缀、算子敏感、冗余索引检测
- 诊断输出必须有稳定 ID、Quick Fix SQL、置信度分级
- 性能限制：单包 ≤ 500ms；支持 `.gormlint.yaml` + `//gormlint:ignore`
- 软删除 `deleted_at` 与 JSONB/GIN 索引建议必须纳入规则

## 项目路线图

### Phase 1：MVP（可用最小版本）

- AST 扫描 SQL 字符串
- 支持 `SELECT/UPDATE/DELETE` 语义解析
- Schema 加载（支持 MySQL 或 DDL）
- 字段/表存在性校验
- CLI 输出诊断信息（JSON + summary）

### Phase 2：GORM 支持与索引提示

- 解析 GORM 链式调用
- `WHERE` 条件索引匹配校验
- 输出索引缺失或未命中的提示
- 支持复合索引检测

### Phase 3：工程化集成

- 规则插件化
- CI 运行支持与基线忽略
- JSON/SARIF 输出
- 更多数据库方言（MySQL/Postgres/DDL）

### Phase 4：扩展与优化

- 支持更多数据库（PostgreSQL、SQLite）
- 支持更复杂 SQL 语法（子查询、CTE、窗口函数）
- 输出优化建议（索引推荐、语句重写建议）

## 快速开始

```
# 初始化配置
sqlsafelint init

# 运行扫描
sqlsafelint scan

# 输出 SARIF
sqlsafelint scan --sarif
```

## 配置文件示例（.sqlsafelint.json）

```
{
  "schema": {
    "driver": "ddl",
    "dsn": "",
    "ddl": "schema.sql",
    "search_path": ["public"],
    "cache_ttl": 300,
    "cache_path": ".sqlsafelint.schema.json"
  },
  "scan": {
    "workspace": ".",
    "include": ["**/*.go"],
    "exclude": ["**/vendor/**", "**/.git/**"],
    "workers": 0,
    "timeout_ms": 500
  },
  "rules": {
    "enable": ["GORM1001", "GORM1002"],
    "disable": [],
    "plugins": []
  },
  "output": {
    "json": true,
    "sarif": false
  }
}
```

## 目录结构（当前）

```
cmd/sqlsafelint
internal/cli
internal/config
internal/core/analyzer
internal/output
internal/parser
internal/report
internal/rules
internal/schema
docs/SDD.md
scripts/ci.sh
.github/workflows/ci.yml
```

## 许可证

MIT
