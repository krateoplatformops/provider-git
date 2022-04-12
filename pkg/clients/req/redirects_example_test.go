package req

import (
	"context"
	"fmt"
	"net/http"
)

func ExampleCheckRedirectPolicy() {
	cl := *http.DefaultClient
	cl.CheckRedirect = NoFollow

	if err := Get().Url("https://httpbingo.org/redirect/1").
		Client(&cl).
		CheckStatus(http.StatusFound).
		Handle(func(res *http.Response) error {
			fmt.Println("Status", res.Status)
			fmt.Println("From", res.Request.URL)
			fmt.Println("To", res.Header.Get("Location"))
			return nil
		}).
		Do(context.Background()); err != nil {
		panic(err)
	}
	// Output:
	// Status 302 Found
	// From https://httpbingo.org/redirect/1
	// To /get
}

func ExampleMaxFollow() {
	cl := *http.DefaultClient
	cl.CheckRedirect = MaxFollow(1)

	if err := Get().Url("https://httpbingo.org/redirect/2").
		Client(&cl).
		CheckStatus(http.StatusFound).
		Handle(func(res *http.Response) error {
			fmt.Println("Status", res.Status)
			fmt.Println("From", res.Request.URL)
			fmt.Println("To", res.Header.Get("Location"))
			return nil
		}).
		Do(context.Background()); err != nil {
		panic(err)
	}
	// Output:
	// Status 302 Found
	// From https://httpbingo.org/relative-redirect/1
	// To /get
}
