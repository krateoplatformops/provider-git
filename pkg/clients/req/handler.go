package req

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// ResponseHandler is used to validate or handle the response to a request.
type ResponseHandler func(*http.Response) error

// ChainHandlers allows for the composing of validators or response handlers.
func ChainHandlers(handlers ...ResponseHandler) ResponseHandler {
	return func(r *http.Response) error {
		for _, h := range handlers {
			if h == nil {
				continue
			}
			if err := h(r); err != nil {
				return err
			}
		}
		return nil
	}
}

func consumeBody(res *http.Response) (err error) {
	const maxDiscardSize = 640 * 1 << 10
	if _, err = io.CopyN(io.Discard, res.Body, maxDiscardSize); err == io.EOF {
		err = nil
	}
	return err
}

// ToJSON decodes a response as a JSON object.
func ToJSON(v interface{}) ResponseHandler {
	return func(res *http.Response) error {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if err = json.Unmarshal(data, v); err != nil {
			return err
		}
		return nil
	}
}

// ToString writes the response body to the provided string pointer.
func ToString(sp *string) ResponseHandler {
	return func(res *http.Response) error {
		var buf strings.Builder
		_, err := io.Copy(&buf, res.Body)
		if err == nil {
			*sp = buf.String()
		}
		return err
	}
}

// ToBytesBuffer writes the response body to the provided bytes.Buffer.
func ToBytesBuffer(buf *bytes.Buffer) ResponseHandler {
	return func(res *http.Response) error {
		_, err := io.Copy(buf, res.Body)
		return err
	}
}

// ToBufioReader takes a callback which wraps the response body in a bufio.Reader.
func ToBufioReader(f func(r *bufio.Reader) error) ResponseHandler {
	return func(res *http.Response) error {
		return f(bufio.NewReader(res.Body))
	}
}

// ToBufioScanner takes a callback which wraps the response body in a bufio.Scanner.
func ToBufioScanner(f func(r *bufio.Scanner) error) ResponseHandler {
	return func(res *http.Response) error {
		return f(bufio.NewScanner(res.Body))
	}
}

// ToWriter copies the response body to w.
func ToWriter(w io.Writer) ResponseHandler {
	return ToBufioReader(func(r *bufio.Reader) error {
		_, err := io.Copy(w, r)

		return err
	})
}

// ToHeaders copies the response headers to h.
func ToHeaders(h map[string][]string) ResponseHandler {
	return func(res *http.Response) error {
		for k, v := range res.Header {
			h[k] = v
		}

		return nil
	}
}
