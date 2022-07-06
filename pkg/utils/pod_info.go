package utils

import "os"

func GetNamespace() string {
	ns, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		return "eraser-system"
	}
	return ns
}
