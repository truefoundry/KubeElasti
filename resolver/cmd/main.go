package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"

	sentryhttp "github.com/getsentry/sentry-go/http"

	"github.com/kubeelasti/kubeelasti/resolver/internal/crdcache"
	"github.com/kubeelasti/kubeelasti/resolver/internal/handler"
	"github.com/kubeelasti/kubeelasti/resolver/internal/hostmanager"
	"github.com/kubeelasti/kubeelasti/resolver/internal/operator"
	"github.com/kubeelasti/kubeelasti/resolver/internal/throttler"

	"github.com/kelseyhightower/envconfig"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	elasti_config "github.com/kubeelasti/kubeelasti/pkg/config"
	"github.com/kubeelasti/kubeelasti/pkg/k8shelper"
	"github.com/kubeelasti/kubeelasti/pkg/logger"
	"go.uber.org/zap"
	"k8s.io/client-go/rest"
)

type config struct {
	MaxIdleProxyConns        int `split_words:"true" default:"1000"`
	MaxIdleProxyConnsPerHost int `split_words:"true" default:"100"`
	// ReqTimeout is the timeout for each request
	ReqTimeout int `split_words:"true" default:"600"`
	// TrafficReEnableDuration is the duration for which the traffic is disabled for a host
	// This is also duration for which we don't recheck readiness of the service
	TrafficReEnableDuration int `split_words:"true" default:"30"`
	// TrafficDisableGraceDuration is the delay before disabling traffic after a proxied request completes
	// This time is for EndpointSlice and CNI controller to converge and start to route requests directly
	TrafficDisableGraceDuration int `split_words:"true" default:"15"`
	// OperatorRetryDuration is the duration for which we don't inform the operator
	// about the traffic on the same host
	OperatorRetryDuration int `split_words:"true" default:"30"`
	// QueueRetryDuration is the duration after we retry the requests in queue
	QueueRetryDuration int `split_words:"true" default:"5"`
	// QueueSize is the size of the queue
	QueueSize int `split_words:"true" default:"100"`
	// MaxQueueConcurrency is the maximum number of concurrent requests
	MaxQueueConcurrency int `split_words:"true" default:"10"`
	// InitialCapacity is the initial capacity of the semaphore
	InitialCapacity int `split_words:"true" default:"100"`
	// HeaderForHost is the header to look for to get the host
	HeaderForHost string `split_words:"true" default:"Host"`
	// CRDCachePollIntervalMinutes is the interval in minutes to poll operator for CRD cache
	CRDCachePollIntervalMinutes int `split_words:"true" default:"5"`
	// Sentry config
	SentryDsn string `split_words:"true" default:""`
	SentryEnv string `envconfig:"SENTRY_ENVIRONMENT" default:""`
	// H2C
	EnableH2C bool `envconfig:"ENABLE_H2C" default:"false"`
}

func main() {
	var env config
	if err := envconfig.Process("", &env); err != nil {
		log.Fatal("Failed to process env: ", err)
	}

	sentryEnabled := env.SentryDsn != ""
	if sentryEnabled {
		fmt.Println("Initializing Sentry")
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              env.SentryDsn,
			EnableTracing:    false,
			TracesSampleRate: 1.0,
			Environment:      env.SentryEnv,
		}); err != nil {
			fmt.Println("Sentry initialization failed:", err)
		}
	}

	logger, err := logger.NewLogger("dev", sentryEnabled)
	if err != nil {
		log.Fatal("Failed to get logger: ", err)
	}
	if sentryEnabled {
		defer sentry.Flush(2 * time.Second)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Fatal("Error fetching cluster config", zap.Error(err))
	}

	// Get components required for the handler
	k8sUtil := k8shelper.NewOps(logger, config)
	newOperatorRPC := operator.NewOperatorClient(logger, time.Duration(env.OperatorRetryDuration)*time.Second)
	newHostManager := hostmanager.NewHostManager(logger, time.Duration(env.TrafficReEnableDuration)*time.Second, time.Duration(env.TrafficDisableGraceDuration)*time.Second, env.HeaderForHost)
	crdCache := crdcache.New(logger, newOperatorRPC, time.Duration(env.CRDCachePollIntervalMinutes)*time.Minute)
	crdCache.StartBackground()
	newTransport := throttler.NewProxyAutoTransport(env.MaxIdleProxyConns, env.MaxIdleProxyConnsPerHost)
	newThrottler := throttler.NewThrottler(&throttler.Params{
		QueueRetryDuration:      time.Duration(env.QueueRetryDuration) * time.Second,
		K8sUtil:                 k8sUtil,
		QueueDepth:              env.QueueSize,
		MaxConcurrency:          env.MaxQueueConcurrency,
		InitialCapacity:         env.InitialCapacity,
		TrafficReEnableDuration: time.Duration(env.TrafficReEnableDuration) * time.Second,
		Logger:                  logger,
	})

	// Create an instance of sentryhttp
	sentryHandler := sentryhttp.New(sentryhttp.Options{})

	// Create a handler
	requestHandler := handler.NewHandler(&handler.Params{
		Logger:      logger,
		CRDCache:    crdCache,
		ReqTimeout:  time.Duration(env.ReqTimeout) * time.Second,
		OperatorRPC: newOperatorRPC,
		HostManager: newHostManager,
		Throttler:   newThrottler,
		Transport:   newTransport,
	})

	// Handle all the incoming requests
	reverseProxyPort := fmt.Sprintf(":%d", elasti_config.GetResolverConfig().ReverseProxyPort)
	reverseProxyServerMux := http.NewServeMux()
	reverseProxyServerMux.Handle("/", sentryHandler.HandleFunc(requestHandler.ServeHTTP))

	reverseProxyServer := &http.Server{
		Addr:              reverseProxyPort,
		Handler:           reverseProxyServerMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if env.EnableH2C {
		protocols := new(http.Protocols)
		protocols.SetHTTP1(true)
		protocols.SetUnencryptedHTTP2(true)
		reverseProxyServer.Protocols = protocols
	}

	logger.Info("Reverse Proxy Server starting at ", zap.String("port", reverseProxyPort))
	go func() {
		if err := reverseProxyServer.ListenAndServe(); err != nil {
			logger.Fatal("ListenAndServe Failed: ", zap.Error(err))
		}
	}()

	// Handle all the incoming internal request like from prometheus that are not related to the reverse proxy
	internalPort := fmt.Sprintf(":%d", elasti_config.GetResolverConfig().Port)
	internalServeMux := http.NewServeMux()
	internalServeMux.Handle("/metrics", promhttp.Handler())
	internalServeMux.Handle("/queue-status", sentryHandler.HandleFunc(requestHandler.GetQueueStatus))
	internalServeMux.Handle("/crd-cache-status", sentryHandler.HandleFunc(requestHandler.GetCRDCacheStatus))
	internalServeMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			logger.Error("Error writing response for /healthz: ", zap.Error(err))
			return
		}
	})
	internalServeMux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		if err != nil {
			logger.Error("Error writing response for /readyz: ", zap.Error(err))
			return
		}
	})

	internalServer := &http.Server{
		Addr:              internalPort,
		Handler:           internalServeMux,
		ReadHeaderTimeout: 2 * time.Second,
	}
	logger.Info("Internal Server starting at ", zap.String("port", internalPort))
	if err := internalServer.ListenAndServe(); err != nil {
		logger.Fatal("ListenAndServe Failed: ", zap.Error(err))
	}
}
