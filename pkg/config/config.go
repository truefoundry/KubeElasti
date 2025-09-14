package config

import (
	"os"
	"strconv"
)

type Config struct {
	Namespace      string
	DeploymentName string
	ServiceName    string
	Port           int32
}

const (
	EnvResolverNamespace      = "ELASTI_RESOLVER_NAMESPACE"
	EnvResolverDeploymentName = "ELASTI_RESOLVER_DEPLOYMENT_NAME"
	EnvResolverServiceName    = "ELASTI_RESOLVER_SERVICE_NAME"
	EnvResolverPort           = "ELASTI_RESOLVER_PORT"
	EnvResolverProxyPort      = "ELASTI_RESOLVER_PROXY_PORT"
	EnvOperatorNamespace      = "ELASTI_OPERATOR_NAMESPACE"
	EnvOperatorDeploymentName = "ELASTI_OPERATOR_DEPLOYMENT_NAME"
	EnvOperatorServiceName    = "ELASTI_OPERATOR_SERVICE_NAME"
	EnvOperatorPort           = "ELASTI_OPERATOR_PORT"
)

type ResolverConfig struct {
	Config

	ReverseProxyPort int32
}

func GetResolverConfig() ResolverConfig {
	return ResolverConfig{
		Config: Config{
			Namespace:      getEnvStringOrPanic(EnvResolverNamespace),
			DeploymentName: getEnvStringOrPanic(EnvResolverDeploymentName),
			ServiceName:    getEnvStringOrPanic(EnvResolverServiceName),
			Port:           getEnvInt32OrPanic(EnvResolverPort),
		},

		ReverseProxyPort: getEnvInt32OrPanic(EnvResolverProxyPort),
	}
}

func GetOperatorConfig() Config {
	return Config{
		Namespace:      getEnvStringOrPanic(EnvOperatorNamespace),
		DeploymentName: getEnvStringOrPanic(EnvOperatorDeploymentName),
		ServiceName:    getEnvStringOrPanic(EnvOperatorServiceName),
		Port:           getEnvInt32OrPanic(EnvOperatorPort),
	}
}

func getEnvStringOrPanic(envName string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		panic("required env value not set: " + envName)
	}

	return envValue
}

func getEnvInt32OrPanic(envName string) int32 {
	envValue := getEnvStringOrPanic(envName)

	envIntValue, err := strconv.ParseInt(envValue, 10, 32)
	if err != nil {
		panic("required env value is not integer: " + envName)
	}

	return int32(envIntValue)
}
