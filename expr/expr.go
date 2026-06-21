package expr

func IfElse[T any](condition bool, consequent, alternative func() T) T {
	if condition {
		return consequent()
	} else {
		return alternative()
	}
}
