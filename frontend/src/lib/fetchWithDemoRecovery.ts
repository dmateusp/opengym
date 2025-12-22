import { API_BASE_URL } from './api'

export async function fetchWithDemoRecovery(
  url: string,
  options?: RequestInit
): Promise<Response> {
  let response = await fetch(url, options)

  // If we get a 401 and demo mode is available, try to auto-impersonate and retry
  if (response.status === 401) {
    try {
      const demoResponse = await fetch(`${API_BASE_URL}/api/demo/users`, {
        credentials: 'include',
      })

      // Only attempt recovery if demo mode is actually available (200 response)
      // If we get 403, demo mode is not enabled and we should use normal OAuth flow
      if (demoResponse.status === 200) {
        const demoUsers = await demoResponse.json()
        if (demoUsers.length > 0) {
          // Clear the auto-impersonate flag so we can try again
          sessionStorage.removeItem('demo_auto_impersonate_attempted')
          
          // Impersonate the first user
          const impersonateResponse = await fetch(
            `${API_BASE_URL}/api/demo/users/${demoUsers[0].id}/impersonate`,
            {
              method: 'POST',
              credentials: 'include',
            }
          )

          if (impersonateResponse.ok) {
            // Successfully impersonated, retry the original request
            response = await fetch(url, options)
          }
        }
      }
      // If demo mode is not available (status !== 200), just return the original 401
      // and let the caller handle the OAuth flow
    } catch (error) {
      // If demo recovery fails, just return the original 401 response
      // This preserves the normal OAuth login flow for non-demo environments
      console.debug('Demo recovery failed:', error)
    }
  }

  return response
}

