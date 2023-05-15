package webtoons

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type rsaEncrypter struct {
	publicKey *rsa.PublicKey
}

func newRSA(mod, exponent string) *rsaEncrypter {
	modulus := new(big.Int)
	modulus.SetString(mod, 16)

	e, err := strconv.ParseInt(exponent, 16, 32)
	if err != nil {
		panic("wrong exponent string value: " + exponent)
	}

	return &rsaEncrypter{
		publicKey: &rsa.PublicKey{
			N: modulus,
			E: int(e),
		},
	}
}

// https://github.com/travist/jsencrypt/blob/24e69d8b197b3322d17500af858f3e44e4276f1d/src/lib/jsbn/rsa.ts#L136
func (r *rsaEncrypter) encrypt(message []byte) (string, error) {
	result, err := rsa.EncryptPKCS1v15(rand.Reader, r.publicKey, message)
	if err != nil {
		return "", fmt.Errorf("rsa: encrypt: %w", err)
	}

	return hex.EncodeToString(result), nil
}

var (
	signKey = []byte("gUtPzJFZch4ZyAGviiyH94P99lQ3pFdRTwpJWDlSGFfwgpr6ses5ALOxWHOIT7R1")
)

func getMessage(str, stamp string) string {
	str = str[0:int(math.Min(255, float64(len(str))))]
	return str + stamp
}

// SignRequest sign the request like the mobile app would do, using HMAC-SHA1
// The return value must be used as the request URL
//
// Before:
//
//	_, _ = http.NewRequest(http.MethodGet, getKeysEndpoint, nil)
//
// After:
//
//	_, _ =  http.NewRequest(http.MethodGet, SignRequest(getKeysEndpoint), nil)
func SignRequest(uri string) string {
	mac := hmac.New(sha1.New, signKey)
	stamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	mac.Write([]byte(getMessage(uri, stamp)))
	encoded := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))

	var builder strings.Builder

	builder.WriteString(uri)
	if strings.ContainsRune(uri, '?') {
		builder.WriteRune('&')
	} else {
		builder.WriteRune('?')
	}
	builder.WriteString("msgpad=" + stamp)
	builder.WriteString("&md=" + encoded)

	return builder.String()
}
