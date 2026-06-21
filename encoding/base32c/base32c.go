// See https://www.crockford.com/base32.html.
package base32c

import (
	"encoding/base32"
)

var Encoding = base32.NewEncoding("0123456789ABCDEFGHJKMNPQRSTVWXYZ")

func DecodeString(s string) ([]byte, error) {
	return Encoding.DecodeString(s)
}

func EncodeToString(b []byte) string {
	return Encoding.EncodeToString(b)
}
