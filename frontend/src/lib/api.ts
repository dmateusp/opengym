// Runtime configuration - can be set via window.OPENGYM_CONFIG
// Falls back to Vite env var for development, then localhost
export const API_BASE_URL =
  (typeof window !== 'undefined' && (window as any).OPENGYM_CONFIG?.API_BASE_URL) ||
  (typeof import.meta !== 'undefined' && import.meta.env?.VITE_API_BASE_URL) ||
  'http://localhost:8080';

export const oauthLoginUrl = (provider: 'google', redirectPage?: string) => {
  let targetPage = redirectPage;

  if (!targetPage && typeof window !== 'undefined') {
    const { pathname, search, hash } = window.location;
    targetPage = `${pathname}${search}${hash}`;
  }

  const params = new URLSearchParams();
  if (targetPage) {
    params.set('redirect_page', targetPage);
  }

  const query = params.toString();
  return `/api/auth/${provider}/login${query ? `?${query}` : ''}`;
};

export const loginPagePath = (redirectPage?: string) => {
  const params = new URLSearchParams();
  const page = redirectPage;

  if (page) {
    params.set('redirect_page', page);
  }

  const query = params.toString();
  return `/auth/login${query ? `?${query}` : ''}`;
};

export const redirectToLogin = (redirectPage?: string) => {
  let target = redirectPage;

  if (!target && typeof window !== 'undefined') {
    const { pathname, search, hash } = window.location;
    target = `${pathname}${search}${hash}`;
  }

  window.location.href = loginPagePath(target);
};
