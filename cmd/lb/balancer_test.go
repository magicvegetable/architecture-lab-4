package main

import "testing"
import "fmt"
import "math/rand"

const (
	maxIPPartValue                      = uint64(255)
	maxPort                             = uint64(65535)
	GetAvailableServerTestsAmount       = 100
	GetAvailableServerTestsChecksAmount = 100

	maxRandStrSize        = uint64(100)
	HashTestsAmount       = 100
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
				t.Fatalf(
					"Got different values from hash:\nh1: %v\nh2: %v\n",
					h1,
					h2,
				)
				t.SkipNow()
			}
		}
	}
}

func TestGetAvailableServer(t *testing.T) {
	for i := 0; i < GetAvailableServerTestsAmount; i++ {
		addr := fmt.Sprintf(
			"%v.%v.%v.%v:%v",
			rand.Uint64()%(maxIPPartValue+1),
			rand.Uint64()%(maxIPPartValue+1),
			rand.Uint64()%(maxIPPartValue+1),
			rand.Uint64()%(maxIPPartValue+1),
			rand.Uint64()%(maxPort+1),
		)

		s1 := GetAvailableServer(addr)
		for i := 0; i < GetAvailableServerTestsChecksAmount; i++ {
			s2 := GetAvailableServer(addr)

			if s1 != s2 {
				t.Fatalf(
					"Got different servers for same address:\ns1: %v\ns2: %v\n",
					s1,
					s2,
				)
				t.SkipNow()
			}
		}
	}
}

func TestBalancer(t *testing.T) {
	// s := GetAvailableServer("123.123.132.132")
	// t.Log(s)
	// t.Log(s)
	// TODO: Реалізуйте юніт-тест для балансувальникка.
}
