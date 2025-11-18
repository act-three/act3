package web

import (
	"fmt"
)

func fmtSize(size int64) string {
	const (
		k = 1000
		M = 1000 * k
		G = 1000 * M
		T = 1000 * G
		P = 1000 * T
	)
	switch {
	case size > P*10:
		return fmt.Sprintf("%dPB", size/P)
	case size > P:
		return fmt.Sprintf("%.1fPB", float64(size)/P)
	case size > T*10:
		return fmt.Sprintf("%dTB", size/T)
	case size > T:
		return fmt.Sprintf("%.1fTB", float64(size)/T)
	case size > G*10:
		return fmt.Sprintf("%dGB", size/G)
	case size > G:
		return fmt.Sprintf("%.1fGB", float64(size)/G)
	case size > M*10:
		return fmt.Sprintf("%dMB", size/M)
	case size > M:
		return fmt.Sprintf("%.1fMB", float64(size)/M)
	case size > k*10:
		return fmt.Sprintf("%dkB", size/k)
	case size > k:
		return fmt.Sprintf("%.1fkB", float64(size)/k)
	}
	return fmt.Sprintf("%d bytes", size)
}

func fmtPercent(num, denom int64) string {
	if denom == 0 {
		return "0%"
	}
	p := num * 100 / denom
	return fmt.Sprintf("%d%%", p)
}
