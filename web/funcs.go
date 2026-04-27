package web

import (
	"fmt"
)

func fmtPercent(num, denom int64) string {
	if denom == 0 {
		return "0%"
	}
	p := num * 100 / denom
	return fmt.Sprintf("%d%%", p)
}
