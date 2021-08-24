package tfconfig

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

		assert.Contains(t, be, "s3")

		assert.Contains(t, be["s3"], "bucket")
		assert.Equal(t, be["s3"].(map[string]interface{})["bucket"], "foo")

		assert.Contains(t, be["s3"], "key")
		assert.Equal(t, be["s3"].(map[string]interface{})["key"], "bar")

		assert.Contains(t, be["s3"], "region")
		assert.Equal(t, be["s3"].(map[string]interface{})["region"], "us-east-1")
	})

	t.Run("Parse local backend with minimal inputs", func(t *testing.T) {
		config := `---
local:
  path: foo/terraform.tfstate
`
		be, err := ParseBackend(config)
		assert.NoError(t, err)

		assert.Contains(t, be, "local")

		assert.Contains(t, be["local"], "path")
		assert.Equal(t, be["local"].(map[string]interface{})["path"], "foo/terraform.tfstate")
	})

	t.Run("Parse any generic backend", func(t *testing.T) {
		config := `---
foo:
  bar: baz
`

		be, err := ParseBackend(config)
		assert.NoError(t, err)

		assert.Contains(t, be, "foo")

		assert.Contains(t, be["foo"], "bar")
		assert.Equal(t, be["foo"].(map[string]interface{})["bar"], "baz")
	})
}
