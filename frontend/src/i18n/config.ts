import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import en from './locales/en.json'
import ptPT from './locales/pt-PT.json'

i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      'pt-PT': { translation: ptPT },
    },
    lng: localStorage.getItem('language') || 'en',
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false,
    },
  })

export default i18n
