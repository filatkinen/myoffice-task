package urlquery

import (
	"fmt"
	"io"
	"strings"
	"time"
)

type UrlQuery struct {
	results  map[string]int
	reader   io.Reader
	exitChan chan struct{}
}

func New(in io.Reader) UrlQuery {
	return UrlQuery{
		results:  make(map[string]int),
		reader:   in,
		exitChan: make(chan struct{}),
	}
}

func (u UrlQuery) Start() {
	ticker := time.NewTicker(time.Second * 4)
	defer ticker.Stop()
	select {
	case <-ticker.C:
	case <-u.exitChan:
		return
	}
}

func (u UrlQuery) Stop() {
	close(u.exitChan)
}

func (u UrlQuery) String() string {
	u.results["100"] = 5
	u.results["200"] = 10

	sb := strings.Builder{}
	for k, v := range u.results {
		sb.WriteString(fmt.Sprintf("Status code:%s, count:%d\n", k, v))
	}
	return sb.String()
}

func readWorker(in io.Reader, out chan<- string, command chan struct{}) {
	//u, err = url.ParseRequestURI("http://golang.org/index.html?#page1")

}

func countWorker(out chan<- string) {

}
