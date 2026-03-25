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
      <div class="terminal-card rounded-lg p-8 w-full max-w-md">
        {/* ASCII art header */}
        <pre class="text-terminal-green text-xs mb-4 opacity-60 leading-tight">{
`╔══════════════════════════════════╗
║  ██╗     ███████╗               ║
║  ██║     ██╔════╝               ║
║  ██║     ███████╗  LinkStash    ║
║  ██║     ╚════██║  v1.0.0       ║
║  ███████╗███████║               ║
║  ╚══════╝╚══════╝               ║
╚══════════════════════════════════╝`
        }</pre>

        {/* System info */}
        <div class="text-terminal-gray text-xs mb-4 font-mono space-y-0.5">
          <div><span class="text-terminal-cyan">system</span>  : linkstash-terminal</div>
          <div><span class="text-terminal-cyan">status</span>  : awaiting authentication</div>
          <div><span class="text-terminal-cyan">session</span> : locked</div>
        </div>

        <div class="border-t border-terminal-border mb-4"></div>

        <h1 class="text-terminal-green text-xl mb-6 font-mono">
          {'>'} AUTHENTICATION REQUIRED<span class="cursor-blink">_</span>
        </h1>
        <form onSubmit={handleSubmit}>
          <div class="mb-4">
            <label class="block text-terminal-gray text-sm mb-2">$ enter secret_key</label>
            <input
              type="password"
              class="terminal-input w-full"
              value={secretKey}
              onInput={(e) => setSecretKey(e.target.value)}
              placeholder="••••••••"
              autocomplete="current-password"
              required
            />
          </div>
          {error && (
            <div class="text-terminal-red text-sm mb-4">
              <span class="text-terminal-red">ERR!</span> {error}
            </div>
          )}
          <button
            type="submit"
            class="terminal-btn w-full"
            disabled={loading}
          >
            {loading ? '> AUTHENTICATING...' : '> AUTHENTICATE'}
          </button>
        </form>
      </div>
    </div>
  );
}
