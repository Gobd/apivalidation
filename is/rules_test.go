package is_test

import (
	"strings"
	"testing"

	"github.com/Gobd/apivalidation/is"
	"github.com/stretchr/testify/require"
)

func TestDomain(t *testing.T) {
	for _, domain := range []string{"example.com", "EXAMPLE.COM", "a.example.com", "example.xn--p1ai", "example.xn--bcher-kva"} {
		require.NoError(t, is.Domain.Validate(domain), domain)
	}
	for _, domain := range []string{"not a domain", "-bad.example.com", "bad-.example.com", "a..com", "https://example.com", strings.Repeat("a", 64) + ".com", strings.Repeat("a", 256)} {
		require.Error(t, is.Domain.Validate(domain), domain)
	}
}

func TestEmailFormatDoesNotRequireDNS(t *testing.T) {
	// .invalid is reserved and cannot have an MX record, but the syntax is valid.
	require.NoError(t, is.EmailFormat.Validate("person@example.invalid"))
	require.Error(t, is.EmailFormat.Validate("not-an-email"))
	require.NoError(t, is.EmailFormat.Validate("")) // use Required for emptiness
	require.NoError(t, is.Email.Validate(""))       // also skips empty without a DNS lookup
}
