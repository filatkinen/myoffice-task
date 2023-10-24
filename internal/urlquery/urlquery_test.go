package urlquery

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

var src = rand.NewSource(time.Now().UnixNano())
var r = rand.New(src)

func TestUrlQuery(t *testing.T) {

	httptest.NewServer()
	hs := http.Server{Addr: "0.0.0.0:8089"}
	hs.Handler = http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			switch request.Method {
			case http.MethodHead:
				if randomBool() {
					writer.Header().Set("Content-Length", "-1")
				} else {
					writer.Header().Set("Content-Length", "-1")
				}
				writer.Write([]byte(nil))
			case http.MethodGet:

			}
		})
	_ = hs.ListenAndServe()
}

func randomBool() bool {
	if r.Intn(2) == 1 {
		return true
	}
	return false
}
