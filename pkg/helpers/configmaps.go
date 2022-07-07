package helpers

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// A ConfigMapReference is a reference to a configMap in an arbitrary namespace.
type ConfigMapReference struct {
	// Name of the configmap.
	Name string `json:"name"`

	// Namespace of the configmap.
	Namespace string `json:"namespace"`
}

// A ConfigMapKeySelector is a reference to a configmap key in an arbitrary namespace.
type ConfigMapKeySelector struct {
	ConfigMapReference `json:",inline"`

	// The key to select.
	Key string `json:"key"`
}

func GetConfigMapValue(ctx context.Context, kube client.Client, ref *ConfigMapKeySelector) (string, error) {
	if ref == nil {
		return "", errors.New("no configmap referenced")
	}

	cm := &corev1.ConfigMap{}
	err := kube.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, cm)
	if err != nil {
		return "", errors.Wrapf(err, "cannot get %s configmap", ref.Name)
	}

	return string(cm.Data[ref.Key]), nil
}
