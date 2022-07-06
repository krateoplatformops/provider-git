package deployment

import (
	"context"
	"fmt"

	"github.com/carlmjohnson/requests"
	"github.com/ghodss/yaml"
)

func Get(serviceUrl, deploymentId string) ([]byte, error) {
	tmp := map[string]any{}

	err := requests.
		URL(serviceUrl).Path(deploymentId).
		ToJSON(&tmp).
		CheckStatus(200).
		Fetch(context.Background())
	if err != nil {
		return nil, err
	}

	clm, ok := tmp["claim"]
	if !ok {
		return nil, fmt.Errorf("claim not found for deployment: %s", deploymentId)
	}

	return yaml.Marshal(clm)
}
