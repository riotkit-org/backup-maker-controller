package locking_test

import (
	"context"
	"github.com/riotkit-org/backup-maker-operator/pkg/locking"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestInMemoryLocker_Flow(t *testing.T) {
	ctx := context.Background()

	locker := locking.NewInMemoryLocker()

	// 1. Lock is created first time
	first := locker.Obtain(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "world", Namespace: "hello"}},
	)
	assert.Nil(t, first.GetError())

	// 2. Cannot obtain lock second time, error will be returned
	second := locker.Obtain(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "world", Namespace: "hello"}},
	)
	assert.NotNil(t, second.GetError())

	// 3. We release the lock
	locker.Done(ctx, first)

	// 4. Now it could be obtained again after it was released
	third := locker.Obtain(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "world", Namespace: "hello"}},
	)
	assert.Nil(t, third.GetError())
}
