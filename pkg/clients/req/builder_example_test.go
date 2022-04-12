package req

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	http.DefaultClient.Transport = Replay("testdata")
}

func Example() {
	// Simple GET into a string
	var s string
	err := Get().
		Url("http://example.com").
		ToString(&s).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to example.com:", err)
	}
	fmt.Println(strings.Contains(s, "Example Domain"))
	// Output:
	// true
}

func Example_getJSON() {
	// GET a JSON object
	id := 1
	var post placeholder
	err := Get().
		Url("https://jsonplaceholder.typicode.com").
		Pathf("/posts/%d", id).
		ToJSON(&post).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to jsonplaceholder.typicode.com:", err)
	}
	fmt.Println(post.Title)
	// Output:
	// sunt aut facere repellat provident occaecati excepturi optio reprehenderit
}

func Example_postJSON() {
	// POST a JSON object and parse the response
	var res placeholder
	req := placeholder{
		Title:  "foo",
		Body:   "baz",
		UserID: 1,
	}
	err := Post().
		Url("/posts").
		Host("jsonplaceholder.typicode.com").
		BodyJSON(&req).
		ToJSON(&res).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to jsonplaceholder.typicode.com:", err)
	}
	fmt.Println(res)
	// Output:
	// {101 foo baz 1}
}

func ExampleBuilder_ToBytesBuffer() {
	// Simple GET into a buffer
	var buf bytes.Buffer
	err := Get().
		Url("http://example.com").
		ToBytesBuffer(&buf).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to example.com:", err)
	}
	fmt.Println(strings.Contains(buf.String(), "Example Domain"))
	// Output:
	// true
}

