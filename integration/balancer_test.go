package integration

import . "github.com/magicvegetable/architecture-lab-4/err"
import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"
	"bytes"
	"io"
	"bufio"
	"net"
	"sync"
)

const (
	ReadDeadline = 10 * time.Millisecond
	HttpBalancerTestsAmount = 3
	HttpBalancerChecksPerTestAmount = 3
	CopyBufferSize = 128
	AmountOfChangeIP = 3
)

var	BaseAddress = "http://balancer:8090"

func Copy(out io.Writer, connection net.Conn) error {
	size := CopyBufferSize
	buffer := make([]byte, size)

	for {
		deadline := time.Now().Add(ReadDeadline)
		connection.SetReadDeadline(deadline)
		_, err := connection.Read(buffer)

		if err != nil {
			if err != io.EOF && time.Now().Before(deadline) {
				return FormatError(
					err,
					"%v.Read(%v)",
					connection,
					buffer,
				)
			}

			break;
		}

		_, err = out.Write(buffer)

		if err != nil {
			return FormatError(
				err,
				"%v.Write(%v)",
				out,
				buffer,
			)
		}

		buffer = make([]byte, size)
	}

	return nil
}

func ReadResponse(url *url.URL, connection net.Conn) (*http.Response, error) {
	buffer := &bytes.Buffer{}

	err := Copy(buffer, connection)

	if err != nil {
		return nil, FormatError(
			err,
			"Copy(%v, %v)",
			buffer,
			connection,
		)
	}

	req, err := http.NewRequest("GET", url.String(), nil)

	if err != nil {
		return nil, FormatError(
			err,
			"http.NewRequest(\"GET\", %v, nil)",
			url.String(),
		)
	}

	resp, err := http.ReadResponse(bufio.NewReader(buffer), req)

	if err != nil {
		return nil, FormatError(
			err,
			"http.ReadResponse(%v, %v)",
			bufio.NewReader(buffer),
			req,
		)
	}

	return resp, nil
}

func GetLbfrom(url *url.URL, connection net.Conn) (string, error) {
	requestStr := "GET " + url.Path + " HTTP/1.1\r\n" + "Host: " + url.Hostname() + "\r\n\r\n"
	_, err := connection.Write([]byte(requestStr))

	if err != nil {
		return "", FormatError(
			err,
			"%v.Write([]byte(%v))",
			connection,
			requestStr,
		)
	}

	resp, err := ReadResponse(url, connection)

	if err != nil {
		return "", FormatError(
			err,
			"ReadResponse(%v, %v)",
			url,
			connection,
		)
	}

	if resp.StatusCode != 200 {
		return "", FormatError(
			fmt.Errorf("%v", requestStr),
			"http.Response.StatusCode == %v for request",
			resp.StatusCode,
		)
	}

	return resp.Header.Get("Lb-from"), nil
}

func balancerHttpGetTest(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	urlStr := BaseAddress + "/api/v1/some-data"
	url, err := url.Parse(urlStr)

	if err != nil {
		err = FormatError(err, "url.Parse(%v)", urlStr)
		panic(err)
	}

	var testsM sync.Mutex
	testsLeft := HttpBalancerTestsAmount

	wait := make(chan struct{})

	for i := 0; i < HttpBalancerTestsAmount; i++ {
		go func() {
			network := "tcp"
			connection, err := net.Dial(network, url.Host)

			if err != nil {
				err = FormatError(err, "net.Dial(%#v, %#v)", network, url.Host)
				panic(err)
			}

			lbfrom, err := GetLbfrom(url, connection)

			if err != nil {
				err = FormatError(err, "GetLbfrom(%#v, %#v)", url, connection)
				panic(err)
			}

			addr := connection.LocalAddr().String()
			t.Run("address: " + addr, func(t *testing.T) {
				for i := 0; i < HttpBalancerChecksPerTestAmount; i++ {
					nextLbfrom, err := GetLbfrom(url, connection)

					if err != nil {
						err = FormatError(err, "GetLbfrom(%v, %v)", url, connection)
						panic(err)
					}

					if nextLbfrom != lbfrom {
						err = FormatError(
							fmt.Errorf("Expected: %v\nGot: %v", lbfrom, nextLbfrom),
							"GetLbfrom(%v, %v)",
							url, 
							connection,
						)

						t.Error(err)
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

func BenchmarkBalancer(b *testing.B) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		b.Skip("Integration test is not enabled")
	}

	client := http.Client{Timeout: 3 * time.Second}

	urlBalancer := fmt.Sprintf("%s/api/v1/some-data", BaseAddress)
	for i := 0; i < b.N; i++ {
		resp, err := client.Get(urlBalancer)
		if err != nil {
			err = FormatError(
				err,
				"%#v.Get(%#v)",
				client,
				urlBalancer,
			)

			b.Error(err)
		}

		if resp.StatusCode != 200 {
			bodyBytes, err := io.ReadAll(resp.Request.Body)

			if err != nil {
				err = FormatError(
					err,
					"io.ReadAll(%#v)",
					resp.Request.Body,
				)
				panic(err)
			}

			err = FormatError(
				fmt.Errorf("%#v", string(bodyBytes)),
				"%#v.StatusCode != %#v for request",
				resp,
				resp.StatusCode,
			)
			b.Error(err)
		}

		resp.Body.Close()
	}
}

func GetBalancerIP(ipNet *net.IPNet) (net.IP, error) {
	addrs, err := net.LookupHost("balancer")

	if err != nil {
		return nil, FormatError(
			err,
			"net.LookupHost(%#v)",
			"balancer",
		)
	}

	for _, addr := range addrs {
		ip := net.ParseIP(addr)

		if ipNet.Contains(ip) {
			return ip, nil
		}
	}

	err = FormatError(err, "Not balancer ip for network %#v", ipNet)

	return nil, err
}

func TestBalancer(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	ipNet, err := GetLocalNetwork()

	if err != nil {
		err = FormatError(err, "GetLocalNetwork()")
		panic(err)
	}

	iface, err := InterfaceByNetwork(ipNet)

	if err != nil {
		err = FormatError(err, "InterfaceByNetwork(%#v)", ipNet)
		panic(err)
	}

	balancerIP, err := GetBalancerIP(ipNet)

	if err != nil {
		err = FormatError(err, "GetBalancerIP()")
		panic(err)
	}

	filterIPs := []net.IP{balancerIP, ipNet.IP}
	for i := 0; i < AmountOfChangeIP; i++ {
		// TODO: fix ipv6 change addr
		if balancerIP.To4() == nil {
			t.Run("ipv6 ", balancerHttpGetTest)
			continue
		}

		ip, err := RandIPFilter(ipNet, filterIPs)

		if err != nil {
			err = FormatError(err, "RandIPFilter(%#v, %#v)", ipNet, filterIPs)
			panic(err)
		}

		cidr, err := IPtoCIDR(ip, ipNet.Mask)

		if err != nil {
			err = FormatError(err, "IPtoCIDR(%#v, %#v)", ip, ipNet.Mask)
			panic(err)
		}

		_, err = ChangeCIDR(cidr, iface.Name)

		if err != nil {
			err = FormatError(err, "ChangeIpAddr(%#v, %#v)", cidr, iface.Name)
			panic(err)
		}

		t.Run("CIDR: " + cidr, balancerHttpGetTest)
	}
}

