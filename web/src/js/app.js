/**
 * LinkStash — Frontend Entry Point
 *
 * Vendor libraries and Alpine.js components bundled together.
 * Alpine must initialize AFTER our components are on window.
 */

// Vendor: htmx
import './vendor/htmx.min.js';

// Alpine components
import { urlListPage } from './alpine/url-list.js';
import { urlCard } from './alpine/url-card.js';
import { detailPage } from './alpine/detail-page.js';
import { loginForm } from './alpine/login-form.js';

// Utilities
import { copyToClipboard } from './utils.js';

// Expose components to window for Alpine.js x-data bindings
window.urlListPage = urlListPage;
window.urlCard = urlCard;
window.detailPage = detailPage;
window.loginForm = loginForm;
window.copyToClipboard = copyToClipboard;

// Alpine.js — import last. Its queueMicrotask(() => Alpine.start())
// runs after the current synchronous execution, but since esbuild
// hoists imports, we need to ensure our window assignments happen first.
// The solution: use dynamic import or defer Alpine's start.
// Since Alpine CDN auto-starts, we'll just import it and rely on the
// fact that all our window assignments above are synchronous and run
// before Alpine's DOMContentLoaded/microtask fires.
import './vendor/alpine.min.js';