func ExampleBuilder_ToWriter() {
	f, err := os.CreateTemp("", "*.to_writer.html")
	if err != nil {
		log.Fatal(err)
	}
	defer os.Remove(f.Name()) // clean up

	// suppose there is some io.Writer you want to stream to
	err = Get().
		Url("http://example.com").
		ToWriter(f).
		Do(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	if err = f.Close(); err != nil {
		log.Fatal(err)
	}
	stat, err := os.Stat(f.Name())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("file is %d bytes\n", stat.Size())

	// Output:
	// file is 1256 bytes
}

type placeholder struct {
	ID     int    `json:"id,omitempty"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	UserID int    `json:"userId"`
}

func ExampleBuilder_CheckStatus() {
	// Expect a specific status code
	err := Get().
		Url("https://jsonplaceholder.typicode.com").
		Pathf("/posts/%d", 9001).
		CheckStatus(404).
		CheckContentType("application/json").
		Do(context.Background())
	if err != nil {
		fmt.Println("should be a 404:", err)
	} else {
		fmt.Println("OK")
	}
	// Output:
	// OK
}

func ExampleBuilder_CheckContentType() {
	// Expect a specific status code
	err := Get().
		Url("https://jsonplaceholder.typicode.com").
		Pathf("/posts/%d", 1).
		CheckContentType("application/bison").
		Do(context.Background())
	if err != nil {
		if re := new(ResponseError); errors.As(err, &re) {
			fmt.Println("content-type was", re.Header.Get("Content-Type"))
		}
	}
	// Output:
	// content-type was application/json; charset=utf-8
}

// Examples with the Postman echo server
type postman struct {
	Args    map[string]string `json:"args"`
	Data    string            `json:"data"`
	Headers map[string]string `json:"headers"`
	JSON    map[string]string `json:"json"`
}

func Example_queryParam() {
	c := 4
	// Set a query parameter
	var params postman
	err := Get().
		Url("https://postman-echo.com/get?a=1&b=2").
		Param("b", "3").
		ParamInt("c", c).
		ToJSON(&params).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Println(params.Args)
	// Output:
	// map[a:1 b:3 c:4]
}

func ExampleBuilder_Header() {
	// Set headers
	var headers postman
	err := Get().
		Url("https://postman-echo.com/get").
		UserAgent("bond/james-bond").
		BasicAuth("bondj", "007!").
		ContentType("secret").
		Header("martini", "shaken").
		ToJSON(&headers).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Println(headers.Headers["user-agent"])
	fmt.Println(headers.Headers["authorization"])
	fmt.Println(headers.Headers["content-type"])
	fmt.Println(headers.Headers["martini"])
	// Output:
	// bond/james-bond
	// Basic Ym9uZGo6MDA3IQ==
	// secret
	// shaken
}

func ExampleBuilder_Bearer() {
	// We get a 401 response if no bearer token is provided
	err := Get().
		Url("http://httpbin.org/bearer").
		CheckStatus(http.StatusUnauthorized).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with httpbin:", err)
	}
	// But our response is accepted when we provide a bearer token
	var res struct {
		Authenticated bool
		Token         string
	}
	err = Get().
		Url("http://httpbin.org/bearer").
		Bearer("whatever").
		ToJSON(&res).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with httpbin:", err)
	}
	fmt.Println(res.Authenticated)
	fmt.Println(res.Token)
	// Output:
	// true
	// whatever
}

func ExampleBuilder_BodyBytes() {
	// Post a raw body
	var data postman
	err := Post().
		Url("https://postman-echo.com/post").
		BodyBytes([]byte(`hello, world`)).
		ContentType("text/plain").
		ToJSON(&data).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Println(data.Data)
	// Output:
	// hello, world
}

func ExampleBuilder_BodyReader() {
	// temp file creation boilerplate
	dir, err := os.MkdirTemp("", "body_reader_*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	exampleFilename := filepath.Join(dir, "example.txt")
	exampleContent := `hello, world`
	if err := os.WriteFile(exampleFilename, []byte(exampleContent), 0644); err != nil {
		log.Fatal(err)
	}

	// suppose there is some io.Reader you want to stream from
	f, err := os.Open(exampleFilename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// send the raw file to server
	var echo postman
	err = Post().
		Url("https://postman-echo.com/post").
		ContentType("text/plain").
		BodyReader(f).
		ToJSON(&echo).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Println(echo.Data)
	// Output:
	// hello, world
}

func ExampleBuilder_Config() {
	var echo postman
	err := Post().
		Url("https://postman-echo.com/post").
		ContentType("text/plain").
		Config(GzipConfig(
			gzip.DefaultCompression,
			func(gw *gzip.Writer) error {
				_, err := gw.Write([]byte(`hello, world`))
				return err
			})).
		ToJSON(&echo).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Println(echo.Data)
	// Output:
	// hello, world
}

func ExampleBuilder_BodyWriter() {
	var echo postman
	err := Post().
		Url("https://postman-echo.com/post").
		ContentType("text/plain").
		BodyWriter(func(w io.Writer) error {
			cw := csv.NewWriter(w)
			cw.Write([]string{"col1", "col2"})
			cw.Write([]string{"val1", "val2"})
			cw.Flush()
			return cw.Error()
		}).
		ToJSON(&echo).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Printf("%q\n", echo.Data)
	// Output:
	// "col1,col2\nval1,val2\n"
}

func ExampleBuilder_BodyForm() {
	// Submit form values
	var echo postman
	err := Put().
		Url("https://postman-echo.com/put").
		BodyForm(url.Values{
			"hello": []string{"world"},
		}).
		ToJSON(&echo).
		Do(context.Background())
	if err != nil {
		fmt.Println("problem with postman:", err)
	}
	fmt.Println(echo.JSON)
	// Output:
	// map[hello:world]
}

func ExampleBuilder_CheckPeek() {
	// Check that a response has a doctype
	const doctype = "<!doctype html>"
	var s string
	err := Get().
		Url("http://example.com").
		CheckPeek(len(doctype), func(b []byte) error {
			if string(b) != doctype {
				return fmt.Errorf("missing doctype: %q", b)
			}
			return nil
		}).
		ToString(&s).
		Do(context.Background())
	if err != nil {
		fmt.Println("could not connect to example.com:", err)
	}
	fmt.Println(
		// Final result still has the prefix
		strings.HasPrefix(s, doctype),
		// And the full body
		strings.HasSuffix(s, "</html>\n"),
	)
	// Output:
	// true true
}

func ExampleBuilder_Transport() {
	const text = "Hello, from transport!"
	var myCustomTransport RoundTripFunc = func(req *http.Request) (res *http.Response, err error) {
		res = &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(text)),
		}
		return
	}
	var s string
	err := Get().
		Url("x://transport.example").
		Transport(myCustomTransport).
		ToString(&s).
		Do(context.Background())
	if err != nil {
		fmt.Println("transport failed:", err)
	}
	fmt.Println(s == text) // true
	// Output:
	// true
}
