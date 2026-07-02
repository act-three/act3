// See https://www.crockford.com/base32.html.
package base32c

import (
	"encoding/base32"
)

// Alphabet is the set of symbols used by Encoding,
// in encoding-value order.
const Alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var Encoding = base32.NewEncoding(Alphabet)

func DecodeString(s string) ([]byte, error) {
	return Encoding.DecodeString(s)
}

func EncodeToString(b []byte) string {
	return Encoding.EncodeToString(b)
}
