import { h } from 'preact';
import { useState, useEffect, useRef, useCallback } from 'preact/hooks';
import { route } from 'preact-router';
import { isAuthenticated } from '../store.js';
import { urlApi, searchApi } from '../api.js';
import { URLCard } from '../components/URLCard.jsx';
import { SearchBar } from '../components/SearchBar.jsx';

export function IndexPage() {
  const [urls, setUrls] = useState([]);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState('');
  const [searchType, setSearchType] = useState('keyword');
  const [category, setCategory] = useState('');
  const [sort, setSort] = useState('weight');
  const [size, setSize] = useState(20);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(true);
  const [minScore, setMinScore] = useState(0.6);
  const [isShortURL, setIsShortURL] = useState(false);
  const [categories, setCategories] = useState([]);

  const sentinelRef = useRef(null);
  const loadingRef = useRef(false);

  // Auth guard
  useEffect(() => {
    if (!isAuthenticated.value) {
      route('/login', true);
    }
  }, []);

  // Fetch data
  const fetchData = useCallback(async (currentPage, append = false) => {
    if (loadingRef.current) return;
    loadingRef.current = true;
    setLoading(true);

    try {
      let result;
      if (query) {
        result = await searchApi.search({
          q: query,
          type: searchType,
          page: currentPage,
          size,
          min_score: searchType === 'hybrid' ? minScore : undefined,
        });
        const items = result.results || [];
        setUrls(prev => append ? [...prev, ...items] : items);
        setHasMore(items.length === size);
      } else {
        result = await urlApi.list({
          page: currentPage,
          size,
          sort,
          category: category || undefined,
          is_shorturl: isShortURL ? 1 : undefined,
        });
        const items = result.urls || [];
        setUrls(prev => append ? [...prev, ...items] : items);
        setHasMore(items.length === size);

        // Extract categories from results for filter dropdown
        if (!append && items.length > 0) {
          const cats = [...new Set(items.map(u => u.Category).filter(Boolean))];
          setCategories(prev => {
            const merged = [...new Set([...prev, ...cats])];
            return merged.sort();
          });
        }
      }
    } catch (err) {
      console.error('Fetch error:', err);
    } finally {
      setLoading(false);
      loadingRef.current = false;
    }
  }, [query, searchType, category, sort, size, minScore, isShortURL]);

  // Load on mount and filter changes
  useEffect(() => {
    setPage(1);
    fetchData(1, false);
  }, [query, searchType, category, sort, size, minScore, isShortURL]);

  // Load more on page change (page > 1)
  useEffect(() => {
    if (page > 1) {
      fetchData(page, true);
    }
  }, [page]);

  // IntersectionObserver for infinite scroll
  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (!sentinel) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingRef.current) {
          setPage(prev => prev + 1);
        }
      },
      { threshold: 0.1 }
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasMore]);

  // ESC key handler
  useEffect(() => {
    function handleKeyDown(e) {
      if (e.key === 'Escape') {
        setQuery('');
        setCategory('');
        setSort('weight');
        setSize(20);
        setIsShortURL(false);
        setMinScore(0.6);
        setSearchType('keyword');
        setPage(1);
      }
    }
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  function handleSearch(q, type) {
    setQuery(q);
    setSearchType(type);
  }

  function handleFilterChange(filters) {
    if (filters.category !== undefined) setCategory(filters.category);
    if (filters.sort !== undefined) setSort(filters.sort);
    if (filters.size !== undefined) setSize(filters.size);
    if (filters.isShortURL !== undefined) setIsShortURL(filters.isShortURL);
    if (filters.minScore !== undefined) setMinScore(filters.minScore);
    if (filters.searchType !== undefined) setSearchType(filters.searchType);
  }

  function handleDelete(id) {
    setUrls(prev => prev.filter(u => u.ID !== id));
  }

  if (!isAuthenticated.value) return null;

  return (
    <div>
      <SearchBar
        query={query}
        searchType={searchType}
        category={category}
        sort={sort}
        size={size}
        isShortURL={isShortURL}
        minScore={minScore}
        categories={categories}
        onSearch={handleSearch}
        onFilterChange={handleFilterChange}
      />

      <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-4 mt-6">
        {urls.map(url => (
          <URLCard key={url.ID} url={url} onDelete={handleDelete} />
        ))}
      </div>

      {loading && (
        <div class="text-center text-terminal-gray py-8">Loading...</div>
      )}

      {!loading && urls.length === 0 && (
        <div class="text-center text-terminal-gray py-16">
          <p class="text-lg mb-2">No URLs found</p>
          <p class="text-sm">Try adjusting your search or <a href="/urls/new" class="text-terminal-green hover:underline">add a new URL</a></p>
        </div>
      )}

      {/* Sentinel for infinite scroll */}
      {hasMore && <div ref={sentinelRef} class="h-4" />}
    </div>
  );
}
