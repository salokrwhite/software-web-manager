// Package signature implements the canonical request-signing scheme shared by
// the client SDK signature and the JWT request signature: it builds the
// canonical string from an HTTP request and computes the HMAC-SHA256 signature.
// It is pure (no gin, no database) so it can be reused by middleware, clients,
// and tests alike.
package signature

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

func queryEscapeRFC3986(input string) string {
	s := url.QueryEscape(input)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	return s
}

// CanonicalQuery renders query parameters in the canonical, deterministically
// sorted form used for signing.
func CanonicalQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		vs := append([]string{}, values[k]...)
		sort.Strings(vs)
		ek := queryEscapeRFC3986(k)
		for _, v := range vs {
			parts = append(parts, ek+"="+queryEscapeRFC3986(v))
		}
	}
	return strings.Join(parts, "&")
}

// CanonicalString assembles the canonical string that gets signed: method, path,
// canonical query, body hash, timestamp, nonce, and the signer identity.
func CanonicalString(req *http.Request, bodySHA256 string, timestamp int64, nonce string, identity string) string {
	path := ""
	query := ""
	method := ""
	if req != nil {
		path = req.URL.Path
		query = CanonicalQuery(req.URL.Query())
		method = strings.ToUpper(req.Method)
	}
	return strings.Join([]string{
		method,
		path,
		query,
		bodySHA256,
		strconv.FormatInt(timestamp, 10),
		nonce,
		identity,
	}, "\n")
}

// Sign computes the lowercase hex HMAC-SHA256 of canonical using secret.
func Sign(secret string, canonical string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}
