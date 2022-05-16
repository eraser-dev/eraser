package imagejob

import (
	"fmt"
)

type nodeSkipSelectors []string

func (nss *nodeSkipSelectors) String() string {
	return fmt.Sprintf("%#v", nss)
}

func (nss *nodeSkipSelectors) Set(s string) error {
	*nss = append(*nss, s)
	return nil
}
