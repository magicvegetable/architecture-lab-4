package TestLauncher

import . "github.com/magicvegetable/architecture-lab-4/err"
import . "github.com/magicvegetable/architecture-lab-4/integration"
import (
	"os/exec"
	"os"
	"strings"
	"testing"
	"io/fs"
	"io"
	"net"
	"log"
)

// TODO:
// figure out why container doesn't launch
// from some ip

const (
	randomCIDRTestAmount = 10
)

func killContainers(containers []string) error {
	exe := "docker"
	for _, container := range containers {
		args := []string{"kill", "--signal", "KILL", container}

		cmd := exec.Command(exe, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()

		if err != nil {
			err = FormatError(
				err,
				"exec.Command(%#v, %#v).Run()",
				exe,
				args,
			)
			return err
		}
	}

	return nil
}

func getContainersInNet(nt string) ([]string, error) {
	exe := "docker"
	args := []string{"ps", "--filter", "network=" + nt, "-q"}

	cmd := exec.Command(exe, args...)
	cmd.Stderr = os.Stderr

	out, err := cmd.Output()

	if err != nil {
		err = FormatError(
			err,
			"exec.Command(%#v, %#v...).Run()",
			exe,
			args,
		)
	}

	var containers []string

	if len(out) != 0 {
		containers = strings.Split(string(out), "\n")

		lastI := len(containers) - 1
		if containers[lastI] == "" {
			containers = containers[:lastI]
		}
	}

	return containers, err
}

func cleanDockerNets() {
	localNets := []string{"architecture-lab-4_servers", "architecture-lab-4_testlan"}

	exe := "docker"

	for _, localNet := range localNets {
		containers, err := getContainersInNet(localNet)

		if err != nil {
			err = FormatError(
				err,
				"getContainersInNet(%#v)",
				localNet,
			)

			panic(err)
		}

		if len(containers) != 0 {
			err = killContainers(containers)

			if err != nil {
				err = FormatError(
					err,
					"killContainers(%#v)",
					containers,
				)

				panic(err)
			}
		}

		args := []string{"network", "rm", localNet}
		cmd := exec.Command(exe, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		_ = cmd.Run()
	}
}

func runTest() error {
	args := []string{
		"compose",
		"-f",
		"docker-compose.yaml",
		"-f",
		"docker-compose.test.yaml",
		"up",
		"--force-recreate",
		"--exit-code-from",
		"test",
	}

	exe := "docker"
	cmd := exec.Command(exe, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		err = FormatError(
			err,
			"exec.Command(%#v, %#v...).Output()",
			exe,
			args,
		)

		return err
	}

	return nil
}

func LoadInfo(cidr string) error {
	ip, _, err := net.ParseCIDR(cidr)

	if err != nil {
		err = FormatError(err, "net.ParseCIDR(%#v)", cidr)
		return err
	}

	templatePath := "templates/docker-compose.test.yaml"

	templateFile, err := os.Open(templatePath)

	if err != nil {
		err = FormatError(err, "os.Open(%#v)", templatePath)
		return err
	}

	templateBytes, err := io.ReadAll(templateFile)

	if err != nil {
		err = FormatError(err, "io.ReplaceAll(%#v)", templateFile)
		return err
	}

	templateFile.Close()

	content := string(templateBytes)

	content = strings.ReplaceAll(content, "@CURRENT_NET@", cidr)

	if ip.To4() != nil {
		content = strings.ReplaceAll(content, "@ENABLE_IPv6@", "false")
	} else {
		content = strings.ReplaceAll(content, "@ENABLE_IPv6@", "true")
	}

	testPath := "docker-compose.test.yaml"

	flag := os.O_WRONLY|os.O_CREATE|os.O_TRUNC
	perm := fs.FileMode(0644)

	testFile, err := os.OpenFile(testPath, flag, perm)

	if err != nil {
		err = FormatError(err, "os.OpenFile(%#v, %#v, %#v)", testPath, flag, perm)
		return err
	}

	contentBytes := []byte(content)

	_, err = testFile.Write(contentBytes)

	if err != nil {
		err = FormatError(err, "%#v.Write(%#v)", testFile, contentBytes)
		return err
	}

	testFile.Close()

	return nil
}

func TestNetworks(t *testing.T) {
	if _, exists := os.LookupEnv("INTEGRATION_TEST"); !exists {
		t.Skip("Integration test is not enabled")
	}

	cidrs := []string{
		"2085:0DAA::/112",
		"3333:0D:1234:f8f8::/64",
		"192.13.0.0/16",
		"12.13.0.0/16",
		"192.13.0.0/16",
		"56.13.0.0/16",
		"192.18.0.0/16",
	}

	for _, cidr := range cidrs {
		t.Run("cidr: " + cidr, func(t *testing.T) {
			err := LoadInfo(cidr)

			if err != nil {
				err = FormatError(err, "LoadInfo(%#v)", cidr)
				panic(err)
			}

			cleanDockerNets()

			err = runTest()

			if err != nil {
				err = FormatError(err, "runTest()")
				t.Error(err)
			}
		})
	}

	reservedCIDRs := []string{
		"2001:0DB8::/120",
		"127.0.0.0/8",
		"::1/128",
		"172.17.0.0/16",
	}

	var reservedIPNets []*net.IPNet

	for _, reservedCIDR := range reservedCIDRs {
		_, ipNet, err := net.ParseCIDR(reservedCIDR)

		if err != nil {
			err = FormatError(err, "net.ParseCIDR(%#v)", reservedCIDR)
			panic(err)
		}

		reservedIPNets = append(reservedIPNets, ipNet)
	}

	for i := 0; i < randomCIDRTestAmount; i++ {
		ipNet, err := RandIPNetFilterNoIntersect(reservedIPNets)

		if err != nil {
			err = FormatError(err, "RandIPNetFilterNoIntersect(%#v)", reservedIPNets)
			panic(err)
		}

		ones, bits := ipNet.Mask.Size()

		if bits - ones < 3 {
			i -= 1
			continue
		}

		cidr := ipNet.String()

		t.Run("cidr: " + cidr, func(t *testing.T) {
			err := LoadInfo(cidr)

			if err != nil {
				err = FormatError(err, "LoadInfo(%#v)", cidr)
				panic(err)
			}

			cleanDockerNets()

			err = runTest()

			if err != nil {
				err = FormatError(err, "runTest()")
				log.Println(err)
			}
		})
	}
}

