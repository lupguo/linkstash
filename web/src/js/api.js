/**
 * Unified API client for LinkStash /api/* endpoints.
 */

import { getCookie } from './utils.js';
import { auth } from './store.js';
import { route } from 'preact-router';

/**
 * Base fetch wrapper for all API calls.
 * - Attaches JWT from `linkstash_token` cookie as Authorization header.
 * - Always sets Content-Type: application/json.
 * - On 401: redirects to /login and throws.
 * - On 204: returns null.
 * - Otherwise: returns parsed JSON.
 *
 * @param {string} path - e.g. "/api/urls"
 * @param {RequestInit} [options]
 * @returns {Promise<any>}
 */
export async function api(path, options = {}) {
  const token = getCookie('linkstash_token');

  const headers = {
    'Content-Type': 'application/json',
    ...(options.headers || {}),
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  const res = await fetch(path, { ...options, headers });

  if (res.status === 401) {
    // Clear invalid token; component-level auth guards handle the redirect
    document.cookie = 'linkstash_token=;path=/;max-age=0';
    auth.value = { token: null };
    throw new Error('Unauthorized');
  }

  if (res.status === 204) {
    return null;
  }

  return res.json();
}

// ---------------------------------------------------------------------------
// Config (public endpoints)
// ---------------------------------------------------------------------------

export const configApi = {
  /**
   * Get configured categories.
   * @returns {Promise<{categories: string[]}>}
   */
  categories() {
    return fetch('/api/config/categories').then(res => res.json());
  },
};

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export const authApi = {
  /**
   * Obtain a JWT token.
   * @param {string} secretKey
   * @returns {Promise<{token: string}>}
   */
  getToken(secretKey) {
    return fetch('/api/auth/token', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ secret_key: secretKey }),
    }).then(res => {
      if (!res.ok) throw new Error(`Auth failed: ${res.status}`);
      return res.json();
    });
  },
};

// ---------------------------------------------------------------------------
// URLs
// ---------------------------------------------------------------------------

export const urlApi = {
  /**
   * List URLs with optional filters.
   * @param {{ page?: number, size?: number, sort?: string, category?: string, tags?: string, is_shorturl?: number }} [params]
   * @returns {Promise<{urls: object[], total: number}>}
   */
  list(params = {}) {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== undefined && v !== null && v !== '')
    ).toString();
    return api(`/api/urls${qs ? `?${qs}` : ''}`);
  },

  /**
   * Get a single URL by ID.
   * @param {number|string} id
   * @returns {Promise<object>}
   */
  get(id) {
    return api(`/api/urls/${id}`);
  },

  /**
   * Create a new URL entry.
   * @param {{ link: string }} data
   * @returns {Promise<object>}
   */
  create(data) {
    return api('/api/urls', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /**
   * Update an existing URL entry.
   * @param {number|string} id
   * @param {object} data
   * @returns {Promise<object>}
   */
  update(id, data) {
    return api(`/api/urls/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  /**
   * Delete a URL entry.
   * @param {number|string} id
   * @returns {Promise<null>}
   */
  delete(id) {
    return api(`/api/urls/${id}`, { method: 'DELETE' });
  },

  /**
   * Record a visit for a URL.
   * @param {number|string} id
   * @returns {Promise<null>}
   */
  visit(id) {
    return api(`/api/urls/${id}/visit`, { method: 'POST' });
  },

  /**
   * Trigger re-analysis of a URL (re-fetch + LLM extraction).
   * @param {number|string} id
   * @returns {Promise<object>}
   */
  reanalyze(id) {
    return api(`/api/urls/${id}/reanalyze`, { method: 'POST' });
  },
};

// ---------------------------------------------------------------------------
// Search
// ---------------------------------------------------------------------------

export const searchApi = {
  /**
   * Search URLs.
   * @param {{ q: string, type?: 'keyword'|'semantic'|'hybrid', page?: number, size?: number, min_score?: number }} params
   * @returns {Promise<{results: object[], total: number}>}
   */
  search(params = {}) {
    const qs = new URLSearchParams(
      Object.entries(params).filter(([, v]) => v !== undefined && v !== null && v !== '')
    ).toString();
    return api(`/api/search${qs ? `?${qs}` : ''}`);
  },
};

// ---------------------------------------------------------------------------
// Short Links
// ---------------------------------------------------------------------------

export const shortApi = {
  /**
   * Create a short link.
   * @param {{ long_url: string, code?: string, ttl?: number }} data
   * @returns {Promise<object>}
   */
  create(data) {
    return api('/api/short-links', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /**
   * List all short links.
   * @returns {Promise<object[]>}
   */
  list() {
    return api('/api/short-links');
  },

  /**
   * Update a short link.
   * @param {number|string} id
   * @param {object} data
   * @returns {Promise<object>}
   */
  update(id, data) {
    return api(`/api/short-links/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  /**
   * Delete a short link.
   * @param {number|string} id
   * @returns {Promise<null>}
   */
  delete(id) {
    return api(`/api/short-links/${id}`, { method: 'DELETE' });
  },
};
