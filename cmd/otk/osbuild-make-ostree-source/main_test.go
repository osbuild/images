package main_test

import (
	"bytes"
	"testing"

	sourcemaker "github.com/osbuild/images/cmd/otk/osbuild-make-ostree-source"
	"github.com/stretchr/testify/require"
)

func TestSourceMakerBasic(t *testing.T) {
	require := require.New(t)

	input := `
{
  "tree": {
	"const": {
	  "ref": "does/not/matter",
	  "checksum": "d04105393ca0617856b34f897842833d785522e41617e17dca2063bf97e294ef",
	  "url": "https://ostree.example.org/repo"
	}
  }
}`

	expOutput := `{
  "tree": {
    "org.osbuild.ostree": {
      "items": {
        "d04105393ca0617856b34f897842833d785522e41617e17dca2063bf97e294ef": {
          "remote": {
            "url": "https://ostree.example.org/repo"
          }
        }
      }
    }
  }
}
`

	inpBuf := bytes.NewBuffer([]byte(input))
	outBuf := &bytes.Buffer{}

	require.NoError(sourcemaker.Run(inpBuf, outBuf))
	require.Equal(expOutput, outBuf.String())
}

func TestSourceMakerWithSecrets(t *testing.T) {
	require := require.New(t)

	input := `
{
  "tree": {
    "const": {
      "ref": "does/not/matter",
      "checksum": "d04105393ca0617856b34f897842833d785522e41617e17dca2063bf97e294ef",
      "url": "https://ostree.example.org/repo",
      "secrets": "org.osbuild.rhsm.consumer"
    }
  }
}`

	expOutput := `{
  "tree": {
    "org.osbuild.ostree": {
      "items": {
        "d04105393ca0617856b34f897842833d785522e41617e17dca2063bf97e294ef": {
          "remote": {
            "url": "https://ostree.example.org/repo",
            "secrets": {
              "name": "org.osbuild.rhsm.consumer"
            }
          }
        }
      }
    }
  }
}
`

	inpBuf := bytes.NewBuffer([]byte(input))
	outBuf := &bytes.Buffer{}

	require.NoError(sourcemaker.Run(inpBuf, outBuf))
	require.Equal(expOutput, outBuf.String())
}
