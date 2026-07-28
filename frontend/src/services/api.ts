import type { JoinResponse, Meeting, User } from '../types'

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...init?.headers },
  })
  if (!response.ok) {
    const body = await response.json().catch(() => ({ error: response.statusText }))
    throw new Error(body.error ?? 'Request failed')
  }
  if (response.status === 204) return undefined as T
  return response.json()
}

export const api = {
  me: () => request<User>('/api/me'),
  logout: () => request<void>('/api/logout', { method: 'POST' }),
  createMeeting: () => request<{ meeting: Meeting; url: string }>('/api/meetings', { method: 'POST' }),
  getMeeting: (id: string) => request<Meeting>(`/api/meetings/${id}`),
  join: (id: string) => request<JoinResponse>(`/api/meetings/${id}/join`, { method: 'POST' }),
  leave: (id: string) => request<void>(`/api/meetings/${id}/leave`, { method: 'POST' }),
  end: (id: string) => request<void>(`/api/meetings/${id}/end`, { method: 'POST' }),
  action: (id: string, action: 'admit'|'reject'|'remove'|'mute', userId: string) =>
    request<Meeting|void>(`/api/meetings/${id}/${action}`, { method: 'POST', body: JSON.stringify({ userId }) }),
  chat: (id: string, text: string) => request(`/api/meetings/${id}/chat`, { method: 'POST', body: JSON.stringify({ text }) }),
  media: (id: string, state: { mic?: boolean; camera?: boolean; screen?: boolean }) =>
    request<void>(`/api/meetings/${id}/media`, { method: 'POST', body: JSON.stringify(state) }),
}

