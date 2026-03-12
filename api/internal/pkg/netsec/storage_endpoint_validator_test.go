package netsec

import (
	"context"
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
