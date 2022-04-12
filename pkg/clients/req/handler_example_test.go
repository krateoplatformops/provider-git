package req

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
)

func ExampleToBufioReader() {
	// read a response line by line for a sentinel
	found := false
	err := Get().
		Url("http://example.com").
		Handle(ToBufioReader(func(r *bufio.Reader) error {
			var err error
			for s := ""; err == nil; {
				if strings.Contains(s, "Example Domain") {
					found = true
					return nil
				}
				// read one line from response
				s, err = r.ReadString('\n')
			}
			if err == io.EOF {
				return nil
			}
			return err
		})).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to example.com:", err)
	}
	fmt.Println(found)
	// Output:
	// true
}

func ExampleToBufioScanner() {
	// read a response line by line for a sentinel
	found := false
	needle := []byte("Example Domain")
	err := Get().
		Url("http://example.com").
		Handle(ToBufioScanner(func(s *bufio.Scanner) error {
			// read one line at time from response
			for s.Scan() {
				if bytes.Contains(s.Bytes(), needle) {
					found = true
					return nil
				}
			}
			return s.Err()
		})).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to example.com:", err)
	}
	fmt.Println(found)
	// Output:
	// true
}
