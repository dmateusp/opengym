export const API_BASE_URL =
  (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_BASE_URL) ||
  'http://localhost:8080';

export const oauthLoginUrl = (provider: 'google') =>
  `${API_BASE_URL}/auth/${provider}/login`;
