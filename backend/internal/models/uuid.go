package models

import "github.com/google/uuid"

// ensureUUID assigns a fresh UUID to id when it is nil/zero. It is used by the
// BeforeCreate hooks of UUID-keyed entities.
func ensureUUID(id *uuid.UUID) {
	if id == nil || *id != uuid.Nil {
		return
	}
	*id = uuid.New()
}
