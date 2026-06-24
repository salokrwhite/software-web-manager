package core

import (
	"encoding/json"

	"software-web-manager/backend/internal/services/ws"
)

// PublishTicketEvent broadcasts a ticket event to subscribed websocket clients.
// It is the cross-domain entry point used by ticket handlers; the underlying hub
// lives in services/ws.
func (h *Handler) PublishTicketEvent(eventType, ticketID, orgID string, payload any) {
	if h.Hub == nil {
		return
	}
	event := ws.Event{
		Type:     eventType,
		TicketID: ticketID,
		OrgID:    orgID,
		Payload:  payload,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.Hub.Publish(ticketID, data)
}
