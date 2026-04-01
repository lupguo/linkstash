# Network Type Feature Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a configurable `network_type` field (internal/domestic/overseas/unknown) to URLs, enabling filtering by network environment across web UI, API, and CLI, with automatic LLM classification on analysis.

**Architecture:** New `network_type` string field on URL entity, configured via YAML (like categories), exposed via public config API endpoint. Backend filters at DB query level; frontend adds chip selector in filter panel + dropdown in detail page. LLM prompt extended to classify network type. CLI display updated with bug fix for ID field.

**Tech Stack:** Go (GORM, chi), Preact (JSX, signals), YAML config, esbuild

---

### Task 1: Config Layer — Add NetworkTypes to Config

**Files:**
- Modify: `app/infra/config/config.go:12-23` (Config struct) and `:200-280` (Load defaults)
- Modify: `conf/app_dev.yaml:67-76` (add network_types section)
- Modify: `conf/app_example.yaml:67-76` (add network_types section)

- [ ] **Step 1: Add NetworkTypeOption struct and field to Config**

In `app/infra/config/config.go`, add the struct before `Config` and add the field:

```go
// NetworkTypeOption represents a network access type for the UI.
type NetworkTypeOption struct {
	Key   string `yaml:"key"`
	Label string `yaml:"label"`
}
```

Add to `Config` struct after `Categories`:
```go
NetworkTypes []NetworkTypeOption `yaml:"network_types"`
```

- [ ] **Step 2: Add defaults in Load() function**

In `app/infra/config/config.go`, inside `Load()`, after the Short TTL defaults block (after line 279), add:

```go
// Default network types if not configured
if len(cfg.NetworkTypes) == 0 {
	cfg.NetworkTypes = []NetworkTypeOption{
		{Key: "internal", Label: "内网"},
		{Key: "domestic", Label: "国内"},
		{Key: "overseas", Label: "海外"},
		{Key: "unknown", Label: "未知"},
	}
}
```

- [ ] **Step 3: Add network_types to both YAML config files**

In `conf/app_dev.yaml`, after the `categories` section (after line 76), add:

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

In `conf/app_example.yaml`, add the same block after the `categories` section.

- [ ] **Step 4: Verify compilation**

Run: `cd /private/data/projects/github.com/lupguo/linkstash && go build ./...`
Expected: Build succeeds with no errors.

- [ ] **Step 5: Commit**

```bash
git add app/infra/config/config.go conf/app_dev.yaml conf/app_example.yaml
git commit -m "feat: add network_types config (internal/domestic/overseas/unknown)"
```

---

### Task 2: Entity + DB Schema — Add NetworkType Field

**Files:**
- Modify: `app/domain/entity/url.go:9-30` (URL struct)

- [ ] **Step 1: Add NetworkType field to URL entity**

In `app/domain/entity/url.go`, add after the `Tags` field (line 16):

```go
NetworkType    string         `gorm:"size:20;default:unknown;index;comment:网络类型(internal/domestic/overseas/unknown)" json:"network_type"`
```

- [ ] **Step 2: Verify compilation and auto-migration**

Run: `go build ./...`
Expected: Build succeeds. GORM AutoMigrate will add the `network_type` column with default `unknown` on next server start.

- [ ] **Step 3: Commit**

```bash
git add app/domain/entity/url.go
git commit -m "feat: add network_type field to URL entity"
```

---

### Task 3: Backend Filtering — Add networkType to List Pipeline

**Files:**
- Modify: `app/domain/repos/url_repo.go:13` (List signature)
- Modify: `app/infra/db/url_repo_impl.go:74` (List implementation)
- Modify: `app/domain/services/url_service.go:77` (ListURLs)
- Modify: `app/application/url_usecase.go:41` (ListURLs)
- Modify: `app/handler/url_handler.go:98-132` (HandleList)

- [ ] **Step 1: Update URLRepo interface**

In `app/domain/repos/url_repo.go`, change the `List` signature on line 13 from:

```go
List(page int, size int, sort string, category string, tags string, isShortURL bool) ([]*entity.URL, int64, error)
```

to:

```go
List(page int, size int, sort string, category string, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error)
```

- [ ] **Step 2: Update URLRepoImpl.List**

In `app/infra/db/url_repo_impl.go`, change the `List` method signature on line 74 from:

```go
func (r *URLRepoImpl) List(page, size int, sort, category, tags string, isShortURL bool) ([]*entity.URL, int64, error) {
```

to:

