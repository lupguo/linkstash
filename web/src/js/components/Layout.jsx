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
    <div class="min-h-screen flex flex-col">
      {/* Nav */}
      <nav class="nav-glass sticky top-0 z-50 border-b border-white/[0.06]">
        <div class="max-w-7xl mx-auto px-6 py-3.5 flex items-center justify-between">
          <a href="/" class="font-mono text-base font-bold tracking-wider flex items-center gap-1.5 no-underline text-terminal-green hover:text-white transition-colors">
            <span class="text-terminal-green/50">{'>'}</span>
            <span class="cursor-blink text-terminal-green/30">_</span>
            <span>LinkStash</span>
          </a>
          <div class="flex items-center gap-2.5">
            {isAuthenticated.value ? (
              <>
                <a href="/" class="terminal-btn px-4 py-1.5 no-underline font-medium">Home</a>
                <a href="/urls/new" class="terminal-btn px-4 py-1.5 no-underline font-medium">+ New</a>
                <a href="#" onClick={handleLogout} class="terminal-btn terminal-btn-danger px-4 py-1.5 no-underline font-medium">Logout</a>
              </>
            ) : (
              <a href="/login" class="terminal-btn px-4 py-1.5 no-underline font-medium">Login</a>
            )}
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main class="flex-1 max-w-7xl w-full mx-auto px-6 py-8">
        {children}
      </main>

      {/* Footer */}
      <footer class="border-t border-white/[0.04] py-5 text-center text-gray-700 text-[11px] tracking-[0.2em] font-medium">
        LINKSTASH
      </footer>
    </div>
  );
}
