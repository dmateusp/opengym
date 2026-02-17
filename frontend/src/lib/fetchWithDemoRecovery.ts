import { API_BASE_URL, IS_DEMO_MODE } from './api'

export async function fetchWithDemoRecovery(
  url: string,
  options?: RequestInit
): Promise<Response> {
  let response = await fetch(url, options)

  // If we get a 401 and demo mode is enabled, try to auto-impersonate and retry
  if (response.status === 401 && IS_DEMO_MODE) {
    try {
      const demoResponse = await fetch(`${API_BASE_URL}/api/demo/users`, {
        credentials: 'include',
      })

      if (demoResponse.ok) {
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
    } catch (error) {
      // If demo recovery fails, just return the original 401 response
      // This preserves the normal OAuth login flow for non-demo environments
      console.debug('Demo recovery failed:', error)
    }
  }

  return response
}

