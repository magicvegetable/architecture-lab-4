package integration

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"
)

const baseAddress = "http://balancer:8090"

var client = http.Client{
	Timeout: 3 * time.Second,
}

const (
	READ_DEADLINE                        = 1 * time.Second
	HTTP_BALANCER_TESTS_AMOUNT           = 10
	HTTP_BALANCER_CHECKS_PER_TEST_AMOUNT = 10
	COPY_BUFFER_SIZE                     = 4096
)

func Copy(out io.Writer, connection net.Conn) error {
	size := COPY_BUFFER_SIZE
	buffer := make([]byte, size)

	for {
		deadline := time.Now().Add(READ_DEADLINE)
		connection.SetReadDeadline(deadline)
		_, err := connection.Read(buffer)

		if err != nil {

			if err != io.EOF && time.Now().Before(deadline) {
				panic("cannot read from io.Reader...")
			}

			break
		}

		n, err := out.Write(buffer)

		if err != nil {
			panic("cannot write to io.Writer...")
		}

		for n != len(buffer) {
			i, err := out.Write(buffer[n:])

			if err != nil {
				panic("cannot write to io.Writer...")
			}

			n += i
		}

		buffer = make([]byte, size)
	}

	return nil
}

func GetLbfrom(url *url.URL, connection net.Conn) (string, error) {
	_, err := connection.Write([]byte("GET " + url.Path + " HTTP/1.1\r\n" + "Host: " + url.Hostname() + "\r\n\r\n"))

	buffer := &bytes.Buffer{}

	err = Copy(buffer, connection)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("GET", url.String(), nil)

	if err != nil {
		return "", err
	}

	resp, err := http.ReadResponse(bufio.NewReader(buffer), req)

	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("request have no success")
	}

	return resp.Header.Get("Lb-from"), nil
}

func TestBalancerHttpGet(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	url, err := url.Parse(baseAddress + "/api/v1/some-data")

	if err != nil {
		panic(err)
	}

	var testsM sync.Mutex
	testsLeft := HTTP_BALANCER_TESTS_AMOUNT

	wait := make(chan struct{})

	for i := 0; i < HTTP_BALANCER_TESTS_AMOUNT; i++ {
		go func() {
			connection, err := net.Dial("tcp", url.Host)

			if err != nil {
				panic(err)
			}

			lbfrom, err := GetLbfrom(url, connection)

			if err != nil {
				panic(err)
			}

			addr := connection.LocalAddr().String()
			t.Run("address: "+addr, func(t *testing.T) {
				for i := 0; i < HTTP_BALANCER_CHECKS_PER_TEST_AMOUNT; i++ {
					nextLbfrom, err := GetLbfrom(url, connection)

					if err != nil {
						t.Fatal(err)
						t.SkipNow()
					}

					if nextLbfrom != lbfrom {
						t.Fatal("Got different Lb-from for address: " + addr)
						t.SkipNow()
					}
				}
			})

			testsM.Lock()

			testsLeft -= 1

			if testsLeft == 0 {
				close(wait)
			}

			testsM.Unlock()

			connection.Close()
		}()
	}

	<-wait
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	// TODO: Реалізуйте інтеграційний тест для балансувальникка.

	for i := 0; i < 6; i++ {
		resp, err := client.Get(fmt.Sprintf("%s/api/v1/some-data", baseAddress))
		if err != nil {
			t.Error(err)
		}
		fmt.Printf("response from [%s]", resp.Header.Get("lb-from"))
		resp.Body.Close()
	}
	// t.SkipNow()

}

func BenchmarkBalancer(b *testing.B) {
	// TODO: Реалізуйте інтеграційний бенчмарк для балансувальникка.
}
