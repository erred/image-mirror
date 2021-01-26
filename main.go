package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/google/go-containerregistry/pkg/authn"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func main() {
	ctx := signals.SetupSignalHandler()
	opts := NewOptions(flag.CommandLine)
	klog.InitFlags(nil)
	flag.Parse()

	opts.logger.Info("starting setup")
	mgr, err := setup(ctx, opts)
	if err != nil {
		opts.logger.Error(err, "setup failed")
		os.Exit(1)
	}

	opts.logger.Info("starting manager")
	err = mgr.Start(ctx)
	if err != nil {
		opts.logger.Error(err, "manager failed")
		os.Exit(1)
	}
}

type Options struct {
	parallel         int
	registry         string
	logger           logr.Logger
	excludeNamespace map[string]struct{}
}

func NewOptions(fs *flag.FlagSet) Options {
	o := Options{
		logger: klogr.New().WithName("image-mirror"),
		excludeNamespace: map[string]struct{}{
			"kube-system": {},
		},
	}

	fs.IntVar(&o.parallel, "parallel", 1, "parallel deployments/daemonsets to process")
	fs.StringVar(&o.registry, "registry", "index.docker.io/skhlimr", "registry to mirror to")
	fs.Func("excludens", "exclude a namespace, repeatable (default: kube-system)", func(s string) error {
		o.excludeNamespace[s] = struct{}{}
		return nil
	})

	return o
}

func setup(ctx context.Context, opts Options) (manager.Manager, error) {
	conf, err := config.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("setup config: %w", err)
	}

	mgr, err := manager.New(conf, manager.Options{
		Logger:                 opts.logger.WithName("manager"),
		MetricsBindAddress:     ":8080",
		HealthProbeBindAddress: ":8081",
	})
	if err != nil {
		return nil, fmt.Errorf("setup manager: %w", err)
	}
	mgr.AddHealthzCheck("ping", healthz.Ping)
	mgr.AddReadyzCheck("ping", healthz.Ping)

	cl, err := client.New(conf, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("setup client: %w", err)
	}

	mirror := &Mirror{
		registry: opts.registry,
		basic: authn.Basic{
			Username: os.Getenv("REGISTRY_USERNAME"),
			Password: os.Getenv("REGISTRY_PASSWORD"),
		},
	}

	for _, kind := range []string{"deployment", "daemonset"} {
		err = newController(kind, mgr, cl, mirror, opts)
		if err != nil {
			return nil, fmt.Errorf("setup %s controller: %w", kind, err)
		}
	}

	return mgr, nil
}

func newController(kind string, mgr manager.Manager, cl client.Client, mirror *Mirror, opts Options) error {
	ctrl, err := controller.New(kind, mgr, controller.Options{
		MaxConcurrentReconciles: opts.parallel,
		Log:                     opts.logger,
		Reconciler: &Reconciler{
			kind:             kind,
			client:           cl,
			logger:           opts.logger.WithName("reconciler").WithName(kind),
			mirror:           mirror,
			excludeNamespace: opts.excludeNamespace,
		},
	})
	if err != nil {
		return fmt.Errorf("new controller: %w", err)
	}
	err = ctrl.Watch(&source.Kind{Type: clientObjectKind(kind)}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return fmt.Errorf("watch: %w", err)
	}
	return nil
}
