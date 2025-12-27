import { useSearchParams } from 'react-router-dom'
import OAuthLogin from '@/components/auth/OAuthLogin'

export default function LoginPage() {
  const [searchParams] = useSearchParams()
  const redirectPage = searchParams.get('redirect_page') || undefined

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center px-4">
      <OAuthLogin redirectPage={redirectPage} />
    </div>
  )
}
