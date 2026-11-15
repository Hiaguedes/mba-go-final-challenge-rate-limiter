package usecase

import (
	"context"
)

type LimitterUseCase[T any] struct{
	Allow func(ctx context.Context, identifier string, isToken bool, client T) (bool, error)
}

func NewLimitterUseCase[T any](Allow func(ctx context.Context, identifier string, isToken bool, client T) (bool, error)) *LimitterUseCase[T] {
	return &LimitterUseCase[T]{
		Allow: Allow,
	}
}

type LimitterInputDTO[T any] struct {
	Client   T
	Capacity int
	FillRate int
}

func (l *LimitterUseCase[T]) Execute(ctx context.Context, identifier string, isToken bool, client T) (bool, error) {
	res, err := l.Allow(ctx, identifier, isToken, client)

	if err != nil {
		return false, err
	}

	return res, nil
}
