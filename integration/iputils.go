package integration

import (
	"net"
	"strings"
	"os/exec"
	"math/rand/v2"
	"slices"
	"os"
	"fmt"
	"reflect"
)

import . "github.com/magicvegetable/architecture-lab-4/err"

const (
	CurrentNetwork = "CURRENT_NET"
)

func GetLocalNetwork() (*net.IPNet, error) {
	data := os.Getenv(CurrentNetwork)

	data, _ = strings.CutSuffix(data, "\n")

	_, localNet, err := net.ParseCIDR(data)

	if err != nil {
		err = FormatError(
			err,
			"net.ParseCIDR(%#v)",
			data,
		)
	}

	return localNet, err
}

func DelCIDR(cidr, dev string) ([]byte, error) {
	args := []string{"addr", "del", cidr, "dev", dev}
	exe := "ip"

	cmd := exec.Command(exe, args...)

	res, err := cmd.Output()

	if err != nil {
		return res, FormatError(
			err,
			"exec.Command(%#v, %#v...).Output()",
			exe,
			args,
		)
	}

	return res, err
}

func AddCIDR(cidr, dev string) ([]byte, error) {
	args := []string{"addr", "add", cidr, "dev", dev}
	exe := "ip"

	cmd := exec.Command(exe, args...)

	res, err := cmd.Output()

	if err != nil {
		return res, FormatError(
			err,
			"exec.Command(%#v, %#v...).Output()",
			exe,
			args,
		)
	}

	return res, err
}

func DelAllCIDR(dev string) ([]byte, error) {
	iface, err := net.InterfaceByName(dev)

	if err != nil {
		return nil, FormatError(err, "net.InterfaceByName(%#v)", dev)
	}

	addrs, err := iface.Addrs()

	if err != nil {
		return nil, FormatError(err, "%#v.Addrs()", iface)
	}

	res := []byte{}
	for _, addr := range addrs {
		cidr := addr.String()

		subres, err := DelCIDR(cidr, dev)

		if err != nil {
			return nil, FormatError(
				err,
				"DelCIDR(%#v, %#v)",
				cidr,
				dev,
			)
		}

		res = append(res, subres...)
	}

	return res, nil
}

func ChangeCIDR(cidr, dev string) ([]byte, error) {
	_, err := DelAllCIDR(dev)

	if err != nil {
		return nil, FormatError(
			err,
			"DelAllCIDR(%#v)",
			dev,
		)
	}

	return AddCIDR(cidr, dev)
}

func InterfaceByNetwork(ipNet *net.IPNet) (*net.Interface, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		err = FormatError(err, "net.Interfaces()")
		return nil, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()

		if err != nil {
			err = FormatError(err, "%#v.Addrs()", iface)
			return nil, err
		}

		for _, addr := range addrs {
			cidr := addr.String()
			ip, _, err := net.ParseCIDR(cidr)

			if err != nil {
				err = FormatError(err, "net.ParseCIDR(%#v)", cidr)
				return nil, err
			}

			if ipNet.Contains(ip) {
				return &iface, nil
			}
		}
	}

	err = FormatError(err, "Not found")

	return nil, err
}

func RandIP(ipNet *net.IPNet) (net.IP, error) {
	if ipNet == nil {
		return nil, FormatError(
			nil,
			"ipNet have to be not nil",
		)
	}

	ones, bits := ipNet.Mask.Size()

	freeBits := bits - ones

	if freeBits == 0 {
		return ipNet.IP, nil
	}

	bitsToSet := rand.Int() % freeBits

	ipLen := len(ipNet.IP)
	ip := make([]byte, ipLen)

	copy(ip, ipNet.IP)

	for i := 0; i < bitsToSet; i++ {
		bitPos := rand.Int() % freeBits

		byteIndex := ipLen - bitPos / 8 - 1

		bit := byte(1 << (bitPos % 8))

		ip[byteIndex] |= bit
	}

	return ip, nil
}

// TODO: make real RandIPFilter
const MAX_AMOUNT_OF_TRY = 1024

func IPsContainsIP(ips []net.IP, ip net.IP) bool {
	if ips == nil {
		return false
	}

	for _, ipsIP := range ips {
		if reflect.DeepEqual(ipsIP, ip) {
			return true
		}
	}

	return false
}

func RandIPFilter(ipNet *net.IPNet, ips []net.IP) (net.IP, error) {
	for i := 0; i < MAX_AMOUNT_OF_TRY; i++ {
		ip, err := RandIP(ipNet)

		if err != nil {
			err = FormatError(err, "RandIP(%#v)", ipNet)
			return nil, err
		}

		if !IPsContainsIP(ips, ip) {
			return ip, err
		}
	}

	return nil, FormatError(
		nil,
		"Exceed amount of try %v",
		MAX_AMOUNT_OF_TRY,
	)
}

func IPtoCIDR(ip net.IP, mask net.IPMask) (string, error) {
	if ip == nil {
		return "", FormatError(nil, "ip have to be not nil")
	}

	if mask == nil {
		return "", FormatError(nil, "mask have to be not nil")
	}

	ones, _ := mask.Size()

	cidr := ip.String() + "/" + fmt.Sprintf("%v", ones)
	return cidr, nil
}

func RandCIDR(ipNet *net.IPNet) (string, error) {
	ip, err := RandIP(ipNet)

	if err != nil {
		err = FormatError(err, "RandIP(%#v)", ipNet)
		return "", err
	}

	cidr, err := IPtoCIDR(ip, ipNet.Mask)

	if err != nil {
		err = FormatError(err, "IPtoCIDR(%#v, %#v)", ip, ipNet.Mask)
		return "", err
	}

	return cidr, err
}

