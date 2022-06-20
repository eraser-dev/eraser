package utils

import (
	"fmt"
)

type MultiFlag []string

func (nss *MultiFlag) String() string {
	return fmt.Sprintf("%#v", nss)
}

func (nss *MultiFlag) Set(s string) error {
	*nss = append(*nss, s)
	return nil
}
