package main

// TODO:
// just put in separate function balancer
// and test it

import (
	"context"
	"crypto/sha512"
	"flag"
	"fmt"
	"hash/crc64"
	"io"
	"log"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/magicvegetable/architecture-lab-4/httptools"
	"github.com/magicvegetable/architecture-lab-4/signal"
)

var (
	port       = flag.Int("port", 8090, "load balancer port")
	timeoutSec = flag.Int("timeout-sec", 3, "request timeout time in seconds")
	https      = flag.Bool("https", false, "whether backends support HTTPs")

	traceEnabled = flag.Bool("trace", false, "whether to include tracing information into responses")

	serversM = sync.Mutex{}
)

var (
	timeout     = time.Duration(*timeoutSec) * time.Second
	serversPool = []string{
		"server1:8080",
		"server2:8080",
		"server3:8080",
	}
)

func scheme() string {
	if *https {
		return "https"
	}
	return "http"
}

func health(dst string) bool {
	ctx, _ := context.WithTimeout(context.Background(), timeout)
	req, _ := http.NewRequestWithContext(ctx, "GET",
		fmt.Sprintf("%s://%s/health", scheme(), dst), nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusOK {
		return false
	}
	return true
}

func forward(dst string, rw http.ResponseWriter, r *http.Request) error {
	ctx, _ := context.WithTimeout(r.Context(), timeout)
	fwdRequest := r.Clone(ctx)
	fwdRequest.RequestURI = ""
	fwdRequest.URL.Host = dst
	fwdRequest.URL.Scheme = scheme()
	fwdRequest.Host = dst

	resp, err := http.DefaultClient.Do(fwdRequest)
	if err == nil {
		for k, values := range resp.Header {
			for _, value := range values {
				rw.Header().Add(k, value)
			}
		}
		if *traceEnabled {
			rw.Header().Set("lb-from", dst)
		}
		log.Println("fwd", resp.StatusCode, resp.Request.URL)
		rw.WriteHeader(resp.StatusCode)
		defer resp.Body.Close()
		_, err := io.Copy(rw, resp.Body)
		if err != nil {
			log.Printf("Failed to write response: %s", err)
		}
		return nil
	} else {
		log.Printf("Failed to get response from %s: %s", dst, err)
		rw.WriteHeader(http.StatusServiceUnavailable)
		return err
	}
}

var poly = uint64(time.Now().Unix())

// In reality is checksum of hash + polt
func hash(str string) uint64 {
	hasher := sha512.New()
	hasher.Write([]byte(str))

	table := crc64.MakeTable(poly)

	return crc64.Checksum(hasher.Sum(nil), table)
}

func GetAvailableServer(addr string) string {
	if len(serversPool) == 0 {
		return ""
	}

	serverIndex := hash(addr)

	serversM.Lock() // should?

	serverIndex %= uint64(len(serversPool))

	serversM.Unlock()

	return serversPool[serverIndex]
}

func main() {
	flag.Parse()

	log.Println("Balancer started")

	for _, server := range serversPool {
		server := server
		go func() {
			for range time.Tick(1 * time.Second) {
				alive := health(server)

				if !alive {
					serversM.Lock()
					serverI := slices.Index(serversPool, server)

					newServersPool := serversPool[:serverI]
					newServersPool = append(newServersPool, serversPool[serverI+1:]...)

					serversPool = newServersPool

					serversM.Unlock()

					break
				}
			}
		}()
	}

	frontend := httptools.CreateServer(*port, http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		server := GetAvailableServer(r.RemoteAddr)

		if server != "" {
			forward(server, rw, r)
		}
	}))

	log.Println("Starting load balancer...")
	log.Printf("Tracing support enabled: %t", *traceEnabled)
	frontend.Start()
	signal.WaitForTerminationSignal()
}
