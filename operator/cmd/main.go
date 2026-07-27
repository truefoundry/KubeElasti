/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/truefoundry/elasti/pkg/config"
	"github.com/truefoundry/elasti/pkg/scaling"

	"truefoundry/elasti/operator/internal/elastiserver"

	"truefoundry/elasti/operator/internal/crddirectory"
	"truefoundry/elasti/operator/internal/informer"

	tfLogger "github.com/truefoundry/elasti/pkg/logger"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	elastiv1alpha1 "truefoundry/elasti/operator/api/v1alpha1"
	"truefoundry/elasti/operator/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(elastiv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	err := mainWithError()
	if err != nil {
		os.Exit(1)
	}
}

func mainWithError() error {
	// Initialize Sentry
	sentryDsn := os.Getenv("SENTRY_DSN")
	sentryEnabled := sentryDsn != ""

	if sentryEnabled {
		fmt.Println("Initializing Sentry")
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              sentryDsn,
			EnableTracing:    false,
			TracesSampleRate: 1.0,
			Environment:      os.Getenv("SENTRY_ENVIRONMENT"),
		}); err != nil {
			fmt.Println("ERROR: Sentry initialization failed")
		}
		defer sentry.Flush(2 * time.Second)
	}

	var watchNamespace string
	flag.StringVar(&watchNamespace, "watch-namespace", metav1.NamespaceAll, "Namespace to watch for resources")

	zapLogger, err := tfLogger.NewLogger("dev", sentryEnabled)
	if err != nil {
		setupLog.Error(err, "unable to create logger")
	}

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false,
		"If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	var tlsOpts []func(*tls.Config)
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	// Determine the namespaces to confine KubeElasti to. An empty result keeps the default
	// cluster-scoped behavior. When scoped, the operator's and resolver's own namespaces are
	// always folded in so leader-election leases and the resolver's EndpointSlices stay reachable.
	watch := config.GetWatchNamespaces()
	if len(watch) == 0 && watchNamespace != metav1.NamespaceAll {
		// Backward compatibility with the legacy single-namespace --watch-namespace flag.
		watch = []string{watchNamespace}
	}
	watchNamespaces := effectiveWatchNamespaces(watch, config.GetOperatorConfig().Namespace, config.GetResolverConfig().Namespace)
	if len(watchNamespaces) > 0 {
		setupLog.Info("running in namespace-scoped mode", "namespaces", watchNamespaces)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache:  buildCacheOptions(watchNamespaces),
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:                 webhookServer,
		HealthProbeBindAddress:        probeAddr,
		LeaderElection:                enableLeaderElection,
		LeaderElectionID:              "acf50383.truefoundry.io",
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		sentry.CaptureException(err)
		return fmt.Errorf("main: %w", err)
	}

	// Start the shared CRD Directory
	crddirectory.InitDirectory(zapLogger)
	// Initiate and start the shared informerManager manager
	informerManager := informer.NewInformerManager(zapLogger, mgr.GetConfig())
	informerManager.Start()
	defer informerManager.Stop()

	// Initiate and start the shared scaleHandler
	scaleHandler := scaling.NewScaleHandler(zapLogger, mgr.GetConfig(), watchNamespaces, mgr.GetEventRecorderFor("elasti-operator"))

	// Set up the ElastiService controller
	reconciler := &controller.ElastiServiceReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		Logger:          zapLogger,
		InformerManager: informerManager,
		ScaleHandler:    scaleHandler,
	}

	if err = reconciler.SetupWithManager(mgr, watchNamespaces); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ElastiService")
		sentry.CaptureException(err)
		return fmt.Errorf("main: %w", err)
	}

	// Start the elasti server
	eServer := elastiserver.NewServer(zapLogger, scaleHandler, 30*time.Second)
	errChan := make(chan error, 1)
	go func() {
		elastiServerPort := fmt.Sprintf(":%d", config.GetOperatorConfig().Port)
		if err := eServer.Start(elastiServerPort); err != nil {
			setupLog.Error(err, "elasti server failed to start")
			sentry.CaptureException(err)
			errChan <- fmt.Errorf("elasti server: %w", err)
		}
	}()

	// Add error channel check before manager start
	select {
	case err := <-errChan:
		return fmt.Errorf("main: %w", err)
	default:
	}

	//+kubebuilder:scaffold:builder
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		sentry.CaptureException(err)
		return fmt.Errorf("main: %w", err)
	}
	// if err := mgr.AddReadyzCheck("readyz", leaderReadinessCheck(mgr)); err != nil {
	// 	setupLog.Error(err, "unable to set up ready check")
	// 	sentry.CaptureException(err)
	// 	return fmt.Errorf("main: %w", err)
	// }
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		sentry.CaptureException(err)
		return fmt.Errorf("main: %w", err)
	}

	setupLog.Info("starting manager")
	mgrErrChan := make(chan error, 1)
	// we are using a goroutine to start the manager because we don't want to block the main thread
	go func() {
		if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
			setupLog.Error(err, "problem running manager")
			mgrErrChan <- fmt.Errorf("manager: %w", err)
		}
	}()

	// Wait for cache to sync
	if !mgr.GetCache().WaitForCacheSync(context.Background()) {
		return fmt.Errorf("failed to sync cache")
	}

	if err = reconciler.Initialize(context.Background(), watchNamespaces); err != nil {
		setupLog.Error(err, "unable to initialize controller")
		return fmt.Errorf("main: %w", err)
	}
	setupLog.Info("initialized controller")

	if err := <-mgrErrChan; err != nil {
		return fmt.Errorf("main: %w", err)
	}

	return nil
}

// effectiveWatchNamespaces returns the de-duplicated set of namespaces to confine the manager
// cache and clients to. An empty watch slice means cluster scope (returns nil). When scoped, the
// alwaysInclude namespaces (operator + resolver) are folded in — this is a correctness invariant,
// not a nicety, since the resolver's own EndpointSlices must remain reachable.
func effectiveWatchNamespaces(watch []string, alwaysInclude ...string) []string {
	if len(watch) == 0 {
		return nil
	}

	all := make([]string, 0, len(watch)+len(alwaysInclude))
	all = append(all, watch...)
	all = append(all, alwaysInclude...)

	seen := make(map[string]struct{}, len(all))
	out := make([]string, 0, len(all))
	for _, ns := range all {
		if ns == "" {
			continue
		}
		if _, ok := seen[ns]; ok {
			continue
		}
		seen[ns] = struct{}{}
		out = append(out, ns)
	}
	return out
}

// buildCacheOptions restricts the manager cache to the given namespaces. When the list is empty it
// returns the zero-value cache.Options — byte-for-byte identical to the default cluster-scoped
// behavior.
func buildCacheOptions(watchNamespaces []string) cache.Options {
	if len(watchNamespaces) == 0 {
		return cache.Options{}
	}

	defaultNamespaces := make(map[string]cache.Config, len(watchNamespaces))
	for _, ns := range watchNamespaces {
		defaultNamespaces[ns] = cache.Config{}
	}
	return cache.Options{DefaultNamespaces: defaultNamespaces}
}

func _(mgr ctrl.Manager) healthz.Checker {
	return func(_ *http.Request) error {
		select {
		case <-mgr.Elected():
			return nil
		default:
			return fmt.Errorf("controller is not the leader yet")
		}
	}
}
