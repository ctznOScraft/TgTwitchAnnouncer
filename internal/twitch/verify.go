package twitch

import (
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"net/http"
)

// VerifySignature verifies the Twitch EventSub webhook signature.
// Twitch sends: Twitch-Eventsub-Message-Signature = sha256=<hex>
// message = message_id + message_timestamp + raw_body
func VerifySignature(secret string, headers http.Header, body []byte) bool {
	msgID := headers.Get("Twitch-Eventsub-Message-Id")
	timestamp := headers.Get("Twitch-Eventsub-Message-Timestamp")
	signature := headers.Get("Twitch-Eventsub-Message-Signature")

	if msgID == "" || timestamp == "" || signature == "" {
		return false
	}

	message := []byte(msgID + timestamp)
	message = append(message, body...)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(message)
	expectedSig := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	return hmac.Equal([]byte(expectedSig), []byte(signature))
}
