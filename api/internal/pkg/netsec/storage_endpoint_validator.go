package netsec

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var errEndpointInvalid = errors.New("storage endpoint is invalid")
var errEndpointRestricted = errors.New("storage endpoint targets a restricted address")

var restrictedNetworks = mustParseCIDRs(
	"0.0.0.0/8",
	"100.64.0.0/10",
	"127.0.0.0/8",
	"169.254.0.0/16",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"198.18.0.0/15",
	"::/128",
	"::1/128",
	"fe80::/10",
	"fc00::/7",
)

type LookupIPFunc func(ctx context.Context, host string) ([]net.IPAddr, error)

type StorageEndpointValidator struct {
	allowPrivate bool
	lookupIP     LookupIPFunc
}

func NewStorageEndpointValidator(allowPrivate bool) *StorageEndpointValidator {
	return &StorageEndpointValidator{
		allowPrivate: allowPrivate,
		lookupIP:     net.DefaultResolver.LookupIPAddr,
	}
}

func (v *StorageEndpointValidator) Validate(ctx context.Context, endpoint string) error {
	host, err := endpointHost(endpoint)
	if err != nil {
		return errEndpointInvalid
	}

	if v.allowPrivate {
		return nil
	}

	if ip := net.ParseIP(host); ip != nil {
		if isRestrictedIP(ip) {
			return errEndpointRestricted
		}
		return nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	addresses, err := v.lookupIP(lookupCtx, host)
	if err != nil || len(addresses) == 0 {
		return errEndpointInvalid
	}

	for _, address := range addresses {
		if isRestrictedIP(address.IP) {
			return errEndpointRestricted
		}
	}

	return nil
}

func endpointHost(endpoint string) (string, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return "", errEndpointInvalid
	}
	if strings.Contains(trimmed, "://") {
		return "", errEndpointInvalid
	}
	if ip := net.ParseIP(trimmed); ip != nil {
		return ip.String(), nil
	}

	parsed, err := url.Parse("//" + trimmed)
	if err != nil {
		return "", errEndpointInvalid
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errEndpointInvalid
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errEndpointInvalid
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return "", errEndpointInvalid
	}

	if port := parsed.Port(); port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return "", errEndpointInvalid
		}
	}

	return host, nil
}

func isRestrictedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsInterfaceLocalMulticast() || ip.IsUnspecified() {
		return true
	}

	for _, network := range restrictedNetworks {
		if network.Contains(ip) {
			return true
		}
	}

	return false
}

func mustParseCIDRs(cidrs ...string) []*net.IPNet {
	networks := make([]*net.IPNet, 0, len(cidrs))
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		networks = append(networks, network)
	}

	return networks
}
