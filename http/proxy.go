package http

import (
	"net"
)

// SetProxyTrust configures how ClientIP treats forwarded headers.
func (c *Context) SetProxyTrust(trustAll bool, proxies []*net.IPNet) {
	c.trustAllProxies = trustAll
	c.trustedProxies = proxies
}

func (c *Context) remoteIP() string {
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return ip
}

func (c *Context) isTrustedProxy(ip string) bool {
	if c.trustAllProxies {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range c.trustedProxies {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}
