package configstore

import (
	"regexp"
)

var (
	clusterAcceptPattern *regexp.Regexp
)

//SetClusterAcceptPattern compile ptn into regular expression object
func SetClusterAcceptPattern(ptn string) {
	clusterAcceptPattern = regexp.MustCompile(ptn)
}

//CheckAddressAcceptable check if remote address is acceptable
//using string compare is fast than calcuate ip and subnet
func CheckAddressAcceptable(addr []byte) bool {
	return clusterAcceptPattern.Match(addr)
}
