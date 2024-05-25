package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

/*

Write a load-test tool:
Send thousands of requests to a server to analyze its performance
Use flag package to accept args
Send concurrent requests, collect results & returns aggregated result
Run pgm like:
go run . -n 10 -c 4 http://localhost:3000

Tool should prnt:
Made 10 requests to http://localhost:3000 with a concurrency setting of 4
Summary:
	Success    : 0%
	RPS        : 1677.8
	Requests   : 10
	Errors     : 10
	Bytes      : 0
	Duration   : 5.96ms
	Fastest    : 509µs
	Slowest    : 4.605ms
*/
type Summary struct {
	Success  float64
	RPS      float64
	Requests int
	Errors   int
	Bytes    int
	Duration time.Duration
	Fastest  time.Duration
	Slowest  time.Duration
}

type HTTPResponse struct {
	Success  int
	Errors   int
	Bytes    int
	Duration time.Duration
}

func main() {
	numRequest := flag.Int("n", 10000, "Number of Requests")
	threads := flag.Int("c", 100, "Number of concurrent threads")
	flag.Parse()
	initialTime := time.Now()
	var wg sync.WaitGroup
	ch := make(chan HTTPResponse, *threads)

	out := make(chan []HTTPResponse)
	go func() {
		var res []HTTPResponse
		for data := range ch {
			res = append(res, data)
			fmt.Printf("Summary:%+v\n", data)
		}
		out <- res
	}()
	for i := 0; i < *numRequest; i++ {
		wg.Add(1)
		go apiCall(&wg, ch)
	}
	wg.Wait()

	close(ch)
	outArr := <-out
	slowest := 0 * time.Second
	fastest := 1000000 * time.Second // Setting to some max time
	var success, errors, bytes int
	var dur, total time.Duration
	for _, out := range outArr {
		success += out.Success
		errors += out.Errors
		bytes += out.Bytes
		dur += out.Duration
		if out.Duration < fastest {
			fastest = out.Duration
		} else {
			slowest = out.Duration
		}
		total += out.Duration
	}
	fmt.Println("success", success)
	successPer := (float64(success) / float64(*numRequest)) * 100
	//sp := fmt.Sprintf("%dpercent", successPer)

	rps := float64(len(outArr)) / time.Since(initialTime).Seconds() // total request /total time
	fmt.Println("Total time", time.Since(initialTime))
	result := Summary{
		Success:  successPer,
		Requests: len(outArr),
		RPS:      rps,
		Errors:   errors,
		Bytes:    bytes,
		Duration: total / time.Duration(len(outArr)),
		Fastest:  fastest,
		Slowest:  slowest,
	}
	fmt.Printf("Final result :%+v", result)

}

func apiCall(wg *sync.WaitGroup, ch chan HTTPResponse) {
	defer wg.Done()

	var response HTTPResponse
	st := time.Now()
	re, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://httpbin.org/get", nil)

	res, err := http.DefaultClient.Do(re)
	if err != nil {
		fmt.Printf("failed to create request:%v\n", err)
		response.Success = 0
		response.Errors = 1
		response.Duration = time.Since(st)
		response.Bytes = 0
		fmt.Println("SendErr", err)

		ch <- response
		return
	}
	defer res.Body.Close()
	byte, _ := io.ReadAll(res.Body)
	response.Bytes = len(byte)
	if res.StatusCode == http.StatusOK {
		response.Success = 1
	} else {
		response.Errors = 1
	}
	response.Duration = time.Since(st)
	fmt.Println("Send", response)
	ch <- response

}
