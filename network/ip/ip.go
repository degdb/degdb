// Package ip returns the current IP address by contacting a number of public servers.
package ip

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
)

var ipServers = []string{
	"https://ipinfo.io/ip",
	"https://icanhazip.com/",
	"https://api.ipify.org/",
	"https://myexternalip.com/raw",
}

// IP returns the current IP address from a quorum of external servers.
func IP() (string, error) {
	ips := make(chan string, 1)
	errs := make(chan error, 1)
	for _, server := range ipServers {
		server := server
		go func() {
			resp, err := http.Get(server)
			if err != nil {
				errs <- err
				return
			}
			body, _ := ioutil.ReadAll(resp.Body)
			ipAddr := strings.TrimSpace(string(body))
			ip := net.ParseIP(ipAddr)
			if ip == nil {
				errs <- fmt.Errorf("invalid ip address: %s", ipAddr)
				return
			}
			ips <- ip.String()
		}()
	}
	var errors []error
	var resps []string
	respMap := make(map[string]int)
	for (len(resps) < 2 || len(respMap) > 1) && len(resps)+len(errors) < len(ipServers) {
		select {
		case resp := <-ips:
			resps = append(resps, resp)
			respMap[resp]++
		case err := <-errs:
			errors = append(errors, err)
		}
	}
	if len(resps) == 0 {
		return "", errors[0]
	}
	// pick best ip address
	var best string
	mostVotes := 0
	for addr, votes := range respMap {
		if votes > mostVotes {
			mostVotes = votes
			best = addr
		}
	}
	return best, nil
}
