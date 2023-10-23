package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	timeStart := time.Now()

	client := &http.Client{}

	req, err := http.NewRequest("HEAD", "http://ya.ru", nil)
	if err != nil {
		log.Fatalln(err)
	}

	//req.Header.Set("User-Agent", "Golang_Spider_Bot/3.0")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	//body, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	log.Fatalln(err)
	//}

	for k, v := range resp.Header {
		fmt.Println(k, v)
	}

	log.Println(resp.Header.Get("Content-Length"))
	log.Println(resp.ContentLength)
	//log.Println(string(body))
	log.Printf("%s", time.Since(timeStart).Truncate(time.Millisecond))
}
