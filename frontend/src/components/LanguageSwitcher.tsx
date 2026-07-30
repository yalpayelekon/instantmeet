import { Languages } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function LanguageSwitcher({ compact = false }: { compact?: boolean }) {
  const { i18n, t } = useTranslation()
  const language = i18n.resolvedLanguage === 'tr' ? 'tr' : 'en'

  return (
    <label className={`language-switcher${compact ? ' compact' : ''}`}>
      <Languages aria-hidden="true" />
      <span className="sr-only">{t('common.language')}</span>
      <select
        aria-label={t('common.language')}
        value={language}
        onChange={event => void i18n.changeLanguage(event.target.value)}
      >
        <option value="tr">{t('common.turkish')}</option>
        <option value="en">{t('common.english')}</option>
      </select>
    </label>
  )
}
