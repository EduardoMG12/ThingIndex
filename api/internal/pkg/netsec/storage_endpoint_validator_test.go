package netsec

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestStorageEndpointValidatorValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		allowPrivate bool
		endpoint     string
		lookup       LookupIPFunc
		wantErr      error
	}{
		{
			name:     "allows public ip",
			endpoint: "8.8.8.8:9000",
		},
		{
			name:     "blocks private ip",
			endpoint: "127.0.0.1:9000",
			wantErr:  errEndpointRestricted,
		},
		{
			name:     "blocks hostname resolving to private ip",
			endpoint: "minio.example.com:9000",
			lookup: func(ctx context.Context, host string) ([]net.IPAddr, error) {
				return []net.IPAddr{{IP: net.ParseIP("10.0.0.5")}}, nil
			},
			wantErr: errEndpointRestricted,
		},
		{
			name:     "allows hostname resolving to public ip",
			endpoint: "storage.example.com:9000",
			lookup: func(ctx context.Context, host string) ([]net.IPAddr, error) {
				return []net.IPAddr{{IP: net.ParseIP("34.117.59.81")}}, nil
			},
		},
		{
			name:         "allows private host when explicitly enabled",
			allowPrivate: true,
			endpoint:     "localhost:9000",
		},
		{
			name:     "rejects scheme in endpoint",
			endpoint: "https://storage.example.com",
			wantErr:  errEndpointInvalid,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			validator := NewStorageEndpointValidator(tt.allowPrivate)
			if tt.lookup != nil {
				validator.lookupIP = tt.lookup
			}

			err := validator.Validate(context.Background(), tt.endpoint)
			if tt.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr != nil && err != tt.wantErr {
				t.Fatalf("expected %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestStorageEndpointValidatorResolve(t *testing.T) {
	t.Parallel()

	validator := NewStorageEndpointValidator(false)
	validator.lookupIP = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		if host != "storage.example.com" {
			t.Fatalf("unexpected host lookup: %s", host)
		}
		return []net.IPAddr{
			{IP: net.ParseIP("2607:f8b0:4004:c07::64")},
			{IP: net.ParseIP("34.117.59.81")},
		}, nil
	}

	resolved, err := validator.Resolve(context.Background(), "storage.example.com:9000", "443")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resolved.DialAddress != "34.117.59.81:9000" {
		t.Fatalf("expected IPv4 dial target, got %s", resolved.DialAddress)
	}
	if resolved.ServerName != "storage.example.com" {
		t.Fatalf("expected server name to preserve hostname, got %s", resolved.ServerName)
	}
}

func TestStorageEndpointValidatorNewDialContext(t *testing.T) {
	t.Parallel()

	validator := NewStorageEndpointValidator(false)
	validator.lookupIP = func(ctx context.Context, host string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("34.117.59.81")}}, nil
	}

	expectedErr := errors.New("dial attempted")
	var dialedAddress string
	dialContext, err := validator.NewDialContext("storage.example.com:9000", "443", func(ctx context.Context, network, address string) (net.Conn, error) {
		dialedAddress = address
		return nil, expectedErr
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = dialContext(context.Background(), "tcp", "storage.example.com:9000")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected base dialer error, got %v", err)
	}
	if dialedAddress != "34.117.59.81:9000" {
		t.Fatalf("expected dialer to receive resolved ip, got %s", dialedAddress)
	}
}

func TestStorageEndpointValidatorNewDialContextRejectsUnexpectedAddress(t *testing.T) {
	t.Parallel()

	validator := NewStorageEndpointValidator(false)
	dialContext, err := validator.NewDialContext("storage.example.com:9000", "443", func(ctx context.Context, network, address string) (net.Conn, error) {
		t.Fatal("base dialer should not be called for unexpected target")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, err = dialContext(context.Background(), "tcp", "localhost:9000")
	if !errors.Is(err, errEndpointInvalid) {
		t.Fatalf("expected invalid endpoint error, got %v", err)
	}
}
