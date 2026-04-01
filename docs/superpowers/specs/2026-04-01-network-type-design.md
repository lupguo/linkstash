# Network Type Feature Design

## Overview

Add a `network_type` field to URLs that classifies network access environment: internal (企业内网), domestic (国内), overseas (海外), or unknown (未知). This enables filtering URLs by network environment across web UI, API, and CLI.

## Enum Values

| DB Value   | Chinese Label | Description           |
|------------|---------------|-----------------------|
| `internal` | 内网          | Enterprise/intranet   |
| `domestic` | 国内          | China domestic sites  |
| `overseas` | 海外          | International sites   |
| `unknown`  | 未知          | Cannot determine      |

Storage: English enum in DB. Display: Chinese labels in UI/CLI.

## Data Layer

### Entity Change (`app/domain/entity/url.go`)

Add field to `URL` struct:

```go
NetworkType string `json:"network_type" gorm:"type:varchar(20);default:unknown;index"`
```

### Migration

GORM AutoMigrate handles the new column automatically. Historical data defaults to `unknown`.

## Configuration Layer

### Config File (`conf/app_dev.yaml`)

```yaml
network_types:
  - key: internal
    label: 内网
  - key: domestic
    label: 国内
  - key: overseas
    label: 海外
  - key: unknown
    label: 未知
```

### Config Struct (`app/infra/config/config.go`)

```go
type NetworkTypeOption struct {
    Key   string `yaml:"key"`
    Label string `yaml:"label"`
}
```

Add to `Config` struct:
```go
NetworkTypes []NetworkTypeOption `yaml:"network_types"`
```

Add defaults in `Load()` if `NetworkTypes` is empty:
```go
if len(cfg.NetworkTypes) == 0 {
    cfg.NetworkTypes = []NetworkTypeOption{
        {Key: "internal", Label: "内网"},
        {Key: "domestic", Label: "国内"},
        {Key: "overseas", Label: "海外"},
        {Key: "unknown", Label: "未知"},
    }
}
```

### API Endpoint (`cmd/server/main.go`)

New public endpoint (no auth required, same pattern as categories):
```
GET /api/config/network-types
→ {"network_types": [{"key":"internal","label":"内网"}, ...]}
```

## LLM Analysis

### Prompt Update (`conf/app_dev.yaml` → `llm.prompts.url_analysis`)

Extend the JSON output format to include `network_type`:

```
分析以下网页内容，返回JSON格式：
{"title":"标题","keywords":"关键词1,关键词2","description":"50字内摘要","category":"分类","tags":"标签1,标签2","network_type":"网络类型"}
...
network_type根据URL域名和页面内容判断网络访问环境，从以下选项中选择：internal(企业内部/内网站点,如内网IP、企业域名), domestic(中国国内站点), overseas(海外站点), unknown(无法确定)
```

### Worker Service Update (`app/domain/services/worker_service.go`)

Parse `network_type` from LLM JSON response and save to URL entity. If missing or invalid, default to `unknown`.

## Backend Filtering

### Repository (`app/infra/db/url_repo_impl.go`)

In `List()` method, add filter condition:
```go
if networkType != "" {
    query = query.Where("network_type = ?", networkType)
}
```

### URL Handler (`app/handler/url_handler.go`)

`HandleList`: Parse `network_type` query parameter and pass to repository.

`HandleCreate` / `HandleUpdate`: Accept `network_type` field in request body. On create, default to `unknown` if not provided.

### Search Handler

Search results include full URL data; no backend changes needed. Frontend applies network_type filter client-side (consistent with existing category filter pattern on search results).

## Frontend

### SearchBar Filter (`web/src/js/components/SearchBar.jsx`)

Add network type chip selector in the filter panel:
- Load options from `/api/config/network-types` on mount
- Default: "全部" (no filter applied)
- Chip-style selector matching existing category filter UI
- Include in active filter count badge

### IndexPage (`web/src/js/pages/IndexPage.jsx`)

- New `networkType` state variable
- Pass `network_type` to `urlApi.list()` query params
- For search results: client-side filter by `network_type`

### DetailPage (`web/src/js/pages/DetailPage.jsx`)

- **View mode**: Display "网络类型" label with Chinese label mapping
- **Edit mode**: Dropdown selector, options loaded from `/api/config/network-types`

### API Client (`web/src/js/api.js`)

No changes needed — existing `list()` passes through query params generically.

## CLI

### info.go (`cmd/cli/cmd/info.go`)

Add Network line:
```
Network:     domestic (国内)
```

### list.go (`cmd/cli/cmd/list.go`)

Add Network column to table output.

### search.go (`cmd/cli/cmd/search.go`)

Add network info to search results display.

## CLI Bug Fix: ID Display

**Root cause**: CLI uses uppercase JSON keys (`result["ID"]`, `result["CreatedAt"]`) but API returns lowercase snake_case (`id`, `created_at`).

**Fix**:
- `info.go`: `result["ID"]` → `result["id"]`, `result["CreatedAt"]` → `result["created_at"]`
- `list.go`: `m["ID"]` → `m["id"]`

## Initial Value Strategy

- **New URLs**: `network_type` defaults to `unknown`
- **LLM analysis**: Automatically updates `network_type` based on URL/content analysis
- **If LLM cannot determine**: Keeps `unknown`
- **User override**: Manual edit always available via Detail page
- **Historical data**: All existing URLs get `unknown` via column default
- **Re-analysis**: Users can trigger "Reanalyze" to have LLM classify existing URLs

## Testing

- `make test` — unit tests pass
- `make smoke-test` — full build→start→test→stop cycle
- Manual verification: CLI list/search/info show Network field correctly
- Manual verification: Web UI filter, detail view, and edit all work
