import { signal, computed } from '@preact/signals';
import { getCookie } from './utils.js';

export const auth = signal({ token: getCookie('linkstash_token') || null });
export const isAuthenticated = computed(() => !!auth.value.token);
