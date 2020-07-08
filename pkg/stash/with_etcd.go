package stash

import (
	"context"
	"time"

	"github.com/gomods/athens/pkg/config"
	"github.com/gomods/athens/pkg/errors"
	"github.com/gomods/athens/pkg/observ"
	"github.com/gomods/athens/pkg/storage"
	"go.etcd.io/etcd/v3/clientv3"
	"go.etcd.io/etcd/v3/clientv3/concurrency"
)

// WithEtcd returns a distributed singleflight
// using an etcd cluster. If it cannot connect,
// to any of the endpoints, it will return an error.
func WithEtcd(endpoints []string, checker storage.Checker) (Wrapper, error) {
	const op errors.Op = "stash.WithEtcd"
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: time.Second * 5,
	})
	if err != nil {
		return nil, errors.E(op, err)
	}
	for _, endpoint := range endpoints {
		_, err = c.Status(ctx, endpoint)
		if err != nil {
			return nil, errors.E(op, err)
		}
	}
	return func(s Stasher) Stasher {
		return &etcd{c, s, checker}
	}, nil
}

type etcd struct {
	client  *clientv3.Client
	stasher Stasher
	checker storage.Checker
}

func (s *etcd) Stash(ctx context.Context, mod, ver string) (newVer string, err error) {
	const op errors.Op = "etcd.Stash"
	ctx, span := observ.StartSpan(ctx, op.String())
	defer span.End()
	sesh, err := concurrency.NewSession(s.client)
	if err != nil {
		return ver, errors.E(op, err)
	}
	mv := config.FmtModVer(mod, ver)
	mu := concurrency.NewMutex(sesh, mv)
	err = mu.Lock(ctx)
	if err != nil {
		return ver, errors.E(op, err)
	}
	defer func() {
		const op errors.Op = "etcd.Unlock"
		lockErr := mu.Unlock(ctx)
		if err == nil && lockErr != nil {
			err = errors.E(op, lockErr)
		}
	}()
	ok, err := s.checker.Exists(ctx, mod, ver)
	if err != nil {
		return ver, errors.E(op, err)
	}
	if ok {
		return ver, nil
	}
	newVer, err = s.stasher.Stash(ctx, mod, ver)
	if err != nil {
		return ver, errors.E(op, err)
	}
	return newVer, nil
}
