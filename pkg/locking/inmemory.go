package locking

import (
	"context"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
)

type InMemoryLocker struct {
	locks []string
}

func (iml *InMemoryLocker) Obtain(ctx context.Context, req ctrl.Request) LockSession {
	ident := createLockingId(req)

	if iml.isLocked(ident) {
		return LockSession{ident, errors.New(ErrAlreadyLocked)}
	}
	iml.locks = append(iml.locks, ident)

	return LockSession{ident, nil}
}

func (iml *InMemoryLocker) Done(ctx context.Context, session LockSession) {
	iml.locks = removeFromList(iml.locks, session.id.(string))
}

func (iml *InMemoryLocker) Close() {
	// nothing
}

func (iml *InMemoryLocker) isLocked(ident string) bool {
	for _, name := range iml.locks {
		if name == ident {
			return true
		}
	}
	return false
}

func NewInMemoryLocker() *InMemoryLocker {
	return &InMemoryLocker{}
}

func removeFromList(s []string, key string) []string {
	index := -1
	for k, v := range s {
		if v == key {
			index = k
			break
		}
	}
	if index == -1 {
		return s
	}
	return append(s[:index], s[index+1:]...)
}
