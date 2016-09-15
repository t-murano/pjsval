package pjsval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/format"
	"io"

	"github.com/achiku/varfmt"
	"github.com/lestrrat/go-jspointer"
	"github.com/lestrrat/go-jsschema"
	"github.com/lestrrat/go-jsval"
	"github.com/lestrrat/go-jsval/builder"
)

type validatorList []*jsval.JSVal

func (vl validatorList) Len() int {
	return len(vl)
}
func (vl validatorList) Swap(i, j int) {
	vl[i], vl[j] = vl[j], vl[i]
}
func (vl validatorList) Less(i, j int) bool {
	return vl[i].Name < vl[j].Name
}

// Generate validator source code
func Generate(in io.Reader, out io.Writer, pkg string) error {
	var m map[string]interface{}
	if err := json.NewDecoder(in).Decode(&m); err != nil {
		return err
	}

	validators := validatorList{}
	b := builder.New()

	for k, v := range m["properties"].(map[string]interface{}) {
		ptr := v.(map[string]interface{})["$ref"].(string)
		resolver, err := jspointer.New(ptr[1:])
		if err != nil {
			return err
		}

		resolved, err := resolver.Get(m)
		if err != nil {
			return err
		}

		m2, ok := resolved.(map[string]interface{})
		if !ok {
			return fmt.Errorf("failed type assertion %s resolved to map[string]interface{}", ptr[1:])
		}

		s := schema.New()
		if err := s.Extract(m2); err != nil {
			return err
		}

		validator, err := b.BuildWithCtx(s, m)
		if err != nil {
			return err
		}
		validator.Name = varfmt.PublicVarName(k + "Validator")
		validators = append(validators, validator)
	}

	g := jsval.NewGenerator()
	var src bytes.Buffer
	fmt.Fprintln(&src, "package "+pkg)
	fmt.Fprintf(&src, "import \"github.com/lestrrat/go-jsval\"")
	g.Process(&src, validators...)
	o, err := format.Source(src.Bytes())
	if err != nil {
		return err
	}
	out.Write(o)
	return nil
}
