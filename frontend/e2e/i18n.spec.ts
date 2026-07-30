import { expect, test } from '@playwright/test'

test.use({ locale: 'tr-TR' })

test('detects the browser language and remembers a manual choice', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: /şimdi buluş/i })).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('lang', 'tr')

  const language = page.getByLabel('Dil')
  await language.selectOption('en')

  await expect(page.getByRole('heading', { name: /meet now/i })).toBeVisible()
  await expect(page.locator('html')).toHaveAttribute('lang', 'en')

  await page.reload()
  await expect(page.getByRole('heading', { name: /meet now/i })).toBeVisible()
  await expect(page.getByLabel('Language')).toHaveValue('en')
})
