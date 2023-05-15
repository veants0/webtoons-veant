package encode

import (
	"net/url"
	"strings"
)

type Values map[string][]string

const Order = "_ORDER"

// Encode will encode url values in the right order
// without the std lib sorting
func (v Values) Encode() string {
	if v == nil {
		return ""
	}
	var buf strings.Builder
	for _, k := range v[Order] {
		vs := v[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

func (v Values) Add(key, value string) {
	v[key] = append(v[key], value)
	v[Order] = append(v[Order], key)
}
