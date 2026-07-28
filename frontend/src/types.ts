export type User = { id: string; email: string; displayName: string; avatar: string }
export type Participant = {
  userId: string; displayName: string; avatar: string; isHost: boolean; joinedAt: string;
  micEnabled: boolean; cameraEnabled: boolean; screenSharing: boolean
}
export type WaitingParticipant = Participant & { requestedAt: string }
export type ChatMessage = { id: string; userId: string; displayName: string; text: string; sentAt: string }
export type Meeting = {
  id: string; hostId: string; createdAt: string; state: 'created'|'waiting'|'active'|'ending';
  participants: Record<string, Participant>; waitingRoom: Record<string, WaitingParticipant>;
  chat: ChatMessage[]; isHost: boolean
}
export type JoinResponse = {
  status: 'waiting'|'admitted'; meeting: Meeting; livekitToken?: string; livekitUrl?: string
}

