package git

import (
	"testing"
)

func TestRepoConfig(t *testing.T) {
	table := []struct {
		inp  RepoOpts
		org  string
		name string
		host string
	}{
		{
			inp: RepoOpts{
				Url: "https://github.com/krateoplatformops/krateo",
			},
			org:  "krateoplatformops",
			name: "krateo",
			host: "github.com",
		},
	}

	for _, tc := range table {
		t.Run(tc.inp.Url, func(t *testing.T) {
			got, err := tc.inp.OrgName()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.org {
				t.Fatalf("expected: %v - got: %v", tc.org, got)
			}

			got, err = tc.inp.RepoName()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.name {
				t.Fatalf("expected: %v - got: %v", tc.name, got)
			}

			got, err = tc.inp.Host()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.host {
				t.Fatalf("expected: %v - got: %v", tc.host, got)
			}

			//if !cmp.Equal(got, tc.want, cmp.AllowUnexported(Info{})) {
			//	t.Fatalf("expected: %+v - got: %+v", tc.want, got)
			//}
		})
	}
}
