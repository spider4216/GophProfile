package repositories

import "context"

type RepositoryInterface interface {
	Ping(ctx context.Context) error
	Source() any
}
