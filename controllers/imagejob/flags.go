package imagejob

import (
	"fmt"
)

type multiFlag []string

func (nss *multiFlag) String() string {
	return fmt.Sprintf("%#v", nss)
}

func (nss *multiFlag) Set(s string) error {
	*nss = append(*nss, s)
	return nil
}