// TODO: make real RandCIDRFilter
func RandCIDRFilter(ipNet *net.IPNet, cidrs []string) (string, error) {
	for i := 0; i < MAX_AMOUNT_OF_TRY; i++ {
		ip, err := RandIP(ipNet)

		if err != nil {
			err = FormatError(err, "RandIP(%#v)", ipNet)
			return "", err
		}

		cidr, err := IPtoCIDR(ip, ipNet.Mask)

		if err != nil {
			err = FormatError(err, "IPtoCIDR(%#v, %#v)", ip, ipNet.Mask)
			return "", err
		}

		if !slices.Contains(cidrs, cidr) {
			return cidr, err
		}
	}

	err := FormatError(nil, "Exceed max amount of try %v", MAX_AMOUNT_OF_TRY)

	return "", err
}

func randIPNet(size int) *net.IPNet {
	maskOnes := rand.Int() % (size * 8)

	mask := make([]byte, size)

	for i := 0; i < maskOnes; i++ {
		bit := byte(1 << (7 - (i % 8)))

		byteIndex := i / 8

		mask[byteIndex] |= bit
	}

	ip := make([]byte, size)

	for i := 0; i < maskOnes; i++ {
		bitPos := rand.Int() % maskOnes

		byteIndex := bitPos / 8

		bit := byte(1 << (7 - (bitPos % 8)))

		ip[byteIndex] |= bit
	}

	return &net.IPNet{IP: ip, Mask: mask}
}

func RandIPNet() *net.IPNet {
	sizes := []int{16, 4}

	size := sizes[rand.Int() % len(sizes)]

	return randIPNet(size)
}

func RandIPNetVersion(version int) (*net.IPNet, error) {
	sizes := map[int]int{
		4: 4,
		6: 16,
	}

	size, contains := sizes[version]

	if !contains {
		err := FormatError(nil, "Not supported version %#v", version)
		return nil, err
	}

	return randIPNet(size), nil
}

func IPNetsIntersect(ipNet1 *net.IPNet, ipNet2 *net.IPNet) (bool, error) {
	if ipNet1 == nil {
		err := FormatError(nil, "ipNet1 have to be not %#v", ipNet1)
		return false, err
	}

	if ipNet2 == nil {
		err := FormatError(nil, "ipNet2 have to be not %#v", ipNet2)
		return false, err
	}

	ones1, bits1 := ipNet1.Mask.Size()

	ones2, bits2 := ipNet2.Mask.Size()

	if bits1 != bits2 {
		return false, nil
	}

	var lowestOnes int

	if ones1 > ones2 {
		lowestOnes = ones2
	} else {
		lowestOnes = ones1
	}

	for i := 0; i < lowestOnes / 8; i++ {
		if ipNet1.IP[i] != ipNet2.IP[i] {
			return false, nil
		}
	}

	byteIndex := lowestOnes / 8
	clearBits := 8 - lowestOnes % 8

	checkBits1 := ipNet1.IP[byteIndex]
	checkBits2 := ipNet2.IP[byteIndex]

	checkBits1 >>= clearBits
	checkBits1 <<= clearBits

	checkBits2 >>= clearBits
	checkBits2 <<= clearBits

	return checkBits1 == checkBits2, nil
}

func IPNetIntersectIPNets(ipNet *net.IPNet, ipNets []*net.IPNet) (bool, error) {
	if ipNet == nil {
		err := FormatError(nil, "ipNet have to be not %#v", ipNet)
		return false, err
	}

	if ipNets == nil {
		return false, nil
	}

	for _, subIPNet := range ipNets {
		intersect, err := IPNetsIntersect(subIPNet, ipNet)

		if err != nil {
			err = FormatError(err, "IPNetsIntersect(%#v, %#v)", subIPNet, ipNet)
			return false, err
		}

		if intersect {
			return intersect, err
		}
	}

	return false, nil
}

// TODO: make real RandIPNetFilterNoIntersect
func RandIPNetFilterNoIntersect(ipNets []*net.IPNet) (*net.IPNet, error) {
	for i := 0; i < MAX_AMOUNT_OF_TRY; i++ {
		randIPNet := RandIPNet()

		intersect, err := IPNetIntersectIPNets(randIPNet, ipNets)

		if err != nil {
			err = FormatError(err, "IPNetIntersectIPNets(%#v, %#v)", randIPNet, ipNets)
			return nil, err
		}

		if !intersect {
			return randIPNet, err
		}
	}

	return nil, FormatError(
		nil,
		"Exceed amount of try %v",
		MAX_AMOUNT_OF_TRY,
	)
}

// TODO: make real RandIPNetVersionFilterNoIntersect
func RandIPNetVersionFilterNoIntersect(version int, ipNets []*net.IPNet) (*net.IPNet, error) {
	for i := 0; i < MAX_AMOUNT_OF_TRY; i++ {
		randIPNet, err := RandIPNetVersion(version)

		if err != nil {
			err = FormatError(err, "RandIPNetVersion(%#v)", version)
			return nil, err
		}

		intersect, err := IPNetIntersectIPNets(randIPNet, ipNets)

		if err != nil {
			err = FormatError(err, "IPNetIntersectIPNets(%#v, %#v)", randIPNet, ipNets)
			return nil, err
		}

		if !intersect {
			return randIPNet, err
		}
	}

	return nil, FormatError(
		nil,
		"Exceed amount of try %v",
		MAX_AMOUNT_OF_TRY,
	)
}
