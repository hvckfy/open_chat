package environment

import "os"

/*
Gets value for environment if exists, otherwise return default value
*/
func GetEnvValue(key string, defaultValue string) string {
	envVal := os.Getenv(key)
	if envVal == "" {
		return defaultValue
	}
	return envVal
}
