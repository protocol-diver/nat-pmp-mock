package mocknat

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

const (
	extenralAddressOpcode = 0 + iota
	udpMappingOpcode
	tcpMappingOpcode

	unsupportedVersion = 1
	unsupportedOpcode  = 5
)

type mockNAT struct {
	conn       *net.UDPConn
	listenAddr *net.UDPAddr
	externalIP net.IP
	epoch      uint32

	// flag; 0: negative, 1: positive
	supportedPMP uint32
	isRun        uint32

	mu      sync.Mutex
	mapping map[string]map[uint16]*Internal // Mapping protocol to external port
}

// Internal stores the internal port number and the expiration timer
// according to the lifetime.
type Internal struct {
	Port  uint16
	timer *time.Timer
}

func New(listenIP net.IP, externalIP net.IP, supportPMP bool) *mockNAT {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", listenIP.String(), 5351))
	if err != nil {
		panic(err)
	}
	nat := &mockNAT{
		conn:       nil,
		listenAddr: addr,
		externalIP: externalIP,
		mapping:    makeProtocolMap(),
	}
	if supportPMP {
		nat.supportedPMP = 1
	}
	return nat
}

// Clear goroutines about lifetime, and also reset 'Seconds Since Start of Epoch'
func (p *mockNAT) Restart() {
	p.mu.Lock()
	for k := range p.mapping {
		for _, v := range p.mapping[k] {
			v.timer.Stop()
		}
	}
	p.mapping = makeProtocolMap()
	p.mu.Unlock()

	atomic.StoreUint32(&p.epoch, 0)
}

// Since calling this, the corresponding NAT-MOCK server supports PMP
func (p *mockNAT) SupportPMP() {
	atomic.StoreUint32(&p.supportedPMP, 1)
}

// Since calling this, the corresponding NAT-MOCK server does not support PMP.
func (p *mockNAT) UnsupportPMP() {
	atomic.StoreUint32(&p.supportedPMP, 0)
}

func (p *mockNAT) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

func (p *mockNAT) ExternalIP() net.IP {
	return p.externalIP
}

func (p *mockNAT) Close() error {
	if atomic.LoadUint32(&p.isRun) == 0 {
		return errors.New("already closed")
	}
	atomic.StoreUint32(&p.isRun, 0)
	return p.conn.Close()
}

func (p *mockNAT) Map(protocol string, extport uint16) *Internal {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.mapping[protocol][extport]
}

func (p *mockNAT) Epoch() uint32 {
	return atomic.LoadUint32(&p.epoch)
}

func (p *mockNAT) Run() {
	if atomic.LoadUint32(&p.isRun) == 1 {
		return
	}
	var err error
	p.conn, err = net.ListenUDP("udp", p.listenAddr)
	if err != nil {
		panic(err)
	}

	atomic.StoreUint32(&p.isRun, 1)
	go p.run()
}

func (p *mockNAT) run() {
	// Count Seconds Since Start of Epoch.
	go func() {
		for {
			time.Sleep(time.Millisecond)

			// Check state is running
			if atomic.LoadUint32(&p.isRun) == 0 {
				return
			}
			atomic.AddUint32(&p.epoch, 1)
		}
	}()

	for {
		// Check state is running
		if atomic.LoadUint32(&p.isRun) == 0 {
			return
		}
		// Read process
		b := make([]byte, 12)
		len, sender, err := p.conn.ReadFromUDP(b)
		if err != nil {
			continue
		}
		// Just ignore if NAT doesn't support PMP.
		if atomic.LoadUint32(&p.supportedPMP) == 0 {
			// OR reply 'ICMP Port unreachable'
			continue
		}
		if len < 2 {
			continue
		}
		// Allowing loopback IPs is not a specification. But user usually
		// use the loopback IP for testing. So this mock NAT allows loopback.
		if !sender.IP.IsPrivate() && !sender.IP.IsLoopback() {
			continue
		}
		// Invalid request format.
		if b[1] > 128 {
			continue
		}
		p.handle(b, sender)
	}
}

