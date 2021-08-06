package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseBackend(t *testing.T) {
	t.Run("Parse S3 backend with minimal inputs", func(t *testing.T) {
		config := `---
s3:
  bucket: foo
  key: bar
  region: us-east-1
`

		be, err := ParseBackend(config)
		assert.NoError(t, err)

		assert.Equal(t, be.S3.Bucket, "foo")
		assert.Equal(t, be.S3.Key, "bar")
		assert.Equal(t, be.S3.Region, "us-east-1")
		assert.Equal(t, be.Local, (*LocalBackendConfig)(nil))
	})

	t.Run("Parse Local backend with minimal inputs", func(t *testing.T) {
		config := `---
local:
  path: foo/terraform.tfstate
`

		be, err := ParseBackend(config)
		assert.NoError(t, err)

		assert.Equal(t, be.Local.Path, "foo/terraform.tfstate")
		assert.Equal(t, be.S3, (*S3BackendConfig)(nil))
	})

	t.Run("Error on unsupported backend type", func(t *testing.T) {
		config := `---
pg:
	conn_str: postgres://user:pass@db.example.com/terraform_backend
`

		_, err := ParseBackend(config)
		assert.Error(t, err)
	})
}
