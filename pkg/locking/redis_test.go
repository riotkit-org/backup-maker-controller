package locking_test

import (
	"context"
	"github.com/riotkit-org/backup-maker-controller/pkg/locking"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"testing"
)

func TestRedisDistributedLockerFlow(t *testing.T) {
	ctx := context.Background()

	//
	// 0. Setup Redis container
	//
	req := testcontainers.ContainerRequest{
		Image:        "ghcr.io/mirrorshub/docker/redis:7.0.7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForExposedPort(),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.Nil(t, err)
	ip, ipErr := container.ContainerIP(ctx)
	assert.Nil(t, ipErr)

	locker := locking.NewRedisDistributedLocker("tcp", ip, 6379)
	//
	// 1. Lock is created first time
	//
	first := locker.Obtain(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "world", Namespace: "hello"}},
	)
	assert.Nil(t, first.GetError())

	//
	// 2. Cannot obtain lock second time, error will be returned
	//
	second := locker.Obtain(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "world", Namespace: "hello"}},
	)
	assert.NotNil(t, second.GetError())

	//
	// 3. We release the lock
	//
	locker.Done(ctx, first)

	//
	// 4. Now it could be obtained again after it was released
	//
	third := locker.Obtain(ctx, controllerruntime.Request{
		NamespacedName: types.NamespacedName{Name: "world", Namespace: "hello"}},
	)
	assert.Nil(t, third.GetError())
}
