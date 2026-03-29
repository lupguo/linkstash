import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';

export function SearchBar({ query, searchType, category, sort, size, isShortURL, minScore, categories, onSearch, onFilterChange }) {
  const [localQuery, setLocalQuery] = useState(query || '');

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
      size: 20,
      isShortURL: false,
      minScore: 0.6,
      searchType: 'keyword',
    });
  }

  return (
    <div class="glass-panel p-5">
      {/* Search input + button */}
      <form onSubmit={handleSubmit} class="flex gap-2.5 mb-4">
        <input
          type="text"
          class="terminal-input flex-1"
          placeholder="Search URLs..."
          value={localQuery}
          onInput={(e) => setLocalQuery(e.target.value)}
        />
        <button type="submit" class="terminal-btn px-5 py-2 whitespace-nowrap">
          Search
        </button>
        {localQuery && (
          <button type="button" onClick={handleClear} class="terminal-btn terminal-btn-danger px-4 py-2 whitespace-nowrap">
            Clear
          </button>
        )}
      </form>

      {/* Search type radios */}
      <div class="flex items-center gap-5 mb-4 text-[12px]">
        <span class="text-gray-600">Type</span>
        {['keyword', 'semantic', 'hybrid'].map(type => (
          <label key={type} class="flex items-center gap-1.5 cursor-pointer text-gray-500 hover:text-terminal-green transition-colors">
            <input
              type="radio"
              name="searchType"
              value={type}
              checked={searchType === type}
              onChange={() => onFilterChange({ searchType: type })}
              class="accent-green-500"
            />
            {type}
          </label>
        ))}
      </div>

      {/* Filter row */}
      <div class="flex flex-wrap items-center gap-3 text-[12px]">
        {/* Category */}
        <select
          class="terminal-input text-[12px] py-1.5 px-3"
          value={category}
          onChange={(e) => onFilterChange({ category: e.target.value })}
        >
          <option value="">All Categories</option>
          {(categories || []).map(cat => (
            <option key={cat} value={cat}>{cat}</option>
          ))}
        </select>

        {/* Sort */}
        <select
          class="terminal-input text-[12px] py-1.5 px-3"
          value={sort}
          onChange={(e) => onFilterChange({ sort: e.target.value })}
        >
          <option value="weight">Weight</option>
          <option value="latest">Latest</option>
        </select>

        {/* Size */}
        <select
          class="terminal-input text-[12px] py-1.5 px-3"
          value={size}
          onChange={(e) => onFilterChange({ size: Number(e.target.value) })}
        >
          <option value="20">20</option>
          <option value="50">50</option>
          <option value="100">100</option>
        </select>

        {/* Short URL filter */}
        <label class="flex items-center gap-1.5 cursor-pointer text-gray-500 hover:text-terminal-green transition-colors ml-1">
          <input
            type="checkbox"
            checked={isShortURL}
            onChange={(e) => onFilterChange({ isShortURL: e.target.checked })}
            class="accent-green-500"
          />
          Short URLs only
        </label>

        {/* Min Score dropdown for hybrid search */}
        {searchType === 'hybrid' && (
          <label class="flex items-center gap-1.5 text-gray-500 ml-1">
            <span>Min Score</span>
            <select
              class="terminal-input text-[12px] py-1.5 px-3"
              value={minScore}
              onChange={(e) => onFilterChange({ minScore: parseFloat(e.target.value) })}
            >
              <option value="0.5">0.5</option>
              <option value="0.6">0.6</option>
              <option value="0.7">0.7</option>
              <option value="0.8">0.8</option>
              <option value="0.9">0.9</option>
            </select>
          </label>
        )}
      </div>
    </div>
  );
}
