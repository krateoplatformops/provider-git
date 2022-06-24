package repo

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/cbroglie/mustache"
)

func TestRenderFunc(t *testing.T) {
	dat, err := ioutil.ReadFile("../../../testdata/values.json")
	if err != nil {
		t.Fatal(err)
	}

	var values map[string]interface{}
	err = json.Unmarshal(dat, &values)
	if err != nil {
		t.Fatal(err)
	}

	renderFunc := createRenderer(values)

	in, err := os.Open("../../../testdata/sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	out := new(bytes.Buffer)
	if err := renderFunc(in, out); err != nil {
		t.Fatal(err)
	}

	t.Logf("\n%s\n", out.String())
}

func createRenderer(values map[string]interface{}) func(in io.Reader, out io.Writer) error {
	return func(in io.Reader, out io.Writer) error {
		bin, err := ioutil.ReadAll(in)
		if err != nil {
			return err
		}
		tmpl, err := mustache.ParseString(string(bin))
		if err != nil {
			return err
		}

		return tmpl.FRender(out, values)
	}
}