func (p *mockNAT) handle(b []byte, sender *net.UDPAddr) {
	response := make([]byte, 16)
	response[0] = 0

	defer func() {
		// Set Seconds Since Start of Epoch if the message is successed.
		if binary.BigEndian.Uint16(response[2:4]) == 0 {
			epoch := atomic.LoadUint32(&p.epoch)
			binary.BigEndian.PutUint32(response[4:8], epoch)
		}
		p.conn.WriteToUDP(response, sender)
	}()

	// Only accept version 0.
	if b[0] != 0 {
		binary.BigEndian.PutUint16(response[2:4], unsupportedVersion)
		return
	}

	switch b[1] {
	case extenralAddressOpcode:
		response[1] = 128 + extenralAddressOpcode
		copy(response[8:12], p.externalIP.To4())
		response = response[0:12]

	case udpMappingOpcode:
		rop, rcode, body := p.handleMappingOpcode(b, "udp", udpMappingOpcode)
		response[1] = rop
		copy(response[2:4], rcode[:])
		copy(response[8:16], body[:])

	case tcpMappingOpcode:
		rop, rcode, body := p.handleMappingOpcode(b, "tcp", tcpMappingOpcode)
		response[1] = rop
		copy(response[2:4], rcode[:])
		copy(response[8:16], body[:])

	default:
		binary.BigEndian.PutUint16(response[2:4], unsupportedOpcode)
	}
}

func (p *mockNAT) handleMappingOpcode(b []byte, protocol string, opcode byte) (ropcode byte, rcode [2]byte, body [8]byte) {
	ropcode = 128 + opcode

	intport, extport, lifeTime := parseMappingRequest(b)

	p.mu.Lock()
	defer p.mu.Unlock()
	v, exist := p.mapping[protocol][extport]

	// Destroying a mapping
	if lifeTime == 0 && extport == 0 {
		if exist {
			v.timer.Stop()
			delete(p.mapping[protocol], extport)
		}
	} else {
		if exist {
			// This is renewal request. Reset the timer to lifetime.
			if v.Port == intport {
				if !v.timer.Stop() {
					<-v.timer.C
				}
				v.timer.Reset(time.Duration(lifeTime) * time.Second)
			} else {
				// This is the case that already used request's port.
				//
				// Maps alternative port.
				extport = p.suggestExternalPort(protocol)
				p.addExternal(protocol, extport, intport, time.Duration(lifeTime)*time.Second)
			}
		}
		// Add new mapping.
		if !exist {
			// Client would prefer to have a high-numbered "anonymous" external port assigned
			if extport == 0 {
				extport = p.suggestExternalPort(protocol)
			}
			p.addExternal(protocol, extport, intport, time.Duration(lifeTime)*time.Second)
		}
	}
	binary.BigEndian.PutUint16(body[0:2], intport)
	binary.BigEndian.PutUint16(body[2:4], extport)
	binary.BigEndian.PutUint32(body[4:8], lifeTime)
	return
}

// addExternal stores the new internal port number and runs the
// lifetime expiration goroutine.
//
// The caller must hold p.mu.
func (p *mockNAT) addExternal(protocol string, extport, intport uint16, duration time.Duration) {
	e := &Internal{
		Port:  intport,
		timer: time.NewTimer(duration),
	}

	p.mapping[protocol][extport] = e

	go func(t *time.Timer) {
		defer t.Stop()
		for range t.C {
			p.mu.Lock()
			delete(p.mapping[protocol], extport)
			p.mu.Unlock()
			return
		}
	}(e.timer)
}

// suggestExternalPort returns a random port number. Originally
// it should respond with an 'Out of resources' error if it
// expected all port numbers to be in use, but currently it simply
// causes a panic.
//
// The caller must hold p.mu.
func (p *mockNAT) suggestExternalPort(protocol string) uint16 {
	for i := uint16(1024); i < 65535; i++ {
		if _, ok := p.mapping[protocol][i]; !ok {
			return i
		}
	}
	panic("Out of resources")
}

func parseMappingRequest(b []byte) (uint16, uint16, uint32) {
	return binary.BigEndian.Uint16(b[4:6]), binary.BigEndian.Uint16(b[6:8]), binary.BigEndian.Uint32(b[8:12])
}

func makeProtocolMap() map[string]map[uint16]*Internal {
	m := make(map[string]map[uint16]*Internal)
	m["tcp"] = make(map[uint16]*Internal)
	m["udp"] = make(map[uint16]*Internal)
	return m
}
