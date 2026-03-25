import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { ScoreFilter } from './ScoreFilter.jsx';

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
    <div class="glass-panel rounded-lg p-4">
      {/* Search input + button */}
      <form onSubmit={handleSubmit} class="flex gap-2 mb-3">
        <input
          type="text"
          class="terminal-input flex-1"
          placeholder='$ grep -r "..."'
          value={localQuery}
          onInput={(e) => setLocalQuery(e.target.value)}
        />
        <button type="submit" class="terminal-btn px-4 text-sm whitespace-nowrap">
          {'>'} grep
        </button>
        <button type="button" onClick={handleClear} class="terminal-btn px-3 text-sm border-terminal-gray text-terminal-gray whitespace-nowrap">
          × clear
        </button>
      </form>

      {/* Search type radios */}
      <div class="flex items-center gap-4 mb-3 text-sm">
        <span class="text-terminal-gray">Type:</span>
        {['keyword', 'semantic', 'hybrid'].map(type => (
          <label key={type} class="flex items-center gap-1.5 cursor-pointer text-terminal-green hover:text-white transition-colors">
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
      <div class="flex flex-wrap items-center gap-3 text-sm">
        {/* Category */}
        <select
          class="terminal-input text-sm py-1"
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
          class="terminal-input text-sm py-1"
          value={sort}
          onChange={(e) => onFilterChange({ sort: e.target.value })}
        >
          <option value="weight">Weight</option>
          <option value="latest">Latest</option>
          <option value="visits">Visits</option>
        </select>

        {/* Size */}
        <select
          class="terminal-input text-sm py-1"
          value={size}
          onChange={(e) => onFilterChange({ size: Number(e.target.value) })}
        >
          <option value="20">20</option>
          <option value="50">50</option>
          <option value="100">100</option>
        </select>

        {/* Short URL filter */}
        <label class="flex items-center gap-1.5 cursor-pointer text-terminal-gray hover:text-terminal-green transition-colors">
          <input
            type="checkbox"
            checked={isShortURL}
            onChange={(e) => onFilterChange({ isShortURL: e.target.checked })}
            class="accent-green-500"
          />
          Short URLs only
        </label>
      </div>

      {/* Score filter for hybrid */}
      {searchType === 'hybrid' && (
        <div class="mt-3">
          <ScoreFilter value={minScore} onChange={(v) => onFilterChange({ minScore: v })} />
        </div>
      )}
    </div>
  );
}
