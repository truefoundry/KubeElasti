package elastiserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"

	"k8s.io/client-go/rest"

	"truefoundry/elasti/operator/internal/crddirectory"
	"truefoundry/elasti/operator/internal/prom"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/truefoundry/elasti/pkg/k8shelper"
	"github.com/truefoundry/elasti/pkg/messages"
	"go.uber.org/zap"
)

type (
	Response struct {
		Message string `json:"message"`
	}

	// Server is used to receive communication from Resolver, or any future components
	// It is used by components about certain events, like when resolver receive the request
	// for a service, that service is scaled up if it's at 0 replicas
	Server struct {
		logger     *zap.Logger
		k8shelper  *k8shelper.Ops
		scaleLocks sync.Map
		// rescaleDuration is the duration to wait before checking to rescaling the target
		rescaleDuration time.Duration
	}
)

func NewServer(logger *zap.Logger, config *rest.Config, rescaleDuration time.Duration) *Server {
	// Get Ops client
	k8sUtil := k8shelper.NewOps(logger, config)
	return &Server{
		logger:          logger.Named("elastiServer"),
		k8shelper:       k8sUtil,
		rescaleDuration: rescaleDuration,
	}
}

// Start starts the ElastiServer and declares the endpoint and handlers for it
func (s *Server) Start(port string) {
	defer func() {
		if rec := recover(); s != nil {
			s.logger.Error("ElastiServer is recovering from panic", zap.Any("error", rec))
			go s.Start(port)
		}
	}()

	sentryHandler := sentryhttp.New(sentryhttp.Options{})
	http.Handle("/metrics", sentryHandler.Handle(promhttp.Handler()))
	http.Handle("/informer/incoming-request", sentryHandler.HandleFunc(s.resolverReqHandler))

	server := &http.Server{
		Addr:              port,
		ReadHeaderTimeout: 2 * time.Second,
	}

	s.logger.Info("Starting ElastiServer", zap.String("port", port))
	if err := server.ListenAndServe(); err != nil {
		s.logger.Fatal("Failed to start StartElastiServer", zap.Error(err))
	}
}

func (s *Server) resolverReqHandler(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			s.logger.Error("Recovered from panic", zap.Any("error", rec))
		}
	}()
	ctx := context.Background()
	if req.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var body messages.RequestCount
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			s.logger.Error("Failed to close Body", zap.Error(err))
		}
	}(req.Body)
	s.logger.Info("-- Received request from Resolver", zap.Any("body", body))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := Response{
		Message: "Request received successfully!",
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = w.Write(jsonResponse)
	if err != nil {
		s.logger.Error("Failed to write response", zap.Error(err))
		return
	}
	err = s.scaleTargetForService(ctx, body.Svc, body.Namespace)
	if err != nil {
		s.logger.Error("Failed to compare and scale target", zap.Error(err))
		return
	}
	s.logger.Info("-- Received fulfilled from Resolver", zap.Any("body", body))
}

func (s *Server) scaleTargetForService(_ context.Context, serviceName, namespace string) error {
	scaleMutex, loaded := s.getMutexForServiceScale(serviceName)
	if loaded {
		return nil
	}
	scaleMutex.Lock()
	defer s.logger.Debug("Scale target lock released")
	s.logger.Debug("Scale target lock taken")
	crd, found := crddirectory.CRDDirectory.GetCRD(serviceName)
	if !found {
		s.releaseMutexForServiceScale(serviceName)
		return fmt.Errorf("scaleTargetForService - error: failed to get CRD details from directory, serviceName: %s", serviceName)
	}

	if err := s.k8shelper.ScaleTargetWhenAtZero(namespace, crd.Spec.ScaleTargetRef.Name, crd.Spec.ScaleTargetRef.Kind, crd.Spec.MinTargetReplicas); err != nil {
		s.releaseMutexForServiceScale(serviceName)
		prom.TargetScaleCounter.WithLabelValues(serviceName, crd.Spec.ScaleTargetRef.Kind+"-"+crd.Spec.ScaleTargetRef.Name, err.Error()).Inc()
		return fmt.Errorf("scaleTargetForService - error: %w, targetRefKind: %s, targetRefName: %s", err, crd.Spec.ScaleTargetRef.Kind, crd.Spec.ScaleTargetRef.Name)
	}
	prom.TargetScaleCounter.WithLabelValues(serviceName, crd.Spec.ScaleTargetRef.Kind+"-"+crd.Spec.ScaleTargetRef.Name, "success").Inc()

	// If the target is scaled up, we will hold the lock for longer, to not scale up again
	time.AfterFunc(s.rescaleDuration, func() {
		s.releaseMutexForServiceScale(serviceName)
	})
	return nil
}

func (s *Server) releaseMutexForServiceScale(service string) {
	lock, loaded := s.scaleLocks.Load(service)
	if !loaded {
		return
	}
	lock.(*sync.Mutex).Unlock()
	s.scaleLocks.Delete(service)
}

func (s *Server) getMutexForServiceScale(serviceName string) (*sync.Mutex, bool) {
	l, loaded := s.scaleLocks.LoadOrStore(serviceName, &sync.Mutex{})
	return l.(*sync.Mutex), loaded
}
