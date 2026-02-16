// Package config handles loading Cloudflare credentials from Kubernetes secrets.
package config

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Config holds the Cloudflare API credentials.
type Config struct {
	APIToken string
}

// secretRef holds the parsed namespace and name of a Kubernetes secret.
type secretRef struct {
	Namespace string
	Name      string
}

// parseSecretRef parses a "namespace/secret-name" string into its parts.
func parseSecretRef(ref string) (secretRef, error) {
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return secretRef{}, fmt.Errorf("invalid --secret value %q: expected namespace/secret-name", ref)
	}
	return secretRef{Namespace: parts[0], Name: parts[1]}, nil
}

// buildKubeClient creates a Kubernetes clientset from the given kubeconfig path.
// If kubeconfig is empty, it falls back to in-cluster config.
func buildKubeClient(kubeconfig string) (kubernetes.Interface, error) {
	var cfg *rest.Config
	var err error

	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Try loading from default kubeconfig location, fall back to in-cluster.
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		configOverrides := &clientcmd.ConfigOverrides{}
		cfg, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, configOverrides).ClientConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("building kubernetes config: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}
	return client, nil
}

// Load reads the Cloudflare API token from a Kubernetes secret.
//
// secretFlag is the --secret flag value in "namespace/secret-name" format.
// kubeconfig is an optional path to a kubeconfig file (empty uses the default).
func Load(ctx context.Context, secretFlag string, kubeconfig string) (*Config, error) {
	ref, err := parseSecretRef(secretFlag)
	if err != nil {
		return nil, err
	}

	client, err := buildKubeClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	return loadFromClient(ctx, client, ref)
}

// loadFromClient fetches the secret using the provided Kubernetes client.
// Separated from Load to allow testing with a fake clientset.
func loadFromClient(ctx context.Context, client kubernetes.Interface, ref secretRef) (*Config, error) {
	secret, err := client.CoreV1().Secrets(ref.Namespace).Get(ctx, ref.Name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("fetching secret %s/%s: %w", ref.Namespace, ref.Name, err)
	}

	token, ok := secret.Data["cloudflare_api_token"]
	if !ok {
		return nil, fmt.Errorf("secret %s/%s does not contain key \"cloudflare_api_token\"", ref.Namespace, ref.Name)
	}

	tokenStr := strings.TrimSpace(string(token))
	if tokenStr == "" {
		return nil, fmt.Errorf("secret %s/%s has an empty \"cloudflare_api_token\" value", ref.Namespace, ref.Name)
	}

	return &Config{APIToken: tokenStr}, nil
}
