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
	ENV_RESOLVER_NAMESPACE       = "ELASTI_RESOLVER_NAMESPACE"
	ENV_RESOLVER_DEPLOYMENT_NAME = "ELASTI_RESOLVER_DEPLOYMENT_NAME"
	ENV_RESOLVER_SERVICE_NAME    = "ELASTI_RESOLVER_SERVICE_NAME"
	ENV_RESOLVER_PORT            = "ELASTI_RESOLVER_PORT"
	ENV_RESOLVER_PROXY_PORT      = "ELASTI_RESOLVER_PROXY_PORT"
	ENV_OPERATOR_NAMESPACE       = "ELASTI_OPERATOR_NAMESPACE"
	ENV_OPERATOR_DEPLOYMENT_NAME = "ELASTI_OPERATOR_DEPLOYMENT_NAME"
	ENV_OPERATOR_SERVICE_NAME    = "ELASTI_OPERATOR_SERVICE_NAME"
	ENV_OPERATOR_PORT            = "ELASTI_OPERATOR_PORT"
)

type ResolverConfig struct {
	Config

	ReverseProxyPort int32
}

func GetResolverConfig() ResolverConfig {
	return ResolverConfig{
		Config: Config{
			Namespace:      getEnvStringOrPanic(ENV_RESOLVER_NAMESPACE),
			DeploymentName: getEnvStringOrPanic(ENV_RESOLVER_DEPLOYMENT_NAME),
			ServiceName:    getEnvStringOrPanic(ENV_RESOLVER_SERVICE_NAME),
			Port:           getEnvInt32OrPanic(ENV_RESOLVER_PORT),
		},

		ReverseProxyPort: getEnvInt32OrPanic(ENV_RESOLVER_PROXY_PORT),
	}
}

func GetOperatorConfig() Config {
	return Config{
		Namespace:      getEnvStringOrPanic(ENV_OPERATOR_NAMESPACE),
		DeploymentName: getEnvStringOrPanic(ENV_OPERATOR_DEPLOYMENT_NAME),
		ServiceName:    getEnvStringOrPanic(ENV_OPERATOR_SERVICE_NAME),
		Port:           getEnvInt32OrPanic(ENV_OPERATOR_PORT),
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
