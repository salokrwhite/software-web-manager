export type WsEvent = {
  type: string
  ticket_id?: string
  org_id?: string
  payload?: any
  message?: string
}

export type WsStatus = 'idle' | 'connecting' | 'open' | 'closed' | 'error'

type EventHandler = (event: WsEvent) => void
type StatusHandler = (status: WsStatus) => void

class TicketWS {
  private socket: WebSocket | null = null
  private handlers = new Set<EventHandler>()
  private statusHandlers = new Set<StatusHandler>()
  private subs = new Set<string>()
  private reconnectAttempts = 0
  private reconnectTimer: number | null = null
  private status: WsStatus = 'idle'
  private manualClose = false

  onEvent(handler: EventHandler) {
    this.handlers.add(handler)
    return () => this.handlers.delete(handler)
  }

  onStatus(handler: StatusHandler) {
    this.statusHandlers.add(handler)
    handler(this.status)
    return () => this.statusHandlers.delete(handler)
  }

  subscribe(ticketId: string) {
    if (!ticketId) return
    this.subs.add(ticketId)
    this.manualClose = false
    this.ensureConnected()
    this.send({ type: 'subscribe', ticket_id: ticketId })
  }

  unsubscribe(ticketId: string) {
    if (!ticketId) return
    this.subs.delete(ticketId)
    this.send({ type: 'unsubscribe', ticket_id: ticketId })
    if (this.subs.size === 0) {
      this.close()
    }
  }

  private setStatus(status: WsStatus) {
    this.status = status
    this.statusHandlers.forEach((handler) => handler(status))
  }

  private ensureConnected() {
    if (this.socket && (this.socket.readyState === WebSocket.OPEN || this.socket.readyState === WebSocket.CONNECTING)) {
      return
    }
    const token = sessionStorage.getItem('access_token')
    if (!token) {
      this.setStatus('closed')
      return
    }
    const base = import.meta.env.VITE_API_BASE || 'http://localhost:8080'
    const wsBase = base.replace(/^http/, 'ws')
    const url = `${wsBase}/api/ws?token=${encodeURIComponent(token)}`

    this.setStatus('connecting')
    this.socket = new WebSocket(url)

    this.socket.onopen = () => {
      this.setStatus('open')
      this.reconnectAttempts = 0
      this.subs.forEach((ticketId) => {
        this.send({ type: 'subscribe', ticket_id: ticketId })
      })
    }

    this.socket.onmessage = (evt) => {
      void this.handleIncoming(evt.data)
    }

    this.socket.onerror = () => {
      this.setStatus('error')
    }

    this.socket.onclose = () => {
      this.socket = null
      if (this.manualClose) {
        this.setStatus('closed')
        return
      }
      this.setStatus('closed')
      this.scheduleReconnect()
    }
  }

  private async handleIncoming(data: any) {
    try {
      const text = await this.readAsText(data)
      if (!text) return
      const parsed = JSON.parse(text)
      if (parsed?.type === 'ping') {
        this.send({ type: 'pong' })
        return
      }
      this.handlers.forEach((handler) => handler(parsed))
    } catch {
      // ignore malformed messages
    }
  }

  private async readAsText(data: any): Promise<string> {
    if (typeof data === 'string') {
      return data
    }
    if (data instanceof Blob) {
      return await data.text()
    }
    if (data instanceof ArrayBuffer) {
      return new TextDecoder().decode(data)
    }
    if (ArrayBuffer.isView(data)) {
      return new TextDecoder().decode(data.buffer)
    }
    return ''
  }

  private scheduleReconnect() {
    if (this.subs.size === 0) return
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer)
    }
    const delay = Math.min(10000, 1000 * Math.pow(2, this.reconnectAttempts))
    this.reconnectAttempts += 1
    this.reconnectTimer = window.setTimeout(() => this.ensureConnected(), delay)
  }

  private send(payload: any) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return
    }
    this.socket.send(JSON.stringify(payload))
  }

  private close() {
    this.manualClose = true
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer)
      this.reconnectTimer = null
    }
    if (this.socket) {
      this.socket.close()
      this.socket = null
    }
    this.setStatus('closed')
  }
}

const ticketWS = new TicketWS()

export default ticketWS
