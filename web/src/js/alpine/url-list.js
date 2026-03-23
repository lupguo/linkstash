import { getPageData, copyToClipboard } from '../utils.js';

/**
 * Alpine.js component for the URL list page (index.html).
 */
export function urlListPage() {
  const data = getPageData();
  return {
    searchQuery: data.Query || '',
    searchType: data.SearchType || 'keyword',
    isSearch: data.IsSearch || false,
    filters: {
      category: data.FilterCategory || '',
      sort: data.FilterSort || '',
      size: String(data.Size || 20),
      isShortURL: data.IsShortURL || false,
      minScore: String(data.MinScore || 0.6),
    },
    nextPage: (data.Page || 1) + 1,
    totalPages: data.TotalPages || 1,
    isLoading: false,
    hasMore: (data.Page || 1) < (data.TotalPages || 1),

    initScroll() {
      if (!this.hasMore) return;
      const observer = new IntersectionObserver((entries) => {
        if (entries[0].isIntersecting && this.hasMore && !this.isLoading) {
          this.loadMore();
        }
      }, { rootMargin: '200px' });
      this.$nextTick(() => {
        if (this.$refs.sentinel) {
          observer.observe(this.$refs.sentinel);
        }
      });
    },

    async loadMore() {
      this.isLoading = true;
      const params = new URLSearchParams();
      params.set('page', this.nextPage);
      params.set('size', this.filters.size);
      if (this.filters.sort) params.set('sort', this.filters.sort);
      if (this.filters.category) params.set('category', this.filters.category);
      if (this.filters.isShortURL) params.set('is_shorturl', '1');
      if (this.searchQuery) {
        params.set('q', this.searchQuery);
        params.set('search_type', this.searchType);
        if (this.filters.minScore && this.filters.minScore !== '0') {
          params.set('min_score', this.filters.minScore);
        }
      }

      try {
        const resp = await fetch('/cards?' + params.toString());
        const html = await resp.text();
        if (html.trim()) {
          document.getElementById('url-list').insertAdjacentHTML('beforeend', html);
          this.nextPage++;
          this.hasMore = this.nextPage <= this.totalPages;
        } else {
          this.hasMore = false;
        }
      } catch (e) {
        console.error('loadMore error:', e);
      }
      this.isLoading = false;
    },

    doSearch() {
      if (!this.searchQuery.trim()) return;
      const params = new URLSearchParams();
      params.set('q', this.searchQuery);
      params.set('search_type', this.searchType);
      if (this.filters.minScore && this.filters.minScore !== '0') {
        params.set('min_score', this.filters.minScore);
      }
      if (this.filters.category) params.set('category', this.filters.category);
      params.set('sort', this.filters.sort);
      params.set('size', this.filters.size);
      if (this.filters.isShortURL) params.set('is_shorturl', '1');
      params.set('page', '1');
      window.location.href = '/?' + params.toString();
    },

    clearSearch() {
      this.searchQuery = '';
      this.applyFilters();
    },

    applyFilters() {
      const params = new URLSearchParams();
      if (this.searchQuery) {
        params.set('q', this.searchQuery);
        params.set('search_type', this.searchType);
        if (this.filters.minScore && this.filters.minScore !== '0') {
          params.set('min_score', this.filters.minScore);
        }
      }
      if (this.filters.category) params.set('category', this.filters.category);
      if (this.filters.sort) params.set('sort', this.filters.sort);
      if (this.filters.size) params.set('size', this.filters.size);
      if (this.filters.isShortURL) params.set('is_shorturl', '1');
      params.set('page', '1');
      window.location.href = '/?' + params.toString();
    },
  };
}
