package apivalidation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// KeyIn ensures that the keys of a map are in the allowed values
func KeyIn(values ...string) Rule {
	return &keyInRule{values}
}

type keyInRule struct {
	values []string
}

func (r *keyInRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	ref.Value.Description += fmt.Sprintf("keys must be in (%s)", strings.Join(r.values, ","))
	return nil
}

func (r keyInRule) Validate(value any) error {
	validKeys := map[string]bool{}
	for _, v := range r.values {
		validKeys[v] = true
	}

	var jsonmap map[string]any
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &jsonmap)
	if err != nil {
		return err
	}

	for k := range jsonmap {
		if !validKeys[k] {
			return fmt.Errorf("key '%s' not allowed", k)
		}
	}
	return nil
}
