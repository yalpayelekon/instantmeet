import { afterEach, describe, expect, it } from 'vitest'
import i18n from './i18n'

describe('localization', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en')
    localStorage.removeItem('instantmeet:language')
  })

  it('provides English as the fallback language', async () => {
    await i18n.changeLanguage('en')

    expect(i18n.t('home.newMeeting')).toBe('New meeting')
    expect(document.documentElement.lang).toBe('en')
  })

  it('switches the interface metadata and translations to Turkish', async () => {
    await i18n.changeLanguage('tr')

    expect(i18n.t('home.newMeeting')).toBe('Yeni toplantı')
    expect(document.documentElement.lang).toBe('tr')
    expect(document.title).toBe('InstantMeet — Süresiz buluşmalar')
    expect(localStorage.getItem('instantmeet:language')).toBe('tr')
  })
})
