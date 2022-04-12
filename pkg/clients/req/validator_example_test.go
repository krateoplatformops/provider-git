package req

import (
	"context"
	"fmt"
)

func ExampleHasStatusErr() {
	err := Get().
		Url("http://example.com/404").
		CheckStatus(200).
		Do(context.Background())
	if HasStatusErr(err, 404) {
		fmt.Println("got a 404")
	}
	// Output:
	// got a 404
}
