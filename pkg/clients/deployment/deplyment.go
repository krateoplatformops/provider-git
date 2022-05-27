package deployment

import (
	"context"
	"fmt"

	"github.com/carlmjohnson/requests"
	"github.com/ghodss/yaml"
)

type Deployment struct {
	Claim   []byte
	Package []byte
}

func Get(serviceUrl, deploymentId string) (*Deployment, error) {
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

	pkg, ok := tmp["package"]
	if !ok {
		return nil, fmt.Errorf("package not found for deployment: %s", deploymentId)
	}

	ret := &Deployment{}

	ret.Claim, err = yaml.Marshal(clm)
	if err != nil {
		return nil, err
	}

	ret.Package, err = yaml.Marshal(pkg)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
