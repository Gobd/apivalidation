package apivalidation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

const creditCardNumberLength = 16

type hasAlphabetic struct {
	isCreditCardNumberCheck bool
}

// HasAlphabetic returns a validation rule that checks if a string contains at least one alphabetic character.
func HasAlphabetic() Rule {
	return hasAlphabetic{}
}

// NonCreditCardNumber returns a validation rule that rejects strings that look like credit card numbers.
func NonCreditCardNumber() Rule {
	return hasAlphabetic{isCreditCardNumberCheck: true}
}

func (r hasAlphabetic) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Description += "Must contain at least one alphabetic character. "
	return nil
}

var (
	alphabeticRegexp = regexp.MustCompile(`[^[:alpha:]]`)
	numberRegexp     = regexp.MustCompile(`\D`)
)

func (r hasAlphabetic) Validate(value any) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("expected string, got %T", value)
	}

	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	ar := alphabeticRegexp.ReplaceAllString(v, "")
	if ar == "" {
		if r.isCreditCardNumberCheck {
			nr := numberRegexp.ReplaceAllString(v, "")
			if len(nr) != creditCardNumberLength {
				return nil
			}
			return fmt.Errorf("must not be a credit card number")
		}
		return fmt.Errorf("must contain at least one alphabetic character")
	}
	return nil
}