```go
func (r *URLRepoImpl) List(page, size int, sort, category, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error) {
```

Add the networkType filter after the `isShortURL` filter (after line 89):

```go
if networkType != "" {
	query = query.Where("network_type = ?", networkType)
}
```

- [ ] **Step 3: Update URLService.ListURLs**

In `app/domain/services/url_service.go`, change line 77 from:

```go
func (s *URLService) ListURLs(page, size int, sort, category, tags string, isShortURL bool) ([]*entity.URL, int64, error) {
	return s.urlRepo.List(page, size, sort, category, tags, isShortURL)
}
```

to:

```go
func (s *URLService) ListURLs(page, size int, sort, category, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error) {
	return s.urlRepo.List(page, size, sort, category, tags, isShortURL, networkType)
}
```

- [ ] **Step 4: Update URLUsecase.ListURLs**

In `app/application/url_usecase.go`, change line 41 from:

```go
func (uc *URLUsecase) ListURLs(page, size int, sort, category, tags string, isShortURL bool) ([]*entity.URL, int64, error) {
	return uc.urlService.ListURLs(page, size, sort, category, tags, isShortURL)
}
```

to:

```go
func (uc *URLUsecase) ListURLs(page, size int, sort, category, tags string, isShortURL bool, networkType string) ([]*entity.URL, int64, error) {
	return uc.urlService.ListURLs(page, size, sort, category, tags, isShortURL, networkType)
}
```

- [ ] **Step 5: Update HandleList to parse network_type param**

In `app/handler/url_handler.go`, in `HandleList` (around line 116-120), add after `isShortURL`:

```go
networkType := q.Get("network_type")
```

And change the `ListURLs` call from:

```go
urls, total, err := h.usecase.ListURLs(page, size, sort, category, tags, isShortURL)
```

to:

```go
urls, total, err := h.usecase.ListURLs(page, size, sort, category, tags, isShortURL, networkType)
```

- [ ] **Step 6: Update HandleUpdate to accept network_type**

In `app/handler/url_handler.go`, inside `HandleUpdate`, add after the `status` update block (after line 214):

```go
if v, ok := updates["network_type"]; ok {
	existing.NetworkType = v.(string)
}
```

- [ ] **Step 7: Update HandleReanalyze to clear NetworkType**

In `app/handler/url_handler.go`, inside `HandleReanalyze`, add after line 292 (`existing.Tags = ""`):

```go
existing.NetworkType = "unknown"
```

- [ ] **Step 8: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

- [ ] **Step 9: Run tests**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 10: Commit**

```bash
git add app/domain/repos/url_repo.go app/infra/db/url_repo_impl.go app/domain/services/url_service.go app/application/url_usecase.go app/handler/url_handler.go
git commit -m "feat: add network_type backend filtering through List pipeline"
```

---

### Task 4: LLM Analysis — Extend Prompt and Parser

**Files:**
- Modify: `conf/app_dev.yaml:57-62` (url_analysis prompt)
- Modify: `conf/app_example.yaml:57-62` (url_analysis prompt)
- Modify: `app/domain/services/worker_service.go:161-177` (parsed struct + update)

- [ ] **Step 1: Update url_analysis prompt in config files**

In `conf/app_dev.yaml`, replace the `url_analysis` prompt (lines 57-62) with:

```yaml
    url_analysis: |
      分析以下网页内容，返回JSON格式：
      {"title":"标题","keywords":"关键词1,关键词2","description":"50字内摘要","category":"分类","tags":"标签1,标签2","network_type":"网络类型"}
      category必须从以下选项中选择：技术、设计、产品、商业、科学、生活、工具、资讯、其他
      network_type根据URL域名和页面内容判断网络访问环境，从以下选项中选择：internal(企业内部/内网站点,如内网IP、企业私有域名), domestic(中国国内站点), overseas(海外站点), unknown(无法确定)
      tags基于内容自由生成，用逗号分隔，2-5个标签。
      仅返回JSON，不要其他内容。
```

Apply the same change to `conf/app_example.yaml`.

- [ ] **Step 2: Update parsed struct in worker_service.go**

In `app/domain/services/worker_service.go`, change the parsed struct (lines 161-167) from:

