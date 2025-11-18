package env

import "context"

type Env struct {
	ctx context.Context
}

func Empty() Env {
	return Env{context.Background()}
}

func WithValue(e Env, key, value any) Env {
	return Env{context.WithValue(e.ctx, key, value)}
}

func Value(e Env, key any) any {
	return e.ctx.Value(key)
}
