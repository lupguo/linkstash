import { h } from 'preact';
import { route } from 'preact-router';
import { isAuthenticated, auth } from '../store.js';

function handleLogout(e) {
  e.preventDefault();
  document.cookie = 'linkstash_token=;path=/;max-age=0';
  auth.value = { token: null };
  route('/login');
}

export function Layout({ children }) {
  return (
    <div class="min-h-screen flex flex-col terminal-bg text-terminal-green font-mono">
      {/* Nav */}
      <nav class="nav-glass sticky top-0 z-50 border-b border-terminal-border">
        <div class="max-w-7xl mx-auto px-4 py-3 flex items-center justify-between">
          <a href="/" class="text-lg font-bold tracking-wider flex items-center gap-1 no-underline text-terminal-green">
            <span class="text-terminal-green">{'>'}</span>
            <span class="animate-pulse">_</span>
            <span>|LinkStash</span>
          </a>
          <div class="flex items-center gap-4 text-sm">
            {isAuthenticated.value ? (
              <>
                <a href="/" class="hover:text-white no-underline text-terminal-green">Home</a>
                <a href="/urls/new" class="hover:text-white no-underline text-terminal-green">+ New URL</a>
                <a href="#" onClick={handleLogout} class="hover:text-terminal-red no-underline text-terminal-green">Logout</a>
              </>
            ) : (
              <a href="/login" class="hover:text-white no-underline text-terminal-green">Login</a>
            )}
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main class="flex-1 max-w-7xl w-full mx-auto px-4 py-6">
        {children}
      </main>

      {/* Footer */}
      <footer class="border-t border-terminal-border py-4 text-center text-terminal-gray text-xs">
        LinkStash // Terminal Mode
      </footer>
    </div>
  );
}
