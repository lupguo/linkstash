/**
 * Alpine.js component for the login form.
 */
export function loginForm() {
  return {
    secretKey: '',
    error: '',
    success: false,
    loading: false,

    async submit() {
      this.error = '';
      this.success = false;
      this.loading = true;

      try {
        const resp = await fetch('/api/auth/token', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ secret_key: this.secretKey }),
        });

        const data = await resp.json();

        if (!resp.ok) {
          this.error = (data.error && data.error.message) || 'authentication failed';
          this.loading = false;
          return;
        }

        const maxAge = data.expires_in || 259200;
        document.cookie = `linkstash_token=${data.token}; path=/; max-age=${maxAge}; SameSite=Strict`;

        this.success = true;
        setTimeout(() => { window.location.href = '/'; }, 800);
      } catch (e) {
        this.error = 'network error: unable to reach server';
        this.loading = false;
      }
    }
  };
}
