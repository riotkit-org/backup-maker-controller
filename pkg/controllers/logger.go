package controllers

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	ctrl "sigs.k8s.io/controller-runtime"
)

func createLogger(ctx context.Context, req ctrl.Request, controllerName string) *logrus.Entry {
	id, _ := uuid.NewUUID()
	return logrus.WithContext(ctx).WithFields(map[string]interface{}{
		"name":       req.NamespacedName,
		"controller": controllerName,
		"rid":        id,
	})
}
