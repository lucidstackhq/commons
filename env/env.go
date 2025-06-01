package env

import "os"

func GetOrDefault(key string, def string) string {
	v := os.Getenv(key)

	if v == "" {
		return def
	}

	return v
}
