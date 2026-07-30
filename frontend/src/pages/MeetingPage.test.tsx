import { act, render, waitFor } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import i18n from '../i18n'
import type { JoinResponse, User } from '../types'

const { join } = vi.hoisted(() => ({ join: vi.fn() }))

vi.mock('../services/api', () => ({
  api: {
    join,
    leave: vi.fn(),
    end: vi.fn(),
    chat: vi.fn(),
    action: vi.fn(),
    media: vi.fn(),
  },
}))

vi.mock('../hooks/useMeetingSocket', () => ({
  useMeetingSocket: () => ({ status: 'open', retryNow: vi.fn() }),
}))

vi.mock('../components/PreJoin', () => ({
  PreJoin: () => <div data-testid="prejoin" />,
}))

vi.mock('react-router-dom', async importOriginal => {
  const actual = await importOriginal<typeof import('react-router-dom')>()
  return {
    ...actual,
    useNavigate: () => vi.fn(),
    useParams: () => ({ id: 'room-language-test' }),
  }
})

import MeetingPage from './MeetingPage'

const user: User = {
  id: 'guest',
  email: 'guest@example.test',
  displayName: 'Guest',
  avatar: '',
}

const waiting: JoinResponse = {
  status: 'waiting',
  meeting: {
    id: 'room-language-test',
    hostId: 'host',
    createdAt: new Date().toISOString(),
    participants: {},
    waitingRoom: {},
    chat: [],
    state: 'waiting',
    isHost: false,
  },
  livekitToken: '',
  livekitUrl: '',
}

describe('MeetingPage localization', () => {
  afterEach(async () => {
    join.mockReset()
    await i18n.changeLanguage('en')
  })

  it('does not join the meeting again when the language changes', async () => {
    join.mockResolvedValue(waiting)
    render(<MeetingPage user={user} />)
    await waitFor(() => expect(join).toHaveBeenCalledTimes(1))

    await act(async () => {
      await i18n.changeLanguage('tr')
    })

    expect(join).toHaveBeenCalledTimes(1)
  })
})
