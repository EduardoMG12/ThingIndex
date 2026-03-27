package netsec

import (
	"context"
	"errors"
	"fmt"
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
type DialContextFunc func(ctx context.Context, network, address string) (net.Conn, error)

type ResolvedStorageEndpoint struct {
	DialAddress string
	ServerName  string
}

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
	_, _, err := v.lookupEndpointAddresses(ctx, endpoint)
	return err
}

func (v *StorageEndpointValidator) Resolve(ctx context.Context, endpoint, defaultPort string) (*ResolvedStorageEndpoint, error) {
	target, addresses, err := v.lookupEndpointAddresses(ctx, endpoint)
	if err != nil {
		return nil, err
	}

	port, err := target.portOrDefault(defaultPort)
	if err != nil {
		return nil, errEndpointInvalid
	}

	ip := selectDialIP(addresses)
	if ip == nil {
		return nil, errEndpointInvalid
	}

	resolved := &ResolvedStorageEndpoint{
		DialAddress: net.JoinHostPort(ip.String(), port),
	}
	if net.ParseIP(target.host) == nil {
		resolved.ServerName = target.host
	}

	return resolved, nil
}

func (v *StorageEndpointValidator) NewDialContext(endpoint, defaultPort string, baseDialContext DialContextFunc) (DialContextFunc, error) {
	target, err := parseEndpointTarget(endpoint)
	if err != nil {
		return nil, errEndpointInvalid
	}
	port, err := target.portOrDefault(defaultPort)
	if err != nil {
		return nil, errEndpointInvalid
	}

	if baseDialContext == nil {
		return nil, errors.New("base dial context is required")
	}

	return func(ctx context.Context, network, address string) (net.Conn, error) {
		if err := target.matchesDialAddress(address, port); err != nil {
			return nil, err
		}

		resolved, err := v.Resolve(ctx, endpoint, port)
		if err != nil {
			return nil, err
		}

		return baseDialContext(ctx, network, resolved.DialAddress)
	}, nil
}

type endpointTarget struct {
	host string
	port string
}

func parseEndpointTarget(endpoint string) (endpointTarget, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return endpointTarget{}, errEndpointInvalid
	}
	if strings.Contains(trimmed, "://") {
		return endpointTarget{}, errEndpointInvalid
	}
	if ip := net.ParseIP(trimmed); ip != nil {
		return endpointTarget{host: ip.String()}, nil
	}

	parsed, err := url.Parse("//" + trimmed)
	if err != nil {
		return endpointTarget{}, errEndpointInvalid
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return endpointTarget{}, errEndpointInvalid
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return endpointTarget{}, errEndpointInvalid
	}

	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return endpointTarget{}, errEndpointInvalid
	}

	port := parsed.Port()
	if port != "" {
		if _, err := strconv.ParseUint(port, 10, 16); err != nil {
			return endpointTarget{}, errEndpointInvalid
		}
	}

	return endpointTarget{host: host, port: port}, nil
}

func (t endpointTarget) portOrDefault(defaultPort string) (string, error) {
	port := strings.TrimSpace(t.port)
	if port == "" {
		port = strings.TrimSpace(defaultPort)
	}
	if port == "" {
		return "", errEndpointInvalid
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", errEndpointInvalid
	}

	return port, nil
}

func (t endpointTarget) matchesDialAddress(address, defaultPort string) error {
	host, port, err := net.SplitHostPort(strings.TrimSpace(address))
	if err != nil {
		return fmt.Errorf("%w: unexpected storage dial target", errEndpointInvalid)
	}

	expectedPort, err := t.portOrDefault(defaultPort)
	if err != nil {
		return err
	}
	if port != expectedPort {
		return fmt.Errorf("%w: unexpected storage dial target", errEndpointInvalid)
	}

	if !strings.EqualFold(strings.TrimSpace(host), strings.TrimSpace(t.host)) {
		return fmt.Errorf("%w: unexpected storage dial target", errEndpointInvalid)
	}

	return nil
}

func (v *StorageEndpointValidator) lookupEndpointAddresses(ctx context.Context, endpoint string) (endpointTarget, []net.IPAddr, error) {
	target, err := parseEndpointTarget(endpoint)
	if err != nil {
		return endpointTarget{}, nil, errEndpointInvalid
	}

	if ip := net.ParseIP(target.host); ip != nil {
		if !v.allowPrivate && isRestrictedIP(ip) {
			return endpointTarget{}, nil, errEndpointRestricted
		}
		return target, []net.IPAddr{{IP: ip}}, nil
	}

	lookupCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	addresses, err := v.lookupIP(lookupCtx, target.host)
	if err != nil || len(addresses) == 0 {
		return endpointTarget{}, nil, errEndpointInvalid
	}

	if !v.allowPrivate {
		for _, address := range addresses {
			if isRestrictedIP(address.IP) {
				return endpointTarget{}, nil, errEndpointRestricted
			}
		}
	}

	return target, addresses, nil
}

func selectDialIP(addresses []net.IPAddr) net.IP {
	for _, address := range addresses {
		if ip := address.IP.To4(); ip != nil {
			return ip
		}
	}
	for _, address := range addresses {
		if address.IP != nil {
			return address.IP
		}
	}

	return nil
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
