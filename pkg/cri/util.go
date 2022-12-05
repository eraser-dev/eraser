package cri

import (
	"strings"
)

type (
	runtimeVersion string

	errors []error
)

func (errs errors) Error() string {
	s := make([]string, 0, len(errs))
	for _, err := range errs {
		s = append(s, err.Error())
	}

	return strings.Join(s, "\n")
}

func (errs *errors) Append(err error) {
	if err == nil {
		return
	}
	*errs = append(*errs, err)
}
