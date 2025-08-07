package utils

import (
    "crypto/hmac"
    "crypto/sha512"
    "encoding/hex"
)

func VerifyPaystackSignature(body []byte, signature string, secretKey string) bool {
    mac := hmac.New(sha512.New, []byte(secretKey))
    mac.Write(body)
    expectedMAC := mac.Sum(nil)
    expectedSignature := hex.EncodeToString(expectedMAC)

    return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
