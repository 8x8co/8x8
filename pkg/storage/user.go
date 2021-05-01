package storage

import (
	"context"
	"errors"

	"github.com/dgraph-io/badger/v3"
	"github.com/gernest/8x8/pkg/models"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
)

const (
	profile = "profile"
)

type badgerUSR struct {
	db *badger.DB
}

func (b *badgerUSR) Create(ctx context.Context, usr *models.User) error {
	old, err := b.Get(ctx, usr.Email)
	return b.db.Update(func(txn *badger.Txn) error {
		k := key(profile, usr.Email)
		if err == nil {
			if proto.Equal(usr, &models.User{
				Name:    old.Name,
				Email:   old.Email,
				Picture: old.Picture,
			}) {
				return nil
			}
			usr.CreatedAt = old.CreatedAt
			usr.UpdatedAt = ptypes.TimestampNow()
		} else {
			usr.CreatedAt = ptypes.TimestampNow()
		}
		b, _ := proto.Marshal(usr)
		return txn.Set(k, b)
	})
}

func (b *badgerUSR) Get(ctx context.Context, email string) (m *models.User, err error) {
	err = b.db.View(func(txn *badger.Txn) error {
		k := key(profile, email)
		it, err := txn.Get(k)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return ErrNotFound
			}
			return err
		}
		return it.Value(func(val []byte) error {
			m = &models.User{}
			return proto.Unmarshal(val, m)
		})
	})
	return
}

func (b *badgerUSR) List(ctx context.Context) (ls []*models.User, err error) {
	err = b.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(profile)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			return it.Item().Value(func(v []byte) error {
				m := &models.User{}
				if err := proto.Unmarshal(v, m); err != nil {
					return err
				}
				ls = append(ls, m)
				return nil
			})
		}
		return nil
	})
	return
}
