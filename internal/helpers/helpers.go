package helpers

import (
	"math/rand"
)

var chars = []rune("abcdefghijkmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandString(l int) string {
	b := make([]rune, l)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
