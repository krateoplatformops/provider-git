package req

import (
	"context"
	"net/http"
	"testing"
)

func TestClone(t *testing.T) {
	t.Run("from URL", func(t *testing.T) {
		rb1 := Get().
			Url("http://example.com").
			Path("a/").
			Header("a", "1").
			Header("b", "2").
			Cookie("cookie", "base").
			Param("a", "1").
			Param("b", "2")
		rb2 := rb1.Clone().
			Host("host.example").
			Path("b").
			Header("b", "3").
			Header("c", "4").
			Cookie("cookie", "override").
			Param("b", "3").
			Param("c", "4")
		rb3 := rb1.Clone().
			Host("host.example3").
			Path("c").
			Header("b", "5").
			Header("c", "6").
			Cookie("alternate", "value").
			Param("b", "5").
			Param("c", "6")
		req1, err := rb1.Request(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if req1.URL.Host != "example.com" {
			t.Fatalf("bad host: %v", req1.URL)
		}
		if req1.URL.Path != "/a/" {
			t.Fatalf("bad path: %v", req1.URL)
		}
		if req1.Header.Get("b") != "2" || req1.Header.Get("c") != "" {
			t.Fatalf("bad header: %v", req1.URL)
		}
		if cookies := req1.Header.Get("Cookie"); cookies != "cookie=base" {
			t.Fatalf("bad cookies: %q", cookies)
		}
		if q := req1.URL.Query(); q.Get("b") != "2" || q.Get("c") != "" {
			t.Fatalf("bad query: %v", req1.URL)
		}
		req2, err := rb2.Request(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if req2.URL.Host != "host.example" {
			t.Fatalf("bad host: %v", req2.URL)
		}
		if req2.URL.Path != "/a/b" {
			t.Fatalf("bad path: %v", req2.URL.Path)
		}
		if req2.Header.Get("b") != "3" || req2.Header.Get("c") != "4" {
			t.Fatalf("bad header: %v", req2.URL)
		}
		if cookies := req2.Header.Get("Cookie"); cookies != "cookie=base; cookie=override" {
			t.Fatalf("bad cookies: %q", cookies)
		}
		if q := req2.URL.Query(); q.Get("b") != "3" || q.Get("c") != "4" {
			t.Fatalf("bad query: %v", req2.URL)
		}
		req3, err := rb3.Request(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if req3.URL.Host != "host.example3" {
			t.Fatalf("bad host: %v", req3.URL)
		}
		if req3.URL.Path != "/a/c" {
			t.Fatalf("bad path: %v", req3.URL.Path)
		}
		if req3.Header.Get("b") != "5" || req3.Header.Get("c") != "6" {
			t.Fatalf("bad header: %v", req3.URL)
		}
		if cookies := req3.Header.Get("Cookie"); cookies != "cookie=base; alternate=value" {
			t.Fatalf("bad cookies: %q", cookies)
		}
		if q := req3.URL.Query(); q.Get("b") != "5" || q.Get("c") != "6" {
			t.Fatalf("bad query: %v", req3.URL)
		}
	})
	t.Run("from new", func(t *testing.T) {
		rb1 := new(Builder).
			Host("example.com").
			Header("a", "1").
			Header("b", "2").
			Param("a", "1").
			Param("b", "2")
		rb2 := rb1.Clone().
			Host("host.example").
			Path("/2").
			Header("b", "3").
			Header("c", "4").
			Param("b", "3").
			Param("c", "4")
		rb3 := rb1.Clone().
			Host("host.example3").
			Path("/3").
			Header("b", "5").
			Header("c", "6").
			Param("b", "5").
			Param("c", "6")
		req1, err := rb1.Request(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if req1.URL.Host != "example.com" {
			t.Fatalf("bad host: %v", req1.URL)
		}
		if req1.Header.Get("b") != "2" || req1.Header.Get("c") != "" {
			t.Fatalf("bad header: %v", req1.URL)
		}
		if q := req1.URL.Query(); q.Get("b") != "2" || q.Get("c") != "" {
			t.Fatalf("bad query: %v", req1.URL)
		}
		req2, err := rb2.Request(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if req2.URL.Host != "host.example" {
			t.Fatalf("bad host: %v", req2.URL)
		}
		if req2.URL.Path != "/2" {
			t.Fatalf("bad path: %v", req2.URL.Path)
		}
		if req2.Header.Get("b") != "3" || req2.Header.Get("c") != "4" {
			t.Fatalf("bad header: %v", req2.URL)
		}
		if q := req2.URL.Query(); q.Get("b") != "3" || q.Get("c") != "4" {
			t.Fatalf("bad query: %v", req2.URL)
		}
		req3, err := rb3.Request(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if req3.URL.Host != "host.example3" {
			t.Fatalf("bad host: %v", req3.URL)
		}
		if req3.URL.Path != "/3" {
			t.Fatalf("bad path: %v", req3.URL.Path)
		}
		if req3.Header.Get("b") != "5" || req3.Header.Get("c") != "6" {
			t.Fatalf("bad header: %v", req3.URL)
		}
		if q := req3.URL.Query(); q.Get("b") != "5" || q.Get("c") != "6" {
			t.Fatalf("bad query: %v", req3.URL)
		}
	})
}

func TestScheme(t *testing.T) {
	const res = `HTTP/1.1 200 OK
Content-Type: text/plain; charset=UTF-8
Date: Mon, 24 May 2021 18:48:50 GMT

An example response.`
	var s string
	const expected = `An example response.`
	var trans http.Transport
	trans.RegisterProtocol("string", ReplayString(res))
	if err := Get().
		Url("example").
		Scheme("string").
		Client(&http.Client{
			Transport: &trans,
		}).
		ToString(&s).
		Do(context.Background()); err != nil {
		t.Fatal(err)
	}
	if s != expected {
		t.Fatalf("%q != %q", s, expected)
	}
}

func TestPath(t *testing.T) {
	cases := map[string]struct {
		base   string
		paths  []string
		result string
	}{
		"base-only": {
			"example",
			[]string{},
			"https://example",
		},
		"base+abspath": {
			"https://example",
			[]string{"/a"},
			"https://example/a",
		},
		"multi-abs-paths": {
			"https://example",
			[]string{"/a", "/b/", "/c"},
			"https://example/c",
		},
		"base+rel-path": {
			"https://example/a/",
			[]string{"./b"},
			"https://example/a/b",
		},
		"base+rel-paths": {
			"https://example/a/",
			[]string{"./b/", "./c"},
			"https://example/a/b/c",
		},
		"rel-path": {
			"https://example/",
			[]string{"a/", "./b"},
			"https://example/a/b",
		},
		"base+multi-paths": {
			"https://example/a/",
			[]string{"b/", "c"},
			"https://example/a/b/c",
		},
		"base+slash+multi-paths": {
			"https://example/a/",
			[]string{"b/", "c"},
			"https://example/a/b/c",
		},
		"multi-root": {
			"https://example/",
			[]string{"a", "b", "c"},
			"https://example/c",
		},
		"dot-dot-paths": {
			"https://example/",
			[]string{"a/", "b/", "../c"},
			"https://example/a/c",
		},
		"more-dot-dot-paths": {
			"https://example/",
			[]string{"a/b/c/", "../d/", "../e"},
			"https://example/a/b/e",
		},
		"more-dot-dot-paths+rel-path": {
			"https://example/",
			[]string{"a/b/c/", "../d/", "../e/", "./f"},
			"https://example/a/b/e/f",
		},
		"even-more-dot-dot-paths+base": {
			"https://example/a/b/c/",
			[]string{"../../d"},
			"https://example/a/d",
		},
		"too-many-dot-dot-paths": {
			"https://example",
			[]string{"../a"},
			"https://example/a",
		},
		"too-many-dot-dot-paths+base": {
			"https://example/",
			[]string{"../a"},
			"https://example/a",
		},
		"last-abs-path-wins": {
			"https://example/a/",
			[]string{"b/", "c/", "/d"},
			"https://example/d",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := Get().Url(tc.base)
			for _, p := range tc.paths {
				b.Path(p)
			}
			r, err := b.Request(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if u := r.URL.String(); u != tc.result {
				t.Fatalf("got %q; want %q", u, tc.result)
			}
		})
	}
}
