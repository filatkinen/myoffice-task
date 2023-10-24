package urlquery

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	WorkerIdleTimeout = time.Second * 5
	MaxSizeObject     = 10 * 1024 * 1024
	ConnectTimeOut    = time.Second * 5
)

type URLQuery struct {
	results map[string]int

	reader io.Reader
	writer io.Writer

	maxThreads int
	curThreads int
	userAgent  string

	exitChan  chan struct{}
	countChan chan string
	taskChan  chan string
	wg        sync.WaitGroup

	lock      sync.Mutex
	transport *http.Transport
}

func New(in io.Reader, out io.Writer, maxThreads int, userAgent string) (*URLQuery, error) {
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		return nil, errors.New("defaultRoundTripper not an *http.Transport")
	}
	defaultTransport := defaultTransportPointer.Clone()
	defaultTransport.MaxIdleConns = 100
	defaultTransport.MaxIdleConnsPerHost = 100
	defaultTransport.DisableCompression = true

	return &URLQuery{
		results:    make(map[string]int),
		reader:     in,
		writer:     out,
		maxThreads: maxThreads,
		exitChan:   make(chan struct{}),
		countChan:  make(chan string),
		taskChan:   make(chan string),
		transport:  defaultTransport,
		userAgent:  userAgent,
	}, nil
}

func (u *URLQuery) Start() {
	go func() {
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
				u.countChan <- "Error parsing URL"
				u.logging(fmt.Sprintf("Error parsing URL:%s", uRL))
				continue
			}
			u.checkQuantityWorkers()
			u.taskChan <- uRL
		}
	}

	close(u.taskChan)
	u.wg.Wait()
	close(u.countChan)
}

func (u *URLQuery) Stop() {
	close(u.exitChan)
}

func (u *URLQuery) jobWorker() {
	ticker := time.NewTicker(WorkerIdleTimeout)
	defer ticker.Stop()

	for {
		select {
		case uRL, ok := <-u.taskChan:
			if !ok {
				return
			}
			u.queryURL(uRL)
		case <-ticker.C:
			u.lock.Lock()
			u.curThreads--
			u.lock.Unlock()
			return
		}
	}
}

func (u *URLQuery) countWorker() {
	for {
		result, ok := <-u.countChan
		if !ok {
			return
		}
		u.results[result]++
	}
}

func (u *URLQuery) queryURL(uRL string) {
	timeStart := time.Now()
	client := &http.Client{
		Transport: u.transport,
		Timeout:   ConnectTimeOut,
	}

	// trying to use HEAD method
	req, err := http.NewRequestWithContext(context.Background(), "HEAD", uRL, nil)
	if err != nil {
		u.countChan <- "Error creating query URL"
		u.logging(fmt.Sprintf("Error preparing quering URL:%s (error=%v)", uRL, err))
		return
	}
	req.Header.Set("User-Agent", u.userAgent)

	respHEAD, err := client.Do(req)
	if err != nil {
		u.countChan <- "Error querying URL"
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
	req, err = http.NewRequestWithContext(context.Background(), "GET", uRL, nil)
	if err != nil {
		u.countChan <- "Error creating query URL"
		u.logging(fmt.Sprintf("Error preparing quering URL:%s (error=%v)", uRL, err))
		return
	}
	req.Header.Set("User-Agent", u.userAgent)

	respGET, err := client.Do(req)
	if err != nil {
		u.countChan <- "Error querying URL"
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
		u.countChan <- "Error reading body"
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
		uRL, len(body), time.Since(timeStart).Truncate(time.Millisecond)))
}

func (u *URLQuery) checkQuantityWorkers() {
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

func (u *URLQuery) String() string {
	sb := strings.Builder{}
	for k, v := range u.results {
		sb.WriteString(fmt.Sprintf("Status:\"%s\", count:%d\n", k, v))
	}
	return sb.String()
}

func (u *URLQuery) logging(msg string) {
	_, err := fmt.Fprintln(u.writer, msg)
	if err != nil {
		log.Printf("error wtiting log\n, %s", err)
	}
}
