package handlers

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	otpIssuer        = "SWM"
	otpPeriodSeconds = 30
	otpDigits        = 6
	otpSkew          = 1
)

var otpEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func generateOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return otpEncoding.EncodeToString(raw), nil
}

func buildOTPAuthURL(email, secret string) string {
	label := fmt.Sprintf("%s:%s", otpIssuer, email)
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", otpIssuer)
	values.Set("period", strconv.Itoa(otpPeriodSeconds))
	values.Set("digits", strconv.Itoa(otpDigits))
	return fmt.Sprintf("otpauth://totp/%s?%s", url.PathEscape(label), values.Encode())
}

func validateTOTP(secret, code string) bool {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	code = strings.TrimSpace(code)
	if secret == "" || code == "" {
		return false
	}
	if len(code) != otpDigits {
		return false
	}
	for _, ch := range code {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	secretBytes, err := otpEncoding.DecodeString(secret)
	if err != nil {
		return false
	}
	counter := time.Now().Unix() / otpPeriodSeconds
	for i := -otpSkew; i <= otpSkew; i++ {
		if counter+int64(i) < 0 {
			continue
		}
		if hotp(secretBytes, uint64(counter+int64(i))) == code {
			return true
		}
	}
	return false
}

func hotp(secret []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binCode := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	mod := 1
	for i := 0; i < otpDigits; i++ {
		mod *= 10
	}
	otp := binCode % mod
	return fmt.Sprintf("%0*d", otpDigits, otp)
}

