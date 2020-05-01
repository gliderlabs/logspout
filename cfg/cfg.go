package cfg

import "os"

// GetEnvDefault is a helper function to retrieve an env variable value OR return a default value
func GetEnvDefault(name, dfault string) string {
	if val := os.Getenv(name); val != "" {
		return val
	}
	return dfault
}
