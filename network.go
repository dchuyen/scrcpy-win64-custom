package main

import (
	"net"
	"sort"
	"strconv"
	"sync"
	"time"
)

func scanPorts(host string, startPort, endPort int, timeout time.Duration, workers int) ([]int, error) {
	if workers <= 0 {
		workers = 200
	}

	jobs := make(chan int)
	results := make(chan int, 1000)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range jobs {
				if isOpen(host, port, timeout) {
					results <- port
				}
			}
		}()
	}

	go func() {
		for port := startPort; port <= endPort; port++ {
			jobs <- port
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var open []int
	for port := range results {
		open = append(open, port)
	}

	sort.Ints(open)
	return open, nil
}

func isOpen(host string, port int, timeout time.Duration) bool {
	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}
