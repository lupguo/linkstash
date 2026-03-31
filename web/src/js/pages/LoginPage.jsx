import { h } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { route } from 'preact-router';
import { authApi } from '../api.js';
import { auth, isAuthenticated } from '../store.js';

export function LoginPage() {
  const [secretKey, setSecretKey] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isAuthenticated.value) {
      route('/', true);
    }
  }, []);

  async function handleSubmit(e) {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await authApi.getToken(secretKey);
      const token = data.token;
      document.cookie = 'linkstash_token=' + token + ';path=/;max-age=604800';
      auth.value = { token };
      route('/');
    } catch (err) {
      setError('Authentication failed. Check your secret key.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div class="flex items-center justify-center min-h-[60vh]">
      <div class="surface-card rounded-lg p-8 w-full max-w-sm">
        {/* Logo */}
        <div class="flex items-center justify-center gap-2 mb-6">
          <svg class="w-8 h-8 text-accent" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
            <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
          </svg>
          <span class="text-xl font-semibold text-text-primary">LinkStash</span>
        </div>

        <h1 class="text-text-secondary text-sm text-center mb-6">
          Sign in to continue
        </h1>

        <form onSubmit={handleSubmit}>
          <div class="mb-4">
            <label class="block text-text-muted text-xs font-medium uppercase tracking-wider mb-1.5">Secret Key</label>
            <input
              type="password"
              class="input w-full"
              value={secretKey}
              onInput={(e) => setSecretKey(e.target.value)}
              placeholder="Enter your secret key"
              autocomplete="current-password"
              required
            />
          </div>
          {error && (
            <div class="text-red-400 text-sm mb-4 p-2 rounded bg-red-400/10 border border-red-400/20">
              {error}
            </div>
          )}
          <button
            type="submit"
            class="btn btn-primary w-full py-2.5 text-sm font-medium"
            disabled={loading}
          >
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
      </div>
    </div>
  );
}