```go
var parsed struct {
	Title       string `json:"title"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Tags        string `json:"tags"`
}
```

to:

```go
var parsed struct {
	Title       string `json:"title"`
	Keywords    string `json:"keywords"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Tags        string `json:"tags"`
	NetworkType string `json:"network_type"`
}
```

- [ ] **Step 3: Apply NetworkType from parsed response**

In `app/domain/services/worker_service.go`, after line 177 (`url.Tags = parsed.Tags`), add:

```go
if parsed.NetworkType != "" {
	url.NetworkType = parsed.NetworkType
} else {
	url.NetworkType = "unknown"
}
```

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add conf/app_dev.yaml conf/app_example.yaml app/domain/services/worker_service.go
git commit -m "feat: extend LLM prompt to classify network_type"
```

---

### Task 5: API Config Endpoint — Expose network-types

**Files:**
- Modify: `cmd/server/main.go:80-83` (add new config endpoint)

- [ ] **Step 1: Add /api/config/network-types endpoint**

In `cmd/server/main.go`, after the categories endpoint (after line 83), add:

```go
r.Get("/api/config/network-types", func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"network_types": app.Config.NetworkTypes})
})
```

- [ ] **Step 2: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "feat: add /api/config/network-types public endpoint"
```

---

### Task 6: Frontend — API Client + IndexPage + SearchBar

**Files:**
- Modify: `web/src/js/api.js:53-61` (configApi)
- Modify: `web/src/js/pages/IndexPage.jsx`
- Modify: `web/src/js/components/SearchBar.jsx`

- [ ] **Step 1: Add networkTypes to configApi**

In `web/src/js/api.js`, add a new method to `configApi` after `categories()`:

```js
export const configApi = {
  /**
   * Get configured categories.
   * @returns {Promise<{categories: string[]}>}
   */
  categories() {
    return fetch('/api/config/categories').then(res => res.json());
  },

  /**
   * Get configured network types.
   * @returns {Promise<{network_types: {key: string, label: string}[]}>}
   */
  networkTypes() {
    return fetch('/api/config/network-types').then(res => res.json());
  },
};
```

- [ ] **Step 2: Add networkType state and data loading to IndexPage**

In `web/src/js/pages/IndexPage.jsx`:

Add state after `categories` state (line 21):
```js
const [networkTypes, setNetworkTypes] = useState([]);
const [networkType, setNetworkType] = useState('');
```

Add data loading after the categories useEffect (after line 41):
```js
// Fetch network types from config API
useEffect(() => {
  if (!isAuthenticated.value) return;
  configApi.networkTypes().then(data => {
    setNetworkTypes(data.network_types || []);
  }).catch(err => {
    console.error('Failed to load network types:', err);
  });
}, []);
```

In `fetchData`, for the non-search branch (line 80), add `network_type` param:
```js
result = await urlApi.list({
  page: currentPage,
  size,
  sort,
  category: category || undefined,
  network_type: networkType || undefined,
  is_shorturl: isShortURL ? 1 : undefined,
});
```

For the search branch (after the category filter around line 64-66), add network_type client-side filter:
```js
if (networkType) {
  items = items.filter(u => u.network_type === networkType);
}
```

Add `networkType` to the `fetchData` dependency array (line 97):
```js
}, [query, searchType, category, sort, size, minScore, isShortURL, networkType]);
```

Add `networkType` to the effect dependency on line 104:
```js
}, [isAuthenticated.value, query, searchType, category, sort, size, minScore, isShortURL, networkType, urlListVersion.value]);
```

In `handleFilterChange`, add:
```js
if (filters.networkType !== undefined) setNetworkType(filters.networkType);
```

In the ESC key handler (line 136-143), add `setNetworkType('');` to the reset block.

Pass new props to `SearchBar`:
```jsx
<SearchBar
  query={query}
  searchType={searchType}
  category={category}
  networkType={networkType}
  sort={sort}
  size={size}
  isShortURL={isShortURL}
  minScore={minScore}
  categories={categories}
  networkTypes={networkTypes}
  onSearch={handleSearch}
  onFilterChange={handleFilterChange}
/>
```

- [ ] **Step 3: Add network type chips to SearchBar**

In `web/src/js/components/SearchBar.jsx`:

Update the function signature to include new props:
```js
export function SearchBar({ query, searchType, category, networkType, sort, size, isShortURL, minScore, categories, networkTypes, onSearch, onFilterChange }) {
```

Update `handleClear` to reset networkType:
```js
function handleClear() {
  setLocalQuery('');
  onSearch('', 'keyword');
  onFilterChange({
    category: '',
    networkType: '',
    sort: 'weight',
    size: 100,
    isShortURL: false,
    minScore: 0.6,
    searchType: 'keyword',
  });
}
```

