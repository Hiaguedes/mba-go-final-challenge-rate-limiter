package usecase

import (
	"context"
)

// AllowFunc é a porta de persistência: decide se a identidade pode seguir,
// dado o limite de requisições por segundo e o tempo de bloqueio em segundos.
type AllowFunc[T any] func(ctx context.Context, identifier string, limit, blockSeconds int, client T) (bool, error)

type LimitterUseCase[T any] struct {
	allow AllowFunc[T]
}

func NewLimitterUseCase[T any](allow AllowFunc[T]) *LimitterUseCase[T] {
	return &LimitterUseCase[T]{allow: allow}
}

func (l *LimitterUseCase[T]) Execute(ctx context.Context, identifier string, limit, blockSeconds int, client T) (bool, error) {
	return l.allow(ctx, identifier, limit, blockSeconds, client)
}
