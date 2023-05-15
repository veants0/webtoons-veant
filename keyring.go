package webtoons

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// KeyRing is used to store the necessary parameters for encryption
type KeyRing struct {
	SessionKey string `json:"sessionKey"`
	Modulus    string `json:"evalue"`
	Exponent   string `json:"nvalue"`
	KeyName    string `json:"keyName"`
}

func getlenChar(val string) rune {
	return rune(utf8.RuneCountInString(val))
}

// EncryptData encrypts the password and the email using RSA PKCS1
func (k *KeyRing) EncryptData(email, password string) (string, error) {
	var buf bytes.Buffer

	buf.WriteRune(getlenChar(k.SessionKey))
	buf.WriteString(k.SessionKey)

	buf.WriteRune(getlenChar(email))
	buf.WriteString(email)

	buf.WriteRune(getlenChar(password))
	buf.WriteString(password)

	toEncrypt := buf.Bytes()

	e := newRSA(k.Modulus, k.Exponent)

	ciphertext, err := e.encrypt(toEncrypt)
	if err != nil {
		return "", fmt.Errorf("keyring: %w", err)
	}

	return ciphertext, nil
}
