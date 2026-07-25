// Package kube builds Kubernetes API clients for the in-cluster discovery path.
package kube

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DynamicClient returns a dynamic Kubernetes client. When context is empty it
// first tries the in-cluster service-account config (for a pod deployment) and
// falls back to the local kubeconfig; when context is set it selects that
// context from the local kubeconfig (respecting $KUBECONFIG / ~/.kube/config).
func DynamicClient(context string) (dynamic.Interface, error) {
	cfg, err := restConfig(context)
	if err != nil {
		return nil, err
	}
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("kube: build dynamic client: %w", err)
	}
	return client, nil
}

func restConfig(context string) (*rest.Config, error) {
	if context == "" {
		if c, err := rest.InClusterConfig(); err == nil {
			return c, nil
		}
	}
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	overrides := &clientcmd.ConfigOverrides{}
	if context != "" {
		overrides.CurrentContext = context
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, overrides).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("kube: load kubeconfig: %w", err)
	}
	return cfg, nil
}
