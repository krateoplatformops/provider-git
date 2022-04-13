package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/krateoplatformops/provider-git/pkg/clients/gitrepo"
)

// Tracer implements http.RoundTripper.  It prints each request and
// response/error to os.Stderr.  WARNING: this may output sensitive information
// including bearer tokens.
type Tracer struct {
	http.RoundTripper
}

// RoundTrip calls the nested RoundTripper while printing each request and
// response/error to os.Stderr on either side of the nested call.  WARNING: this
// may output sensitive information including bearer tokens.
func (t *Tracer) RoundTrip(req *http.Request) (*http.Response, error) {
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

func main() {

	token := "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"
	fromRepoUrl := "https://github.com/projectkerberus/aws-stack-template"
	toRepoUrl := "https://github.com/lucasepe/deletami"

	log.Printf("creating repository: %s", toRepoUrl)
	err := gitrepo.CreateEventually(toRepoUrl, &gitrepo.CreateOptions{
		Token:   token,
		Private: true,
	})
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	/*fromRepo*/
	log.Printf("cloning repository: %s", fromRepoUrl)
	fromRepo, err := gitrepo.Clone(fromRepoUrl, gitrepo.GitToken(token))
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	log.Printf("cloning repository: %s", toRepoUrl)
	toRepo, err := gitrepo.Clone(toRepoUrl, gitrepo.GitToken(token))
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	log.Printf("create branch 'main' on: %s\n", toRepoUrl)
	err = toRepo.Branch("main")
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	log.Printf("copying dir 'skeleton' from: %s to: %s...", fromRepoUrl, toRepoUrl)
	err = gitrepo.CopyDir(fromRepo, toRepo, "skeleton", ".")
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	log.Printf("commit files")
	err = toRepo.Commit(".", ":tada: first commit")
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	log.Printf("push files to: %s", toRepoUrl)
	err = toRepo.Push("origin", "main")
	if err != nil {
		fmt.Printf("==> %v\n", err)
		os.Exit(1)
	}

	/*
		hc := &http.Client{
			Transport: &Tracer{http.DefaultTransport},
			Timeout:   40 * time.Second,
		}
	*/
}
