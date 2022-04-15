package clients

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/krateoplatformops/provider-git/apis/v1alpha1"
	"github.com/krateoplatformops/provider-git/pkg/clients/repo"
	"github.com/krateoplatformops/provider-git/pkg/helpers"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetConfig constructs a CreateOpts configuration that
// can be used to authenticate to the git API provider by the ReST client
func GetConfig(ctx context.Context, c client.Client, mg resource.Managed) (*repo.ProviderOpts, error) {
	switch {
	case mg.GetProviderConfigReference() != nil:
		return UseProviderConfig(ctx, c, mg)
	default:
		return nil, errors.New("providerConfigRef is not given")
	}
}

// UseProviderConfig to produce a config that can be used to create an ArgoCD client.
func UseProviderConfig(ctx context.Context, k client.Client, mg resource.Managed) (*repo.ProviderOpts, error) {
	pc := &v1alpha1.ProviderConfig{}
	err := k.Get(ctx, types.NamespacedName{Name: mg.GetProviderConfigReference().Name}, pc)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get referenced Provider")
	}

	t := resource.NewProviderConfigUsageTracker(k, &v1alpha1.ProviderConfigUsage{})
	err = t.Track(ctx, mg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot track ProviderConfig usage")
	}

	cfg := &repo.ProviderOpts{
		Verbose: helpers.IsBoolPtrEqualToBool(pc.Spec.Verbose, true),
	}

	if cfg.Verbose {
		cfg.HttpClient = &http.Client{
			Transport: &verboseTracer{http.DefaultTransport},
			Timeout:   50 * time.Second,
		}
	}

	return cfg, nil
}

// verboseTracer implements http.RoundTripper.  It prints each request and
// response/error to os.Stderr.  WARNING: this may output sensitive information
// including bearer tokens.
type verboseTracer struct {
	http.RoundTripper
}

// RoundTrip calls the nested RoundTripper while printing each request and
// response/error to os.Stderr on either side of the nested call.  WARNING: this
// may output sensitive information including bearer tokens.
func (t *verboseTracer) RoundTrip(req *http.Request) (*http.Response, error) {
	// Dump the request to os.Stderr.
	b, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		return nil, err
	}
	os.Stderr.Write(b)
	os.Stderr.Write([]byte{'\n'})

	// Call the nested RoundTripper.
	resp, err := t.RoundTripper.RoundTrip(req)

	// If an error was returned, dump it to os.Stderr.
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return resp, err
	}

	// Dump the response to os.Stderr.
	b, err = httputil.DumpResponse(resp, req.URL.Query().Get("watch") != "true")
	if err != nil {
		return nil, err
	}
	os.Stderr.Write(b)
	os.Stderr.Write([]byte{'\n'})

	return resp, err
}
