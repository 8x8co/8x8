package storage

import (
	"context"
	"errors"
	"strings"

	"github.com/dgraph-io/badger/v3"
	"github.com/gernest/8x8/pkg/models"
)

var ErrNotFound = errors.New("Not found")

type Store interface {
	User() User
}

type storeKey struct{}

func Set(ctx context.Context, v Store) context.Context {
	return context.WithValue(ctx, storeKey{}, v)
}

func Get(ctx context.Context) Store {
	return ctx.Value(storeKey{}).(Store)
}

type User interface {
	Get(ctx context.Context, email string) (*models.User, error)
	Create(ctx context.Context, usr *models.User) error
	List(ctx context.Context) ([]*models.User, error)
}

func key(parts ...string) []byte {
	return []byte(strings.Join(parts, "/"))
}

type DefaultStore struct {
	DB *badger.DB
}

func (d *DefaultStore) User() User {
	return &badgerUSR{db: d.DB}
}
