package config

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestParseSecretRef(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    secretRef
		wantErr bool
	}{
		{
			name:  "valid ref",
			input: "my-namespace/my-secret",
			want:  secretRef{Namespace: "my-namespace", Name: "my-secret"},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "no slash",
			input:   "just-a-name",
			wantErr: true,
		},
		{
			name:    "empty namespace",
			input:   "/my-secret",
			wantErr: true,
		},
		{
			name:    "empty name",
			input:   "my-namespace/",
			wantErr: true,
		},
		{
			name:  "extra slashes in name",
			input: "ns/name/with/slashes",
			want:  secretRef{Namespace: "ns", Name: "name/with/slashes"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSecretRef(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadFromClient_Success(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cloudflare-creds",
			Namespace: "infra",
		},
		Data: map[string][]byte{
			"api-token": []byte("my-test-token"),
		},
	}

	client := fake.NewSimpleClientset(secret)
	ref := secretRef{Namespace: "infra", Name: "cloudflare-creds"}

	cfg, err := loadFromClient(context.Background(), client, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIToken != "my-test-token" {
		t.Errorf("got token %q, want %q", cfg.APIToken, "my-test-token")
	}
}

func TestLoadFromClient_TokenTrimmed(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "creds",
			Namespace: "ns",
		},
		Data: map[string][]byte{
			"api-token": []byte("  token-with-whitespace \n"),
		},
	}

	client := fake.NewSimpleClientset(secret)
	cfg, err := loadFromClient(context.Background(), client, secretRef{Namespace: "ns", Name: "creds"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIToken != "token-with-whitespace" {
		t.Errorf("got token %q, want %q", cfg.APIToken, "token-with-whitespace")
	}
}

func TestLoadFromClient_SecretNotFound(t *testing.T) {
	client := fake.NewSimpleClientset() // no secrets
	ref := secretRef{Namespace: "ns", Name: "missing"}

	_, err := loadFromClient(context.Background(), client, ref)
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
	if !strings.Contains(err.Error(), "ns/missing") {
		t.Errorf("error should mention secret ref, got: %v", err)
	}
}

func TestLoadFromClient_MissingAPITokenKey(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "creds",
			Namespace: "ns",
		},
		Data: map[string][]byte{
			"wrong-key": []byte("some-value"),
		},
	}

	client := fake.NewSimpleClientset(secret)
	ref := secretRef{Namespace: "ns", Name: "creds"}

	_, err := loadFromClient(context.Background(), client, ref)
	if err == nil {
		t.Fatal("expected error for missing api-token key, got nil")
	}
	if !strings.Contains(err.Error(), "api-token") {
		t.Errorf("error should mention missing key name, got: %v", err)
	}
}

func TestLoadFromClient_EmptyToken(t *testing.T) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "creds",
			Namespace: "ns",
		},
		Data: map[string][]byte{
			"api-token": []byte("   "),
		},
	}

	client := fake.NewSimpleClientset(secret)
	ref := secretRef{Namespace: "ns", Name: "creds"}

	_, err := loadFromClient(context.Background(), client, ref)
	if err == nil {
		t.Fatal("expected error for empty api-token value, got nil")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention empty value, got: %v", err)
	}
}
