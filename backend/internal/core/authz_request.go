package core

import (
	"strings"
	"time"

	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/models"

	"github.com/gin-gonic/gin"
)

// AuthzNonceHeader is the request header carrying the client challenge that a
// signed verdict is bound to (same header the client signature uses).
const AuthzNonceHeader = "X-Nonce"

// authzVerdictTTL bounds how long a signed verdict attached to a client response
// stays valid: short enough to limit replay value, long enough to tolerate clock
// skew and the client verifying it right after receipt.
const authzVerdictTTL = 10 * time.Minute

// SignAuthzForRequest builds a signed "allow" verdict for (app, deviceID) bound
// to this request's X-Nonce challenge, or nil when no signer is available (the
// client then fails closed if it requires authz). Shared by every client
// endpoint that attaches an authz envelope to its response, so device-bound,
// nonce-bound verdicts can be verified the same way everywhere.
func (h *Handler) SignAuthzForRequest(c *gin.Context, app models.App, deviceID string) *auth.AuthzEnvelope {
	signer := h.AuthzSignerForApp(app)
	if signer == nil {
		return nil
	}
	nonce := strings.TrimSpace(c.GetHeader(AuthzNonceHeader))
	env := signer.SignAllow(app.ID.String(), deviceID, nonce, authzVerdictTTL)
	return &env
}
