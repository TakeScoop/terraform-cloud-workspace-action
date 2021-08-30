package action

import (
	"fmt"
	"net/http"
	"testing"

	tfe "github.com/hashicorp/go-tfe"
)

func testServerResHandler(t *testing.T, code int, resBody string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(code)

		_, err := fmt.Fprint(w, resBody)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func newTestTFClient(t *testing.T, serverUrl string) *tfe.Client {
	client, err := tfe.NewClient(&tfe.Config{
		Address: serverUrl,
		Token:   "12345",
	})
	if err != nil {
		t.Fatal(err)
	}

	return client
}