Add `networkType !== ''` to the `activeFilterCount` array:
```js
const activeFilterCount = [
  searchType !== 'keyword',
  category !== '',
  networkType !== '',
  sort !== 'weight',
  size !== 100,
  isShortURL,
  searchType === 'hybrid' && minScore !== 0.6,
].filter(Boolean).length;
```

Add the Network chip section after the Category chips section (after the closing `</div>` of the Category `mb-3` block, around line 128):
```jsx
{/* Network type chips */}
<div class="mb-3">
  <span class="text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5 block">Network</span>
  <div class="flex flex-wrap gap-1.5">
    <button
      type="button"
      class={`filter-chip ${networkType === '' ? 'active' : ''}`}
      onClick={() => onFilterChange({ networkType: '' })}
    >
      全部
    </button>
    {(networkTypes || []).map(nt => (
      <button
        key={nt.key}
        type="button"
        class={`filter-chip ${networkType === nt.key ? 'active' : ''}`}
        onClick={() => onFilterChange({ networkType: nt.key })}
      >
        {nt.label}
      </button>
    ))}
  </div>
</div>
```

- [ ] **Step 4: Build frontend**

Run: `make frontend`
Expected: CSS + JS build succeeds.

- [ ] **Step 5: Commit**

```bash
git add web/src/js/api.js web/src/js/pages/IndexPage.jsx web/src/js/components/SearchBar.jsx
git commit -m "feat: add network type filter to frontend index page"
```

---

### Task 7: Frontend — DetailPage View + Edit

**Files:**
- Modify: `web/src/js/pages/DetailPage.jsx`

- [ ] **Step 1: Add network_type to EMPTY_FORM and state**

In `web/src/js/pages/DetailPage.jsx`, add to `EMPTY_FORM` (after `tags`):
```js
network_type: '',
```

- [ ] **Step 2: Load network types and populate form**

Add `networkTypes` state after `categories` state (around line 36):
```js
const [networkTypes, setNetworkTypes] = useState([]);
```

Add useEffect to load network types (after the categories useEffect):
```js
useEffect(() => {
  configApi.networkTypes().then(data => {
    setNetworkTypes(data.network_types || []);
  }).catch(err => {
    console.error('Failed to load network types:', err);
  });
}, []);
```

In the `loadUrl` function, add to the `setForm` call (after `tags`):
```js
network_type: data.network_type || '',
```

- [ ] **Step 3: Add network_type to update API calls**

In `handleSubmit`, for the update path (around line 144), add `network_type: form.network_type` to the update object:
```js
await urlApi.update(id, {
  link: form.link,
  title: form.title,
  description: form.description,
  keywords: form.keywords,
  category: form.category,
  tags: form.tags,
  network_type: form.network_type,
  manual_weight: Number(form.manual_weight) || 0,
  visit_count: Number(form.visit_count) || 0,
  color: form.color,
  icon: form.icon,
  favicon: form.favicon,
});
```

Also for the new URL update path (around line 117):
```js
await urlApi.update(newId, {
  title: form.title,
  description: form.description,
  keywords: form.keywords,
  category: form.category,
  tags: form.tags,
  network_type: form.network_type,
  manual_weight: Number(form.manual_weight) || 0,
  color: form.color,
  icon: form.icon,
  favicon: form.favicon,
});
```

- [ ] **Step 4: Add network_type dropdown to edit form**

In the edit form, change the Category + Tags grid (around line 285-302) to a 3-column grid that includes Network Type:

```jsx
{/* Category + Network Type + Tags */}
<div class="grid grid-cols-3 gap-4">
  <div>
    <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Category</label>
    <select class="input w-full text-sm" value={form.category} onChange={(e) => updateField('category', e.target.value)}>
      <option value="">Select...</option>
      {categories.map(cat => (
        <option key={cat} value={cat}>{cat}</option>
      ))}
      {form.category && !categories.includes(form.category) && (
        <option value={form.category}>{form.category}</option>
      )}
    </select>
  </div>
  <div>
    <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Network</label>
    <select class="input w-full text-sm" value={form.network_type} onChange={(e) => updateField('network_type', e.target.value)}>
      <option value="">Select...</option>
      {networkTypes.map(nt => (
        <option key={nt.key} value={nt.key}>{nt.label}</option>
      ))}
      {form.network_type && !networkTypes.find(nt => nt.key === form.network_type) && (
        <option value={form.network_type}>{form.network_type}</option>
      )}
    </select>
  </div>
  <div>
    <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Tags</label>
    <input type="text" class="input w-full text-sm" value={form.tags} onInput={(e) => updateField('tags', e.target.value)} placeholder="tag1, tag2" />
  </div>
</div>
```

