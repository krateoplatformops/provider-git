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

func GetConfigMapData(ctx context.Context, k client.Client, ref *ConfigMapReference) (map[string]string, error) {
	if ref == nil {
		return nil, errors.New("no configmap referenced")
	}

	cm := &corev1.ConfigMap{}
	err := k.Get(ctx, types.NamespacedName{Namespace: ref.Namespace, Name: ref.Name}, cm)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get %s configmap", ref.Name)
	}

	return cm.Data, nil
}
