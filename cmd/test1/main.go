package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"time"
)

func main() {
	var src = rand.NewSource(time.Now().UnixNano())
	var r = rand.New(src)

	hs := http.Server{Addr: "0.0.0.0:8089"}
	hs.Handler = http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			size := r.Intn(1000)
			switch request.Method {
			case http.MethodHead:
				//	if randomBool(r) {
				//		writer.Header().Set("Content-Length", strconv.Itoa(size))
				//	} else {
				//		writer.Header().Set("Content-Length", "0")
				//	}
				//	//writer.Write([]byte(nil))
				writer.Write(make([]byte, size))
			case http.MethodGet:
				writer.Write(make([]byte, size))
			}
		})
	_ = hs.ListenAndServe()

}

func randomBool(r *rand.Rand) bool {
	if r.Intn(2) == 1 {
		return true
	}
	return false
}

func main1() {
	cst := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBlob, _ := httputil.DumpRequest(r, true)
		w.Write(reqBlob)
	}))
	defer cst.Close()

	reqBody := ioutil.NopCloser(bytes.NewBufferString(`{}`))
	req, _ := http.NewRequest("POST", cst.URL, reqBody)
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	resBody, _ := ioutil.ReadAll(res.Body)
	log.Printf("Response Body:\n%s", resBody)
}