- [ ] **Step 5: Add network_type to view mode**

In the view mode section, add after the Category + Tags `flex gap-6` block (after the closing `</div>` around line 435):

```jsx
{/* Network Type */}
{urlData.network_type && urlData.network_type !== '' && (
  <div>
    <label class="text-text-muted text-xs font-medium uppercase tracking-wider">Network</label>
    <p class="text-sm text-accent mt-1">
      {(() => {
        const nt = networkTypes.find(n => n.key === urlData.network_type);
        return nt ? nt.label : urlData.network_type;
      })()}
    </p>
  </div>
)}
```

- [ ] **Step 6: Build frontend**

Run: `make frontend`
Expected: Build succeeds.

- [ ] **Step 7: Commit**

```bash
git add web/src/js/pages/DetailPage.jsx
git commit -m "feat: add network_type view/edit to detail page"
```

---

### Task 8: CLI — Fix ID Bug + Add Network Display

**Files:**
- Modify: `cmd/cli/cmd/info.go`
- Modify: `cmd/cli/cmd/list.go`
- Modify: `cmd/cli/cmd/search.go`

- [ ] **Step 1: Fix info.go ID bug and add Network**

Replace the display block in `cmd/cli/cmd/info.go` (lines 45-53) with:

```go
fmt.Printf("ID:          %v\n", result["id"])
fmt.Printf("Link:        %v\n", result["link"])
fmt.Printf("Title:       %v\n", result["title"])
fmt.Printf("Description: %v\n", result["description"])
fmt.Printf("Category:    %v\n", result["category"])
fmt.Printf("Network:     %v\n", result["network_type"])
fmt.Printf("Tags:        %v\n", result["tags"])
fmt.Printf("Status:      %v\n", result["status"])
fmt.Printf("Visits:      %v\n", result["visit_count"])
fmt.Printf("Created:     %v\n", result["created_at"])
```

- [ ] **Step 2: Fix list.go ID bug and add Network column**

In `cmd/cli/cmd/list.go`, replace the display block (lines 47-58) with:

```go
fmt.Printf("%-6s %-50s %-10s %-12s\n", "ID", "Link", "Network", "Status")
fmt.Println(strings.Repeat("-", 80))
for _, item := range data {
	m, ok := item.(map[string]interface{})
	if !ok {
		continue
	}
	id := m["id"]
	link := m["link"]
	network := m["network_type"]
	status := m["status"]
	fmt.Printf("%-6v %-50v %-10v %-12v\n", id, link, network, status)
}
```

- [ ] **Step 3: Add Network to search.go results**

In `cmd/cli/cmd/search.go`, after `desc` (line 61), add network info:

```go
title := urlData["title"]
link := urlData["link"]
desc := urlData["description"]
network := urlData["network_type"]
fmt.Printf("%d. %v (score: %.2f) [%v]\n", i+1, title, score, network)
fmt.Printf("   %v\n", link)
if desc != nil && desc != "" {
	fmt.Printf("   %v\n", desc)
}
fmt.Println()
```

Replace lines 59-67 with the above block.

- [ ] **Step 4: Verify compilation**

Run: `go build ./...`
Expected: Build succeeds.

- [ ] **Step 5: Commit**

```bash
git add cmd/cli/cmd/info.go cmd/cli/cmd/list.go cmd/cli/cmd/search.go
git commit -m "feat: add network_type to CLI display + fix ID display bug"
```

---

### Task 9: Smoke Test + Merge to Main

- [ ] **Step 1: Full build**

Run: `make build`
Expected: Frontend + server + CLI all build successfully.

- [ ] **Step 2: Run tests**

Run: `make test`
Expected: All tests pass.

- [ ] **Step 3: Run smoke test**

Run: `make smoke-test`
Expected: Full build→start→test→stop cycle passes.

- [ ] **Step 4: Manual CLI verification**

Start server and test CLI:
```bash
make start
# Wait for server to start
./bin/linkstash-cli --server http://localhost:8888 --token <JWT> list
./bin/linkstash-cli --server http://localhost:8888 --token <JWT> info 1
make stop
```

Expected: CLI list shows Network column with values. CLI info shows Network line. ID displays correctly (not `<nil>`).

- [ ] **Step 5: Merge worktree to main**

Follow the worktree merge process — merge the feature branch into main.
