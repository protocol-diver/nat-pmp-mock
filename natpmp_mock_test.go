package mocknat

import (
	"net"
	"testing"

	natpmp "github.com/jackpal/go-nat-pmp"
)

func TestExternalIP(t *testing.T) {
	nat := New(net.IPv4(127, 0, 0, 1), net.IPv4(1, 1, 1, 1), true)
	defer nat.Close()
	nat.Run()

	client := natpmp.NewClient(net.IPv4(127, 0, 0, 1))
	result, err := client.GetExternalAddress()
	if err != nil {
		t.Fatal(err)
	}

	got := net.IP(result.ExternalIPAddress[:])
	if !got.Equal(nat.externalIP) {
		t.Fatalf("invalid GetExternalAddress, want: %v, got: %v", nat.externalIP, got)
	}
}

func TestAddPortMapping(t *testing.T) {
	nat := New(net.IPv4(127, 0, 0, 1), net.IPv4(1, 1, 1, 1), true)
	defer nat.Close()
	nat.Run()

	var result *natpmp.AddPortMappingResult
	var err error

	client := natpmp.NewClient(net.IPv4(127, 0, 0, 1))
	result, err = client.AddPortMapping("udp", 2000, 5000, 30)
	if err != nil {
		t.Fatal(err)
	}
	_ = result

	if internal := nat.Map("udp", 5000); internal == nil {
		t.Fatal("")
	}
	if internal := nat.Map("udp", 5001); internal != nil {
		t.Fatal("")
	}

}
