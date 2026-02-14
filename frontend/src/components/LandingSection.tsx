import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { loginPagePath } from '@/lib/api'

export default function LandingSection() {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const redirectPage =
    typeof window !== 'undefined'
      ? `${window.location.pathname}${window.location.search}${window.location.hash}`
      : undefined

  const features = [
    {
      icon: '🏐',
      label: 'landing.createGames',
      descLabel: 'landing.createGamesDesc',
    },
    {
      icon: '👥',
      label: 'landing.manageParticipants',
      descLabel: 'landing.manageParticipantsDesc',
    },
    {
      icon: '🔓',
      label: 'landing.openSource',
      descLabel: 'landing.openSourceDesc',
      href: 'https://github.com/dmateusp/opengym',
    },
  ]

  return (
    <div className="min-h-screen bg-gradient-to-br from-orange-50 via-yellow-50 to-blue-50">
      <div className="container mx-auto px-4 py-20 max-w-5xl">
        {/* Language Switcher */}
        <div className="flex justify-end mb-8">
          <LanguageSwitcher />
        </div>

        {/* Hero Section */}
        <div className="text-center mb-20">
          <div className="mb-6 text-7xl">🏐</div>
          <h1 className="text-5xl font-bold text-gray-900 mb-4">{t('landing.title')}</h1>
          <p className="text-xl text-gray-600 mb-8 max-w-2xl mx-auto">
            {t('landing.subtitle')}
          </p>
          <Button
            size="lg"
            onClick={() => navigate(loginPagePath(redirectPage))}
            className="bg-accent hover:bg-accent/90 px-8 py-6 text-lg"
          >
            {t('landing.getStarted')}
          </Button>
        </div>

        {/* Features Grid */}
        <div className="grid md:grid-cols-3 gap-6 mb-20">
          {features.map((feature, index) => {
            const cardContent = (
              <>
                <CardHeader>
                  <div className="text-4xl mb-3">{feature.icon}</div>
                  <CardTitle className="text-gray-900">{t(feature.label)}</CardTitle>
                </CardHeader>
                <CardContent>
                  <CardDescription className="text-gray-600">
                    {t(feature.descLabel)}
                  </CardDescription>
                </CardContent>
              </>
            )

            if (feature.href) {
              return (
                <a key={index} href={feature.href} target="_blank" rel="noopener noreferrer">
                  <Card className="border-2 border-gray-200 hover:shadow-xl hover:border-accent hover:-translate-y-1 transition-all cursor-pointer h-full group">
                    <CardHeader>
                      <div className="text-4xl mb-3">{feature.icon}</div>
                      <CardTitle className="text-gray-900 flex items-center gap-2 group-hover:text-accent transition-colors">
                        {t(feature.label)}
                        <span className="text-lg group-hover:translate-x-1 transition-transform">↗</span>
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <CardDescription className="text-gray-600">
                        {t(feature.descLabel)}
                      </CardDescription>
                    </CardContent>
                  </Card>
                </a>
              )
            }

            return (
              <Card key={index} className="border-2 border-gray-200 hover:shadow-lg transition-shadow">
                {cardContent}
              </Card>
            )
          })}
        </div>

        {/* Additional Info Section */}
        <div className="bg-white rounded-lg shadow-md p-8 text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">
            {t('landing.cta')}
          </h2>
          <p className="text-gray-600 mb-6 max-w-xl mx-auto">
            {t('landing.ctaDescription')}
          </p>
          <Button
            size="lg"
            onClick={() => navigate(loginPagePath(redirectPage))}
            className="bg-accent hover:bg-accent/90"
          >
            {t('landing.signIn')}
          </Button>
        </div>
      </div>
    </div>
  )
}
