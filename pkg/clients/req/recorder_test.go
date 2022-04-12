package req

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestRecordReplay(t *testing.T) {
	dir := t.TempDir()

	var s1, s2 string
	err := Get().Url("http://example.com").
		Transport(Record(http.DefaultTransport, dir)).
		ToString(&s1).
		Do(context.Background())
	if err != nil {
		log.Fatalln("unexpected error:", err)
	}

	err = Get().Url("http://example.com").
		Transport(Replay(dir)).
		ToString(&s2).
		Do(context.Background())
	if err != nil {
		log.Fatalln("unexpected error:", err)
	}
	if s1 != s2 {
		log.Fatalf("%q != %q", s1, s2)
	}
}

func TestCaching(t *testing.T) {
	dir := t.TempDir()
	hasRun := false
	content := "some content"
	var onceTrans RoundTripFunc = func(req *http.Request) (res *http.Response, err error) {
		if hasRun {
			t.Fatal("ran twice")
		}
		hasRun = true
		res = &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(content)),
		}
		return
	}
	trans := Caching(onceTrans, dir)
	var s1, s2 string
	err := Get().Url("http://example.com").
		Transport(trans).
		ToString(&s1).
		Do(context.Background())
	if err != nil {
		log.Fatalln("unexpected error:", err)
	}
	err = Get().Url("http://example.com").
		Transport(trans).
		ToString(&s2).
		Do(context.Background())
	if err != nil {
		log.Fatalln("unexpected error:", err)
	}
	if s1 != content {
		log.Fatalf("%q != %q", s1, content)
	}
	if s1 != s2 {
		log.Fatalf("%q != %q", s1, s2)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("something wrong with cache dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entries in cache dir: %v", entries)
	}
}
