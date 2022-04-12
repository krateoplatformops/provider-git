package gitrepo

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestInfo(t *testing.T) {
	table := []struct {
		url  string
		want Info
	}{
		{
			url: "https://github.com/krateoplatformops/krateo",
			want: Info{
				host:     "github.com",
				owner:    "krateoplatformops",
				repoName: "krateo",
			},
		},
	}

	for _, tc := range table {
		t.Run(tc.url, func(t *testing.T) {
			got, err := GetRepoInfo(tc.url)
			if err != nil {
				t.Fatal(err)
			}

			if !cmp.Equal(got, tc.want, cmp.AllowUnexported(Info{})) {
				t.Fatalf("expected: %+v - got: %+v", tc.want, got)
			}
		})
	}
}
