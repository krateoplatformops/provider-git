package req

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
)

// DefaultValidator is the validator applied by Builder unless otherwise specified.
var DefaultValidator ResponseHandler = CheckStatus(
	http.StatusOK,
	http.StatusCreated,
	http.StatusAccepted,
	http.StatusNonAuthoritativeInfo,
	http.StatusNoContent,
)

// ResponseError is the error type produced by CheckStatus and CheckContentType.
type ResponseError http.Response

// Error fulfills the error interface.
func (se *ResponseError) Error() string {
	return fmt.Sprintf("response error for %s", se.Request.URL.Redacted())
}

// CheckStatus validates the response has an acceptable status code.
func CheckStatus(acceptStatuses ...int) ResponseHandler {
	return func(res *http.Response) error {
		for _, code := range acceptStatuses {
			if res.StatusCode == code {
				return nil
			}
		}

		return fmt.Errorf("%w: unexpected status: %d",
			(*ResponseError)(res), res.StatusCode)
	}
}

type StatusError struct {
	Code  int
	Inner error
}

func (e StatusError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("unexpected status: %d: %v", e.Code, e.Inner)
	}
	return fmt.Sprintf("unexpected status: %d:", e.Code)
}

func (e StatusError) Unwrap() error {
	return e.Inner
}

// ErrorJSON validates the response has an acceptable status
// code and if it's bad, attempts to marshal the JSON
// into the error object provided.
func ErrorJSON(v error, acceptStatuses ...int) ResponseHandler {
	return func(res *http.Response) error {
		for _, code := range acceptStatuses {
			if res.StatusCode == code {
				return nil
			}
		}

		if res.Body == nil {
			return StatusError{Code: res.StatusCode} //fmt.Errorf("%w: unexpected status: %d", (*ResponseError)(res), res.StatusCode)
		}

		data, err := io.ReadAll(res.Body)
		if err != nil {
			return StatusError{Code: res.StatusCode, Inner: err}
		}

		if err = json.Unmarshal(data, &v); err != nil {
			return StatusError{Code: res.StatusCode, Inner: err}
		}

		return StatusError{Code: res.StatusCode, Inner: v}
	}
}

// HasStatusErr returns true if err is a ResponseError caused by any of the codes given.
func HasStatusErr(err error, codes ...int) bool {
	if err == nil {
		return false
	}
	if se := new(ResponseError); errors.As(err, &se) {
		for _, code := range codes {
			if se.StatusCode == code {
				return true
			}
		}
	}
	return false
}

// CheckContentType validates that a response has one of the given content type headers.
func CheckContentType(cts ...string) ResponseHandler {
	return func(res *http.Response) error {
		mt, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
		if err != nil {
			return fmt.Errorf("%w: problem matching Content-Type",
				(*ResponseError)(res))
		}
		for _, ct := range cts {
			if mt == ct {
				return nil
			}
		}
		return fmt.Errorf("%w: unexpected Content-Type: %s",
			(*ResponseError)(res), mt)
	}
}

type bufioCloser struct {
	*bufio.Reader
	io.Closer
}

// CheckPeek wraps the body of a response in a bufio.Reader and
// gives f a peek at the first n bytes for validation.
func CheckPeek(n int, f func([]byte) error) ResponseHandler {
	return func(res *http.Response) error {
		// ensure buffer is at least minimum size
		buf := bufio.NewReader(res.Body)
		// ensure large peeks will fit in the buffer
		buf = bufio.NewReaderSize(buf, n)
		res.Body = &bufioCloser{
			buf,
			res.Body,
		}
		b, err := buf.Peek(n)
		if err != nil && err != io.EOF {
			return err
		}
		return f(b)
	}
}
