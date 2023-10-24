package urlquery

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"
)

const maxSlice = 10000

var (
	host = "localhost"
	port = "8089"
)

func TestUrlQuery(t *testing.T) {

	t.Run("10 url, 4 threads,  output os.Stdout", func(t *testing.T) {
		testUrlQuery(t, 10, 4, os.Stdout)
	})
	t.Run("10k url, 100 threads, output discarded", func(t *testing.T) {
		testUrlQuery(t, 10*1000, 100, io.Discard)
	})
	t.Run("100k url, 10k threads, output discarded", func(t *testing.T) {
		testUrlQuery(t, 100*1000, 10*1000, io.Discard)
	})
}

func testUrlQuery(t *testing.T, maxURL int, maxThreads int, output io.Writer) {
	var src = rand.NewSource(time.Now().UnixNano())
	var rnd = rand.New(src)
	b := make([]byte, maxSlice)

	hs := http.Server{Addr: net.JoinHostPort(host, port)}
	hs.Handler = http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			size := rnd.Intn(maxSlice)
			switch r.Method {
			case http.MethodHead:
				if randomBool(rnd) {
					w.Header().Set("Content-Length", strconv.Itoa(size))
				}
				// empty Content-Length header, so next time CLI will come using method GET
				time.Sleep(time.Millisecond * time.Duration(rnd.Intn(30)))
				w.WriteHeader(http.StatusOK)
			case http.MethodGet:
				time.Sleep(time.Millisecond * time.Duration(rnd.Intn(30)+30))
				w.Write(b[:size])
			}
		})

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := hs.ListenAndServe()
		require.Error(t, err, http.ErrServerClosed)
	}()

	// waiting till http server start
	time.Sleep(time.Millisecond * 200)

	urlGen := newURLGenerator("http://"+host+":"+port, maxURL)

	urlQuery, err := New(&urlGen, output, maxThreads, "testing agent:))")
	require.NoError(t, err)

	urlQuery.Start()

	err = hs.Close()
	require.NoError(t, err)

	wg.Wait()

	fmt.Println("Results:")
	fmt.Println(urlQuery)

}

func randomBool(rnd *rand.Rand) bool {
	if rnd.Intn(2) == 1 {
		return true
	}
	return false
}

type urlGenerator struct {
	count   int
	baseURL string
	maxURL  int
}

func newURLGenerator(baseURL string, maxURL int) urlGenerator {
	return urlGenerator{baseURL: baseURL, maxURL: maxURL}
}

func (u *urlGenerator) Read(p []byte) (int, error) {
	if u.count+1 > u.maxURL {
		return 0, io.EOF
	}
	u.count++
	url := u.baseURL + "/" + strconv.Itoa(u.count) + "\n"
	copy(p, url)
	return len(url), nil
}
