package urlquery

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const WorkerIdleTimeout = time.Second * 5
const MaxSizeObject = 10 * 1024 * 1024

type UrlQuery struct {
	results map[string]int

	reader io.Reader

	maxThreads int
	curThreads int

	exitChan   chan struct{}
	finishChan chan struct{}
	countChan  chan string
	workChan   chan string
	wg         sync.WaitGroup
	lock       sync.Mutex

	transport *http.Transport
	userAgent string
}

func New(in io.Reader, maxThreads int, userAgent string) (*UrlQuery, error) {
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		return nil, errors.New("defaultRoundTripper not an *http.Transport")
	}
	defaultTransport := defaultTransportPointer.Clone()
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100
	defaultTransport.DisableCompression = true

	return &UrlQuery{
		results:    make(map[string]int),
		reader:     in,
		maxThreads: maxThreads,
		exitChan:   make(chan struct{}),
		finishChan: make(chan struct{}),
		countChan:  make(chan string),
		workChan:   make(chan string),
		transport:  defaultTransport,
		userAgent:  userAgent,
	}, nil
}

func (u *UrlQuery) Start() {
	u.wg.Add(1)
	go func() {
		defer u.wg.Done()
		u.countWorker()
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
			u.checkQuantityWorkers()
			u.workChan <- uRL
		}
	}

	close(u.finishChan)
	u.wg.Wait()
}

func (u *UrlQuery) Stop() {
	close(u.exitChan)
}

func (u *UrlQuery) jobWorker() {
	ticker := time.NewTicker(WorkerIdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case uRL := <-u.workChan:
			u.queryURL(uRL)
		case <-ticker.C:
			u.lock.Lock()
			u.curThreads--
			u.lock.Unlock()
			return
		case <-u.finishChan:
			return
		}
	}
}

func (u *UrlQuery) countWorker() {
	for {
		select {
		case <-u.finishChan:
			return
		case result := <-u.countChan:
			u.results[result]++
		}
	}
}

func (u *UrlQuery) queryURL(uRL string) {
	timeStart := time.Now()
	client := &http.Client{Transport: u.transport}

	// trying to use HEAD method
	req, err := http.NewRequest("HEAD", uRL, nil)
	if err != nil {
		u.countChan <- err.Error()
		u.logging(fmt.Sprintf("Error preparing quering URL:%s (error=%v)", uRL, err))
		return
	}
	req.Header.Set("User-Agent", u.userAgent)

	respHEAD, err := client.Do(req)
	if err != nil {
		u.countChan <- err.Error()
		u.logging(fmt.Sprintf("Error quering URL:%s (error=%v)", uRL, err))
		return
	}

	defer func() {
		if err = respHEAD.Body.Close(); err != nil {
			u.logging(fmt.Sprintf("Error closing respHEAD.body URL:%s (error=%v)", uRL, err))
		}
	}()

	if respHEAD.ContentLength != -1 {
		u.countChan <- respHEAD.Status
		u.logging(fmt.Sprintf("Result of quering(HEAD) URL:%s, size(byte)=%d , time query=%s",
			uRL, respHEAD.ContentLength, time.Since(timeStart).Truncate(time.Millisecond)))
		return
	}

	// got  ContentLength==-1 so, using GET method
	req, err = http.NewRequest("GET", uRL, nil)
	if err != nil {
		u.countChan <- err.Error()
		u.logging(fmt.Sprintf("Error preparing quering URL:%s (error=%v)", uRL, err))
		return
	}
	req.Header.Set("User-Agent", u.userAgent)

	respGET, err := client.Do(req)
	if err != nil {
		u.countChan <- err.Error()
		u.logging(fmt.Sprintf("Error quering URL:%s (error=%v)", uRL, err))
		return
	}

	defer func() {
		if err = respGET.Body.Close(); err != nil {
			u.logging(fmt.Sprintf("Error closing respGET.body URL:%s (error=%v)", uRL, err))
		}
	}()

	if respGET.ContentLength != -1 {
		u.countChan <- respGET.Status
		u.logging(fmt.Sprintf("Result of quering(HEAD) URL:%s, size(byte)=%d , time query=%s",
			uRL, respGET.ContentLength, time.Since(timeStart).Truncate(time.Millisecond)))
		return
	}

	limitReader := io.LimitReader(respGET.Body, MaxSizeObject+1)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		u.countChan <- err.Error()
		u.logging(fmt.Sprintf("Error reading body URL:%s (error=%v)", uRL, err))
		return
	}

	if len(body) == (MaxSizeObject + 1) {
		u.countChan <- "Max size exceed"
		u.logging(fmt.Sprintf("Error MaxSize %d limit exceed  URL:%s",
			MaxSizeObject, uRL))
		return
	}

	u.countChan <- respGET.Status
	u.logging(fmt.Sprintf("Result of quering(GET) URL:%s, size(byte)=%d , time query=%s",
		uRL, respGET.ContentLength, time.Since(timeStart).Truncate(time.Millisecond)))
}

func (u *UrlQuery) checkQuantityWorkers() {
	u.lock.Lock()
	if u.curThreads < u.maxThreads {
		u.wg.Add(1)
		go func() {
			defer u.wg.Done()
			u.jobWorker()
		}()
		u.curThreads++
	}
	u.lock.Unlock()
}

func (u *UrlQuery) String() string {
	sb := strings.Builder{}
	for k, v := range u.results {
		sb.WriteString(fmt.Sprintf("Status code:%s, count:%d\n", k, v))
	}
	return sb.String()
}

func (u *UrlQuery) logging(msg string) {
	fmt.Println(msg)

}
