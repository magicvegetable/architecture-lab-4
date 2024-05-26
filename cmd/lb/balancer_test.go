package main

import . "github.com/magicvegetable/architecture-lab-4/err"
import . "github.com/magicvegetable/architecture-lab-4/integration"
import (
	"testing"
	"fmt"
	"math/rand"
	"time"
	"slices"
)

const (
	maxIPPartValue = uint64(255)
	maxPort = uint64(65535)
	GetAvailableServerTestsAmount = 100
	GetAvailableServerTestsChecksAmount = 100

	maxRandStrSize = uint64(100)
	HashTestsAmount = 100
	HashTestsChecksAmount = 100
)

func randStr() string {
	size := rand.Uint64() % (maxRandStrSize + 1)
	str := ""

	for i := uint64(0); i < size; i++ {
		str += fmt.Sprintf("%c", rand.Int())
	}

	return str
}

func TestHash(t *testing.T) {
	for i := 0; i < HashTestsAmount; i++ {
		str := randStr()

		h1 := hash(str)
		for i := 0; i < HashTestsChecksAmount; i++ {
			h2 := hash(str)

			if h1 != h2 {
				err := fmt.Errorf("Expected: %v\nGot: %v", h1, h2)
				err = FormatError(err, "Wrong hash")
				t.Error(err)
			}
		}
	}
}

func TestGetAvailableServer(t *testing.T) {
	for i := 0; i < GetAvailableServerTestsAmount; i++ {
		ipNet := RandIPNet()
		ip, err := RandIP(ipNet)

		if err != nil {
			err = FormatError(err, "RandIP(%#v)", ipNet)
			panic(err)
		}

		addr := ip.String() + fmt.Sprintf("%v", rand.Uint64() % (maxPort + 1))

		s1 := GetAvailableServer(addr)
		for i := 0; i < GetAvailableServerTestsChecksAmount; i++ {
			s2 := GetAvailableServer(addr)

			if s1 != s2 {
				err := fmt.Errorf("Expected: %v\nGot: %v", s1, s2)
				err = FormatError(err, "Wrong server")
				t.Error(err)
			}
		}
	}
}

var aliveServers = append([]string{}, ServersPool...)

func killServer(server string) {
	serverI := slices.Index(aliveServers, server)

	if serverI == -1 {
		panic(fmt.Errorf("%v", aliveServers))
	}

	newServersPool := aliveServers[:serverI]
	newServersPool = append(newServersPool, aliveServers[serverI + 1:]...)

	aliveServers = newServersPool
}

func resurrectServer(server string) {
	if slices.Contains(aliveServers, server) {
		return
	}

	aliveServers = append(aliveServers, server)
}

func HealthMock(dst string) bool {
	return slices.Contains(aliveServers, dst)
}

func TestBalancer(t *testing.T) {
	allServers := append([]string{}, ServersPool...)

	TestServersPoolStateInterval := 10 * time.Millisecond
	CheckServerHealthInterval = 1 * time.Millisecond

	MonitorServers(HealthMock)
	
	for _, server := range allServers {
		t.Run("kill " + server, func(t *testing.T) {
			killServer(server)
			time.Sleep(TestServersPoolStateInterval)

			if slices.Contains(ServersPool, server) {
				err := FormatError(nil, "Dead server (%v) in ServersPool (%v)", server, ServersPool)
				t.Error(err)
			}
		})
	}

	for _, server := range allServers {
		t.Run("resurrect " + server, func(t *testing.T) {
			resurrectServer(server)
			time.Sleep(TestServersPoolStateInterval)

			if !slices.Contains(ServersPool, server) {
				err := FormatError(nil, "No Alive server (%v) in ServersPool (%v)", server, ServersPool)
				t.Error(err)
			}
		})
	}
}

