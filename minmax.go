package apivalidation

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type thresholdRule struct {
	validation.ThresholdRule
	threshold any
	min       bool
}

// Min returns a validation rule that checks if a value is greater than or equal to the specified minimum.
func Min(threshold any) Rule {
	return thresholdRule{
		validation.Min(threshold),
		threshold,
		true,
	}
}

// Max returns a validation rule that checks if a value is less than or equal to the specified maximum.
func Max(threshold any) Rule {
	return thresholdRule{
		validation.Max(threshold),
		threshold,
		false,
	}
}

func (r thresholdRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Type.Is(openapi3.TypeString) {
		ref.Value.Format = fmt.Sprintf("%T", r.threshold)
	}
	f, err := getFloat(r.threshold)
	if err != nil {
		return err
	}
	if r.min {
		ref.Value.Min = &f
	} else {
		ref.Value.Max = &f
	}
	return nil
}

var floatType = reflect.TypeOf(float64(0))

func getFloat(unk any) (float64, error) {
	v := reflect.ValueOf(unk)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return 0, fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
}

// Validate checks if the given value is valid or not.
func (r thresholdRule) Validate(value any) error {
	value, isNil := validation.Indirect(value)
	if isNil || validation.IsEmpty(value) {
		return nil
	}

	if reflect.ValueOf(value).Kind() != reflect.String {
		return r.ThresholdRule.Validate(value)
	}

	// Handle json.Number and other types
	if v, ok := value.(fmt.Stringer); ok {
		value = v.String()
	}

	var err error
	rv := reflect.ValueOf(r.threshold)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err = strconv.ParseInt(value.(string), 10, 64)
		if err != nil {
			return errors.New("must be int64")
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		value, err = strconv.ParseUint(value.(string), 10, 64)
		if err != nil {
			return errors.New("must be uint64")
		}
	case reflect.Float32, reflect.Float64:
		value, err = strconv.ParseFloat(value.(string), 64)
		if err != nil {
			return errors.New("must be float64")
		}
	}

	return r.ThresholdRule.Validate(value)
}
