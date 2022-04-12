package imagejob

import (
	"fmt"
	"strings"
)

type stringMap map[string]string

func (lm stringMap) String() string {
	return fmt.Sprintf("%#v", lm)
}

func (lm stringMap) Set(s string) error {
	labels := strings.Split(s, ",")
	for _, label := range labels {
		keyVal := strings.Split(label, "=")
		if len(keyVal) != 2 {
			return fmt.Errorf("label selectors must be key/value pairs, separated by '='")
		}
		lm[keyVal[0]] = keyVal[1]
	}

	return nil
}
