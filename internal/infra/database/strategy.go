package database

import (
	"context"
	"errors"
)

type Database[T any] interface {
	LoadDatabase(ctx context.Context) (T, error)
}

type DatabaseStrategy[T any] struct {
	strategy Database[T]
}

func NewDatabaseStrategy[T any]() *DatabaseStrategy[T] {
	return &DatabaseStrategy[T]{}
}

func (d *DatabaseStrategy[T]) SetStrategy(ds Database[T]) {
	d.strategy = ds
}

func (d *DatabaseStrategy[T]) LoadDatabase(ctx context.Context) (T, error) {
	if d.strategy == nil {
		var noT T
		return noT, errors.New("no database strategy selected")
	}

	return d.strategy.LoadDatabase(ctx)
}
