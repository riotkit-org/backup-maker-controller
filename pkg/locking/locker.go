package locking

import (
	"context"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	ErrAlreadyLocked = "Already locked"
)

type LockSession struct {
	id  any
	err error
}

func (ls *LockSession) AlreadyLocked() bool {
	return ls.err != nil && ls.err.Error() == ErrAlreadyLocked
}

func (ls *LockSession) HasFailure() bool {
	return ls.err != nil && ls.err.Error() != ErrAlreadyLocked
}

func (ls *LockSession) GetError() error {
	return ls.err
}

type Locker interface {
	Obtain(ctx context.Context, req ctrl.Request) LockSession
	Done(ctx context.Context, session LockSession)
	Close()
}

func createLockingId(req ctrl.Request) string {
	return req.NamespacedName.String()
}
