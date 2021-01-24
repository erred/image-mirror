package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Reconciler struct {
	kind string

	client client.Client
	logger logr.Logger
	mirror *Mirror

	excludeNamespace map[string]struct{}
}

// Reconcile ensures all container images use a mirrored registry
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := r.logger.WithValues("namespace", req.Namespace, r.kind, req.Name)
	res := reconcile.Result{}

	logger.Info("reconcile")

	if _, ok := r.excludeNamespace[req.Namespace]; ok {
		logger.Info("excluded namespace")
		return reconcile.Result{}, nil
	}

	logger.Info("get")

	obj := clientObjectKind(r.kind)

	err := r.client.Get(ctx, req.NamespacedName, obj)
	if k8serrors.IsNotFound(err) {
		logger.Error(err, "not found")
		return res, nil
	} else if err != nil {
		return res, fmt.Errorf("fetch %s %s: %w", r.kind, req.NamespacedName, err)
	}

	logger.Info("ensuring")

	var initContainers, containers []corev1.Container
	switch o := obj.(type) {
	case *appsv1.Deployment:
		initContainers = o.Spec.Template.Spec.InitContainers
		containers = o.Spec.Template.Spec.Containers
	case *appsv1.DaemonSet:
		initContainers = o.Spec.Template.Spec.InitContainers
		containers = o.Spec.Template.Spec.Containers
	}

	var wg sync.WaitGroup
	update0 := ensureContainers(logger, &wg, r.mirror.Ensure, initContainers)
	update1 := ensureContainers(logger, &wg, r.mirror.Ensure, containers)
	wg.Wait()
	if !update0.Load().(bool) && !update1.Load().(bool) {
		logger.Info("no update needed")
		return res, nil
	}

	logger.Info("updating")

	// TODO: use patch
	err = r.client.Update(ctx, obj)
	if err != nil {
		return res, fmt.Errorf("update %s %s: %w", r.kind, req.NamespacedName, err)
	}

	logger.Info("waiting")

	// TODO: wait

	return res, nil
}

// ensureContainers calls ensure for all images in containers,
// returns a bool indicating if anything needs updating
func ensureContainers(logger logr.Logger, wg *sync.WaitGroup, ensure func(string) (string, error), containers []corev1.Container) *atomic.Value {
	var update atomic.Value
	update.Store(false)
	wg.Add(len(containers))
	for i := range containers {
		go func(i int) {
			defer wg.Done()

			img, err := ensure(containers[i].Image)
			if err != nil {
				logger.Error(err, "ensure image in mirror registry", "image", containers[i].Image)
			} else if img != containers[i].Image {
				update.Store(true)
				containers[i].Image = img
			}
		}(i)
	}
	return &update
}

func clientObjectKind(kind string) client.Object {
	var obj client.Object
	switch kind {
	case "deployment":
		obj = &appsv1.Deployment{}
	case "daemonset":
		obj = &appsv1.DaemonSet{}
	default:
		panic("unknown kind: " + kind)
	}
	return obj
}
