package deployment

import "testing"

func TestDeployment(t *testing.T) {
	res, err := Get("https://deployment.krateo.site/", "626c03950944e84673f8b82b")
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("%s", res.Package)

}
