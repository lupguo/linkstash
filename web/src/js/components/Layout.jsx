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
            <span class="cursor-blink">_</span>
            <span>|LinkStash</span>
          </a>
          <div class="flex items-center gap-2 text-sm">
            {isAuthenticated.value ? (
              <>
                <a href="/" class="terminal-btn text-xs px-3 py-1 no-underline">Home</a>
                <a href="/urls/new" class="terminal-btn text-xs px-3 py-1 no-underline">+ New</a>
                <a href="#" onClick={handleLogout} class="terminal-btn text-xs px-3 py-1 no-underline border-terminal-red text-terminal-red hover:bg-terminal-red hover:text-black">Logout</a>
              </>
            ) : (
              <a href="/login" class="terminal-btn text-xs px-3 py-1 no-underline">Login</a>
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
