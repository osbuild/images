package cmdutil_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/buildconfig"
	"github.com/osbuild/images/internal/cmdutil"
)

func TestNewRNGSeed(t *testing.T) {
	// env is global - run all tests in one function so they don't run in
	// parallel
	t.Run("default", func(t *testing.T) {
		t.Setenv(cmdutil.RNG_SEED_ENV_KEY, "")
		seed1, err := cmdutil.NewRNGSeed()
		require.Nil(t, err)
		require.IsType(t, int64(0), seed1)

		seed2, err := cmdutil.NewRNGSeed()
		require.Nil(t, err)
		require.IsType(t, int64(0), seed1)
		require.NotEqual(t, seed1, seed2) // 1/2^64 chance this will fail randomly
	})

	t.Run("happy", func(t *testing.T) {
		t.Setenv(cmdutil.RNG_SEED_ENV_KEY, "1234")
		seed, err := cmdutil.NewRNGSeed()
		require.Nil(t, err)
		assert.Equal(t, seed, int64(1234))
	})

	t.Run("error", func(t *testing.T) {
		t.Setenv(cmdutil.RNG_SEED_ENV_KEY, "NaN")
		_, err := cmdutil.NewRNGSeed()
		require.EqualError(t, err, fmt.Sprintf(`failed to parse %s: strconv.ParseInt: parsing "NaN": invalid syntax`, cmdutil.RNG_SEED_ENV_KEY))
	})
}

func TestSeedArgFor(t *testing.T) {
	t.Setenv(cmdutil.RNG_SEED_ENV_KEY, "1234")

	for _, tc := range []struct {
		bcName, imgTypeName, distroName, archName string
		expectedSeed                              int64
	}{
		{"bcName", "fakeImgType", "fakeDistro", "x86_64", 9170052743323116054},
		{"bcName1", "fakeImgType", "fakeDistro", "x86_64", -7134826073208782961},
		{"bcName", "fakeImgType1", "fakeDistro", "x86_64", 4026045880862600579},
		{"bcName", "fakeImgType", "fakeDistro1", "x86_64", 3669869122697339647},
		{"bcName", "fakeImgType", "fakeDistro1", "aarch64", 47752167762999679},
	} {
		bc := &buildconfig.BuildConfig{Name: tc.bcName}
		seedArg, err := cmdutil.SeedArgFor(bc, tc.imgTypeName, tc.distroName, tc.archName)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedSeed, seedArg)
	}
}
