package database

import (
	"context"
	"fmt"
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

func (d *DatabaseStrategy[T]) LoadDatabase(ctx context.Context) (interface{}, error) {
	if d.strategy == nil {
		fmt.Println("No payment strategy selected")
		return nil, nil
	}

	return d.strategy.LoadDatabase(ctx)
}
