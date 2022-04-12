package req

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

func ExampleReplayString() {
	const res = `HTTP/1.1 200 OK

An example response.`

	var s string
	const expected = `An example response.`
	if err := Get().
		Url("http://response.example").
		Client(&http.Client{
			Transport: ReplayString(res),
		}).
		ToString(&s).
		Do(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println(s == expected)
	// Output:
	// true
}

func ExamplePermitURLTransport() {
	// Wrap an existing transport or use nil for http.DefaultTransport
	baseTrans := http.DefaultClient.Transport
	trans := PermitURLTransport(baseTrans, `^http://example\.com/?`)
	var s string
	if err := Get().
		Url("http://example.com").
		Transport(trans).
		ToString(&s).
		Do(context.Background()); err != nil {
		panic(err)
	}
	fmt.Println(strings.Contains(s, "Example Domain"))

	if err := Get().
		Url("http://unauthorized.example.com").
		Transport(trans).
		ToString(&s).
		Do(context.Background()); err != nil {
		fmt.Println(err) // unauthorized subdomain not allowed
	}
	// Output:
	// true
	// Get "http://unauthorized.example.com": requested URL not permitted by regexp: ^http://example\.com/?
}

func ExampleRoundTripFunc() {
	// Wrap an underlying transport in order to add request middleware
	var logTripper RoundTripFunc = func(req *http.Request) (res *http.Response, err error) {
		fmt.Printf("req [%s] %s\n", req.Method, req.URL)
		res, err = http.DefaultClient.Transport.RoundTrip(req)
		if err != nil {
			fmt.Printf("res [error] %s %s\n", err, req.URL)
		} else {
			fmt.Printf("res [%s] %s\n", res.Status, req.URL)
		}
		return
	}
	err := Get().
		Url("http://example.com").
		Transport(logTripper).
		Do(context.Background())
	if err != nil {
		fmt.Println("something went wrong:", err)
	}
	// Output:
	// req [GET] http://example.com
	// res [200 OK] http://example.com
}
