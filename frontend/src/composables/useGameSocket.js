import { ref } from 'vue'
import { API_URL } from '@/api'

// useGameSocket opens a realtime WebSocket to a game and forwards server events
// to `onEvent(type, data)`. The server only pushes lightweight invalidation
// signals; the caller refreshes the relevant slice over the REST API.
//
// Reconnects automatically with exponential backoff. The JWT is passed as a
// query param because browsers can't set an Authorization header on the
// WebSocket handshake.
export function useGameSocket(getGameId, onEvent) {
  const status = ref('disconnected') // 'connecting' | 'open' | 'disconnected'

  let socket = null
  let reconnectTimer = null
  let attempts = 0
  let manualClose = false

  function buildUrl() {
    const gameId = typeof getGameId === 'function' ? getGameId() : getGameId
    if (!gameId) return null
    const token = localStorage.getItem('access_token')
    if (!token) return null
    const base = API_URL.replace(/^http/i, 'ws')
    return `${base}/api/ws/games/${gameId}?token=${encodeURIComponent(token)}`
  }

  function teardownSocket() {
    if (socket) {
      socket.onopen = socket.onmessage = socket.onclose = socket.onerror = null
      try { socket.close() } catch { /* already closing */ }
      socket = null
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer || manualClose) return
    attempts += 1
    // 2s, 4s, 8s, 16s, capped at 30s
    const delay = Math.min(30000, 1000 * 2 ** Math.min(attempts, 5))
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, delay)
  }

  function connect() {
    manualClose = false
    const url = buildUrl()
    if (!url) return

    teardownSocket()
    status.value = 'connecting'

    try {
      socket = new WebSocket(url)
    } catch {
      scheduleReconnect()
      return
    }

    socket.onopen = () => {
      attempts = 0
      status.value = 'open'
    }

    socket.onmessage = (event) => {
      let payload
      try {
        payload = JSON.parse(event.data)
      } catch {
        return
      }
      if (payload?.type) onEvent(payload.type, payload.data || {})
    }

    socket.onclose = () => {
      status.value = 'disconnected'
      scheduleReconnect()
    }

    socket.onerror = () => {
      // An error is always followed by onclose, which handles reconnect.
      try { socket?.close() } catch { /* noop */ }
    }
  }

  function disconnect() {
    manualClose = true
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    teardownSocket()
    status.value = 'disconnected'
  }

  return { status, connect, disconnect }
}
