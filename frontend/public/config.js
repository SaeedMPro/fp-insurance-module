// Runtime frontend config. In Docker this file is overwritten on container
// start from the API_BASE_URL env (see docker-entrypoint.sh). Local Vite
// serves this copy from public/ as-is.
window.__APP_CONFIG__ = {
  apiBaseUrl: '/api/v1',
}
