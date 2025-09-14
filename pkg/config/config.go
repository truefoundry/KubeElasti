package config

import (
	"os"
	"strconv"
)

type Config struct {
	Namespace      string
	DeploymentName string
	ServiceName    string
	Port           int
}

type ResolverConfig struct {
	Config

	ReverseProxyPort int
}

func GetResolverConfig() ResolverConfig {
	return ResolverConfig{
		Config: Config{
			Namespace:      getEnvStringOrPanic("ELASTI_RESOLVER_NAMESPACE"),
			DeploymentName: getEnvStringOrPanic("ELASTI_RESOLVER_DEPLOYMENT_NAME"),
			ServiceName:    getEnvStringOrPanic("ELASTI_RESOLVER_SERVICE_NAME"),
			Port:           getEnvIntOrPanic("ELASTI_RESOLVER_PORT"),
		},

		ReverseProxyPort: getEnvIntOrPanic("ELASTI_RESOLVER_PROXY_PORT"),
	}
}

func GetOperatorConfig() Config {
	return Config{
		Namespace:      getEnvStringOrPanic("ELASTI_OPERATOR_NAMESPACE"),
		DeploymentName: getEnvStringOrPanic("ELASTI_OPERATOR_DEPLOYMENT_NAME"),
		ServiceName:    getEnvStringOrPanic("ELASTI_OPERATOR_SERVICE_NAME"),
		Port:           getEnvIntOrPanic("ELASTI_OPERATOR_PORT"),
	}
}

func getEnvStringOrPanic(envName string) string {
	envValue := os.Getenv(envName)
	if envValue == "" {
		panic("required env value not set: " + envName)
	}

	return envValue
}

func getEnvIntOrPanic(envName string) int {
	envValue := getEnvStringOrPanic(envName)

	envIntValue, err := strconv.Atoi(envValue)
	if err != nil {
		panic("required env value is not integer: " + envName)
	}

	return envIntValue
}
