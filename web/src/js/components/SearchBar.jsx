import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';

export function SearchBar({ query, searchType, category, sort, size, isShortURL, minScore, categories, onSearch, onFilterChange }) {
  const [localQuery, setLocalQuery] = useState(query || '');
  const [filtersOpen, setFiltersOpen] = useState(false);

  // Sync local query when parent clears it (e.g. ESC key)
  useEffect(() => {
    if (query === '' && localQuery !== '') {
      setLocalQuery('');
    }
  }, [query]);

  function handleSubmit(e) {
    e.preventDefault();
    onSearch(localQuery, searchType);
  }

  function handleClear() {
    setLocalQuery('');
    onSearch('', 'keyword');
    onFilterChange({
      category: '',
      sort: 'weight',
      size: 100,
      isShortURL: false,
      minScore: 0.6,
      searchType: 'keyword',
    });
  }

  // Count active filters
  const activeFilterCount = [
    searchType !== 'keyword',
    category !== '',
    sort !== 'weight',
    size !== 100,
    isShortURL,
    searchType === 'hybrid' && minScore !== 0.6,
  ].filter(Boolean).length;

  return (
    <div>
      {/* Search row: input + Filters button */}
      <form onSubmit={handleSubmit} class="flex gap-2 items-center">
        <div class="relative flex-1">
          <input
            type="text"
            class="input w-full pl-9"
            placeholder="Search URLs..."
            value={localQuery}
            onInput={(e) => setLocalQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'ArrowDown') { e.preventDefault(); setFiltersOpen(true); }
              else if (e.key === 'ArrowUp') { e.preventDefault(); setFiltersOpen(false); }
            }}
          />
          <svg class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-text-muted" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8" />
            <path d="m21 21-4.3-4.3" />
          </svg>
        </div>
        <button type="submit" class="btn btn-primary px-4 py-2 text-sm whitespace-nowrap">
          Search
        </button>
        {localQuery && (
          <button type="button" onClick={handleClear} class="btn btn-danger px-3 py-2 text-sm whitespace-nowrap">
            Clear
          </button>
        )}
        <button
          type="button"
          onClick={() => setFiltersOpen(!filtersOpen)}
          class={`btn px-3 py-2 text-sm whitespace-nowrap relative ${filtersOpen ? 'border-accent text-accent' : ''}`}
        >
          Filters
          {activeFilterCount > 0 && (
            <span class="absolute -top-1.5 -right-1.5 w-4 h-4 rounded-full bg-accent text-bg-primary text-[10px] font-bold flex items-center justify-center">
              {activeFilterCount}
            </span>
          )}
        </button>
      </form>

      {/* Collapsible filter panel */}
      {filtersOpen && (
        <div class="filter-panel mt-2 p-4">
          {/* Search type chips */}
          <div class="mb-3">
            <span class="text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5 block">Type</span>
            <div class="flex gap-1.5">
              {['keyword', 'semantic', 'hybrid'].map(type => (
                <button
                  key={type}
                  type="button"
                  class={`filter-chip ${searchType === type ? 'active' : ''}`}
                  onClick={() => onFilterChange({ searchType: type })}
                >
                  {type}
                </button>
              ))}
            </div>
          </div>

          {/* Category chips */}
          <div class="mb-3">
            <span class="text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5 block">Category</span>
            <div class="flex flex-wrap gap-1.5">
              <button
                type="button"
                class={`filter-chip ${category === '' ? 'active' : ''}`}
                onClick={() => onFilterChange({ category: '' })}
              >
                All
              </button>
              {(categories || []).map(cat => (
                <button
                  key={cat}
                  type="button"
                  class={`filter-chip ${category === cat ? 'active' : ''}`}
                  onClick={() => onFilterChange({ category: cat })}
                >
                  {cat}
                </button>
              ))}
            </div>
          </div>

          {/* Sort + Size + Short URL + Min Score */}
          <div class="flex flex-wrap items-center gap-3">
            <div class="flex items-center gap-1.5">
              <span class="text-text-muted text-xs font-medium uppercase tracking-wider">Sort</span>
              <select
                class="input text-xs py-1 px-2"
                value={sort}
                onChange={(e) => onFilterChange({ sort: e.target.value })}
              >
                <option value="weight">Weight</option>
                <option value="latest">Latest</option>
              </select>
            </div>

            <div class="flex items-center gap-1.5">
              <span class="text-text-muted text-xs font-medium uppercase tracking-wider">Size</span>
              <select
                class="input text-xs py-1 px-2"
                value={size}
                onChange={(e) => onFilterChange({ size: Number(e.target.value) })}
              >
                <option value="20">20</option>
                <option value="50">50</option>
                <option value="100">100</option>
              </select>
            </div>

            <label class="flex items-center gap-1.5 cursor-pointer text-text-secondary hover:text-accent transition-colors text-xs">
              <input
                type="checkbox"
                checked={isShortURL}
                onChange={(e) => onFilterChange({ isShortURL: e.target.checked })}
                class="accent-sky-400"
              />
              Short URLs only
            </label>

            {searchType === 'hybrid' && (
              <div class="flex items-center gap-1.5">
                <span class="text-text-muted text-xs font-medium uppercase tracking-wider">Min Score</span>
                <select
                  class="input text-xs py-1 px-2"
                  value={minScore}
                  onChange={(e) => onFilterChange({ minScore: parseFloat(e.target.value) })}
                >
                  <option value="0.5">0.5</option>
                  <option value="0.6">0.6</option>
                  <option value="0.7">0.7</option>
                  <option value="0.8">0.8</option>
                  <option value="0.9">0.9</option>
                </select>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
