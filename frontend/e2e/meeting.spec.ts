import { expect, test, type Browser, type BrowserContext } from '@playwright/test'
import { SignJWT } from 'jose'

const secret = new TextEncoder().encode(
  process.env.JWT_SECRET ?? 'development-only-change-this-secret',
)

async function authenticatedContext(
  browser: Browser,
  identity: { id: string; name: string; email: string },
): Promise<BrowserContext> {
  const token = await new SignJWT({
    Email: identity.email,
    Name: identity.name,
    Avatar: '',
  })
    .setProtectedHeader({ alg: 'HS256', typ: 'JWT' })
    .setSubject(identity.id)
    .setIssuer('instantmeet')
    .setIssuedAt()
    .setExpirationTime('1h')
    .sign(secret)

  const context = await browser.newContext({
    permissions: ['camera', 'microphone'],
    locale: 'en-US',
  })
  await context.addCookies([{
    name: 'instantmeet_token',
    value: token,
    domain: 'localhost',
    path: '/',
    httpOnly: true,
    sameSite: 'Lax',
  }])
  return context
}

test('host creates, admits, chats with guest, and ends a meeting', async ({ browser }) => {
  const hostContext = await authenticatedContext(browser, {
    id: 'e2e-host',
    name: 'E2E Host',
    email: 'host@example.test',
  })
  const guestContext = await authenticatedContext(browser, {
    id: 'e2e-guest',
    name: 'E2E Guest',
    email: 'guest@example.test',
  })
  const observerContext = await authenticatedContext(browser, {
    id: 'e2e-observer',
    name: 'E2E Observer',
    email: 'observer@example.test',
  })

  try {
    const host = await hostContext.newPage()
    const guest = await guestContext.newPage()
    const observer = await observerContext.newPage()

    await host.goto('/')
    await host.getByRole('button', { name: /new meeting/i }).click()
    const roomCode = await host.locator('.created-box strong').innerText()
    const meetingPath = `/meet/${roomCode}`
    const disableMedia = (code: string) => sessionStorage.setItem(
      `instantmeet:media:${code}`,
      JSON.stringify({ micEnabled: false, cameraEnabled: false }),
    )

    await host.evaluate(disableMedia, roomCode)
    let hostJoinRequests = 0
    host.on('request', request => {
      if (request.method() === 'POST' && request.url().endsWith(`/api/meetings/${roomCode}/join`)) {
        hostJoinRequests += 1
      }
    })
    await host.getByRole('button', { name: /enter room/i }).click()
    await expect(host.getByRole('button', { name: /join now/i })).toBeVisible()
    expect(hostJoinRequests).toBe(1)

    await host.getByLabel('Language').selectOption('tr')
    await expect(host.getByRole('button', { name: /şimdi katıl/i })).toBeVisible()
    expect(hostJoinRequests).toBe(1)
    await host.getByLabel('Dil').selectOption('en')

    await host.getByRole('button', { name: /join now/i }).click()
    await expect(host.locator('.meeting-shell')).toBeVisible()

    await guest.goto('/')
    await guest.evaluate(disableMedia, roomCode)
    await guest.goto(meetingPath)
    await expect(guest.getByText(/host knows you/i)).toBeVisible()

    await host.locator('.control-right button').nth(0).click()
    const waitingGuest = host.locator('.person').filter({ hasText: 'E2E Guest' })
    await expect(waitingGuest).toBeVisible()
    await waitingGuest.locator('button.accept').click()

    await expect(guest.getByRole('button', { name: /join now/i })).toBeVisible()
    await guest.getByRole('button', { name: /join now/i }).click()
    await expect(guest.locator('.meeting-shell')).toBeVisible()

    await observer.goto('/')
    await observer.evaluate(disableMedia, roomCode)
    await observer.goto(meetingPath)
    await expect(observer.getByText(/host knows you/i)).toBeVisible()
    const waitingObserver = host.locator('.person').filter({ hasText: 'E2E Observer' })
    await expect(waitingObserver).toBeVisible()
    await waitingObserver.locator('button.accept').click()
    await expect(observer.getByRole('button', { name: /join now/i })).toBeVisible()
    await observer.getByRole('button', { name: /join now/i }).click()
    await expect(observer.locator('.meeting-shell')).toBeVisible()

    await guest.locator('.control-right button').nth(1).click()
    await guest.getByPlaceholder('Send a message').fill('Hello from the guest')
    await guest.getByPlaceholder('Send a message').press('Enter')

    await host.locator('.control-right button').nth(1).click()
    await expect(host.getByText('Hello from the guest')).toBeVisible()
    await host.getByPlaceholder('Send a message').fill('Welcome to InstantMeet')
    await host.getByPlaceholder('Send a message').press('Enter')
    await expect(guest.getByText('Welcome to InstantMeet')).toBeVisible()

    await observer.locator('.control-right button').nth(1).click()
    await expect(observer.getByText('Hello from the guest')).toBeVisible()
    await expect(observer.getByText('Welcome to InstantMeet')).toBeVisible()

    await host.getByLabel('To').selectOption({ label: 'E2E Guest' })
    await host.getByPlaceholder('Send a message').fill('Private for guest only')
    await host.getByPlaceholder('Send a message').press('Enter')
    await expect(guest.getByText('Private for guest only')).toBeVisible()
    await expect(guest.getByText(/Privately from E2E Host/i)).toBeVisible()
    await expect(host.getByText(/Privately to E2E Guest/i)).toBeVisible()
    await expect(observer.getByText('Private for guest only')).toHaveCount(0)
    await expect(observer.getByText(/Privately (to|from)/i)).toHaveCount(0)
    await expect(observer.getByText('Welcome to InstantMeet')).toBeVisible()

    host.on('dialog', dialog => dialog.accept())
    await host.getByTitle('End for everyone').click()
    await expect(host).toHaveURL('/')
    await expect(guest).toHaveURL('/')
    await expect(observer).toHaveURL('/')

    const response = await guest.request.get(`/api/meetings/${roomCode}`)
    expect(response.status()).toBe(404)
  } finally {
    await hostContext.close()
    await guestContext.close()
    await observerContext.close()
  }
})
