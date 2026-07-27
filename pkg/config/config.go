package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	EnvResolverNamespace       = "ELASTI_RESOLVER_NAMESPACE"
	EnvResolverDeploymentName  = "ELASTI_RESOLVER_DEPLOYMENT_NAME"
	EnvResolverServiceName     = "ELASTI_RESOLVER_SERVICE_NAME"
	EnvResolverPort            = "ELASTI_RESOLVER_PORT"
	EnvResolverProxyPort       = "ELASTI_RESOLVER_PROXY_PORT"
	EnvOperatorNamespace       = "ELASTI_OPERATOR_NAMESPACE"
	EnvOperatorDeploymentName  = "ELASTI_OPERATOR_DEPLOYMENT_NAME"
	EnvOperatorServiceName     = "ELASTI_OPERATOR_SERVICE_NAME"
	EnvOperatorPort            = "ELASTI_OPERATOR_PORT"
	EnvKubernetesClusterDomain = "KUBERNETES_CLUSTER_DOMAIN"
	// EnvWatchNamespaces is an optional, comma-separated list of namespaces to confine
	// KubeElasti to. When unset or empty, KubeElasti runs cluster-scoped (the default).
	EnvWatchNamespaces = "WATCH_NAMESPACES"
)

// Config holds component namespace/name/service and listen port sourced from env.
type Config struct {
	Namespace      string
	DeploymentName string
	ServiceName    string
	Port           int32
}

// ResolverConfig embeds Config and adds ReverseProxyPort for the resolver.
type ResolverConfig struct {
	Config

	ReverseProxyPort int32
}

// GetKubernetesClusterDomain reads kubernetes cluster domain or panics if it is missing
func GetKubernetesClusterDomain() string {
	return getEnvStringOrPanic(EnvKubernetesClusterDomain)
}

// GetResolverConfig reads resolver env vars or panics if any are missing or invalid.
func GetResolverConfig() ResolverConfig {
	return ResolverConfig{
		Config: Config{
			Namespace:      getEnvStringOrPanic(EnvResolverNamespace),
			DeploymentName: getEnvStringOrPanic(EnvResolverDeploymentName),
			ServiceName:    getEnvStringOrPanic(EnvResolverServiceName),
			Port:           getEnvPortOrPanic(EnvResolverPort),
		},

		ReverseProxyPort: getEnvPortOrPanic(EnvResolverProxyPort),
	}
}

// GetOperatorConfig reads operator env vars or panics if any are missing or invalid.
func GetOperatorConfig() Config {
	return Config{
		Namespace:      getEnvStringOrPanic(EnvOperatorNamespace),
		DeploymentName: getEnvStringOrPanic(EnvOperatorDeploymentName),
		ServiceName:    getEnvStringOrPanic(EnvOperatorServiceName),
		Port:           getEnvPortOrPanic(EnvOperatorPort),
	}
}

// GetWatchNamespaces returns the list of namespaces KubeElasti is confined to, parsed from the
// WATCH_NAMESPACES env var (comma-separated). An empty result means cluster scope (the default).
func GetWatchNamespaces() []string {
	raw := os.Getenv(EnvWatchNamespaces)
	if raw == "" {
		return nil
	}

	namespaces := make([]string, 0)
	for _, ns := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(ns); trimmed != "" {
			namespaces = append(namespaces, trimmed)
		}
	}
	return namespaces
}

// getEnvStringOrPanic returns the env value or panics if unset.
func getEnvStringOrPanic(envName string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		panic("required env value not set: " + envName)
	}

	return envValue
}

// getEnvPortOrPanic parses env value as tcp port or panics if value is unset of invalid.
func getEnvPortOrPanic(envName string) int32 {
	envValue := getEnvStringOrPanic(envName)

	port, err := strconv.ParseInt(envValue, 10, 32)
	if err != nil {
		panic("required env value is not integer: " + envName)
	}

	if port < 1 || port > 65535 {
		panic(fmt.Sprintf("port out of range for %s: %d (want 1..65535)", envName, port))
	}

	return int32(port)
}
