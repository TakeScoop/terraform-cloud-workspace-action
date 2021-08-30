package action

import (
	"fmt"
	"net/http"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
)

// testServerResHandler returns a basic mux server route handler function
func testServerResHandler(t *testing.T, code int, resBody string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)

		_, err := fmt.Fprint(w, resBody)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// newTestTFClient returns a Terraform Cloud API client pointed at the provided address
func newTestTFClient(t *testing.T, address string) *tfe.Client {
	client, err := tfe.NewClient(&tfe.Config{
		Address: address,
		Token:   "12345",
	})
	if err != nil {
		t.Fatal(err)
	}

	return client
}
