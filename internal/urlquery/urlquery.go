package urlquery

import (
	"bufio"
	"fmt"
	"io"
	"net/url"
	"strings"
	"sync"
)

type UrlQuery struct {
	results map[string]int

	reader io.Reader

	maxThreads int
	curThreads int32

	exitChan   chan struct{}
	finishChan chan struct{}
	countChan  chan string
	workChan   chan string
	wg         *sync.WaitGroup
}

func New(in io.Reader, maxThreads int) UrlQuery {
	return UrlQuery{
		results:    make(map[string]int),
		reader:     in,
		maxThreads: maxThreads,
		exitChan:   make(chan struct{}),
		finishChan: make(chan struct{}),
		countChan:  make(chan string),
		workChan:   make(chan string),
		wg:         new(sync.WaitGroup),
	}
}

func (u UrlQuery) Start() {
	defer u.wg.Wait()

	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		u.countWorker(u.countChan)
	}()
	scanner := bufio.NewScanner(u.reader)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		select {
		case <-u.exitChan:
			return
		default:
			uRL := scanner.Text()
			_, err := url.ParseRequestURI(uRL)
			if err != nil {
				u.countChan <- err.Error()
				u.logging(fmt.Sprintf("Error parsing URL:%s", uRL))
				continue
			}
			u.addJob(uRL)
			//fmt.Println(scanner.Text())
		}
	}

	//ticker := time.NewTicker(time.Second * 4)
	//defer ticker.Stop()
	//select {
	//case <-ticker.C:
	//case <-u.exitChan:
	//	return
	//}
	close(u.finishChan)
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

func (u UrlQuery) countWorker(in <-chan string) {
	for {
		select {
		case <-u.finishChan:
			return
		case result := <-in:
			u.results[result]++
		}
	}
}

func (u UrlQuery) logging(msg string) {
	fmt.Println(msg)

}

func (u UrlQuery) addJob(uRL string) {

}
