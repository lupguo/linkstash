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
      <nav class="sticky top-0 z-50 border-b border-border-hi bg-bg-primary/95 backdrop-blur-xs">
        <div class="max-w-[1920px] mx-auto px-4 sm:px-6 lg:px-8 py-3 flex items-center justify-between">
          <a href="/" class="text-base font-semibold tracking-tight flex items-center gap-2 no-underline text-accent hover:text-sky-300 transition-colors">
            <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71" />
              <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" />
            </svg>
            <span>LinkStash</span>
          </a>
          <div class="flex items-center gap-2">
            {isAuthenticated.value ? (
              <>
                <a href="/" class="btn px-3.5 py-1.5 no-underline text-sm">Home</a>
                <a href="/urls/new" class="btn btn-primary px-3.5 py-1.5 no-underline text-sm">+ New Link</a>
                <a href="#" onClick={handleLogout} class="btn btn-danger px-3.5 py-1.5 no-underline text-sm">Logout</a>
              </>
            ) : (
              <a href="/login" class="btn px-3.5 py-1.5 no-underline text-sm">Login</a>
            )}
          </div>
        </div>
      </nav>

      {/* Main content */}
      <main class="flex-1 max-w-[1920px] w-full mx-auto px-4 sm:px-6 lg:px-8 py-6">
        {children}
      </main>

      {/* Footer */}
      <footer class="border-t border-border-default py-4 text-center text-text-muted text-xs tracking-wide">
        LinkStash
      </footer>
    </div>
  );
}
