package apptools

import (
	"os"
	"strconv"
)

func Getenv(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return defaultValue
}

func GetenvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value != "" {
		intVal, err := strconv.Atoi(value)
		if err != nil {
			panic(err)
		}
		return intVal
	}
	return defaultValue
}

func GetenvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value != "" {
		return value == "true"
	}
	return defaultValue
}
