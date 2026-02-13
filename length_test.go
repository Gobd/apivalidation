package apivalidation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLength(t *testing.T) {
	r := Length(0, 100)
	err := r.Validate("Richard-Breslau-Straãƒæ'ã†Â€™Ãƒâ€ Ã¢Â‚¬Â„¢Ãƒæ'ã¢Â‚¬Â¦Ãƒâ€Šã‚Â¸E 2") // 65
	require.Nil(t, err)

	err = r.Validate("Richard-Breslau-Straãƒæ'ã†Â€™Ãƒâ€ Ã¢Â‚¬Â„¢Ãƒæ'ã¢Â‚¬Â¦Ãƒâ€Šã‚Â¸E 21234567890abcdefghijklmnopqrstuvwxy") // 100
	require.Nil(t, err)

	err = r.Validate("Richard-Breslau-Straãƒæ'ã†Â€™Ãƒâ€ Ã¢Â‚¬Â„¢Ãƒæ'ã¢Â‚¬Â¦Ãƒâ€Šã‚Â¸E 21234567890abcdefghijklmnopqrstuvwxyz") // 101
	require.NotNil(t, err)
}
