package wireguard

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/curve25519"
)

var ErrUnavailable = errors.New("wireguard tools are not available")

type Key [32]byte

func GeneratePrivateKey() (Key, error) {
	var k Key
	if _, err := rand.Read(k[:]); err != nil {
		return Key{}, fmt.Errorf("generate private key: %w", err)
	}
	k[0] &= 248
	k[31] &= 127
	k[31] |= 64
	return k, nil
}

func ParseKey(value string) (Key, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Key{}, errors.New("empty key")
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return Key{}, fmt.Errorf("decode key: %w", err)
	}
	if len(raw) != 32 {
		return Key{}, fmt.Errorf("key has invalid size: %d", len(raw))
	}
	var k Key
	copy(k[:], raw)
	return k, nil
}

func (k Key) String() string {
	return base64.StdEncoding.EncodeToString(k[:])
}

func (k Key) PublicKey() Key {
	var out [32]byte
	curve25519.ScalarBaseMult(&out, (*[32]byte)(&k))
	return Key(out)
}

func (k Key) IsZero() bool {
	return k == (Key{})
}

type Device struct {
	Name          string
	ListenPort    int
	PrivateKey    Key
	PublicKey     Key
	HasPrivateKey bool
	HasPublicKey  bool
	Peers         []Peer
}

type Peer struct {
	PublicKey         Key
	HasPublicKey      bool
	AllowedIPs        []net.IPNet
	LastHandshakeTime time.Time
}

type PeerConfig struct {
	PublicKey                   Key
	Remove                      bool
	ReplaceAllowedIPs           bool
	AllowedIPs                  []net.IPNet
	PersistentKeepaliveInterval *time.Duration
}

type Service struct {
	mu sync.Mutex
}

func NewService() (*Service, error) {
	return &Service{}, nil
}

func (s *Service) Close() error {
	return nil
}

func (s *Service) Devices() ([]*Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	output, err := runCommand("wg", "show", "all", "dump")
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			return nil, ErrUnavailable
		}
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			if cmdErr.ExitCode == 1 {
				return []*Device{}, nil
			}
		}
		return nil, err
	}
	return parseDump(output)
}

func (s *Service) Device(name string) (*Device, error) {
	if name == "" {
		return nil, errors.New("device name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	output, err := runCommand("wg", "show", name, "dump")
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			return nil, ErrUnavailable
		}
		return nil, err
	}

	devices, err := parseDump(output)
	if err != nil {
		return nil, err
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("device %s not found", name)
	}
	return devices[0], nil
}

func (s *Service) ConfigureServer(name, privateKey string, listenPort int, replacePeers bool, peers []PeerConfig) error {
	if name == "" {
		return errors.New("device name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ensureDevice(name); err != nil {
		return err
	}

	if privateKey != "" {
		if err := setPrivateKey(name, privateKey); err != nil {
			return err
		}
	}

	if listenPort != 0 {
		if _, err := runCommand("wg", "set", name, "listen-port", strconv.Itoa(listenPort)); err != nil {
			return err
		}
	}

	if replacePeers {
		if err := clearPeers(name); err != nil {
			return err
		}
	}

	for _, peer := range peers {
		if err := configurePeer(name, peer); err != nil {
			return err
		}
	}

	return bringLinkUp(name)
}

func (s *Service) RemovePeer(deviceName, peerPublicKey string) error {
	if deviceName == "" {
		return errors.New("device name is required")
	}
	if peerPublicKey == "" {
		return errors.New("peer public key is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := runCommand("wg", "set", deviceName, "peer", peerPublicKey, "remove"); err != nil {
		return err
	}
	return nil
}

func ParseAllowedIPs(allowedIPs []string) ([]net.IPNet, error) {
	result := make([]net.IPNet, 0, len(allowedIPs))
	for _, cidr := range allowedIPs {
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			cidr += "/32"
		}
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("parse allowed ip %s: %w", cidr, err)
		}
		result = append(result, *network)
	}
	return result, nil
}

func AllocateAddress(cidr string, used map[string]struct{}) (string, error) {
	if cidr == "" {
		return "", errors.New("network is required")
	}
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("parse cidr: %w", err)
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		return "", errors.New("only IPv4 networks are supported")
	}
	ones, bits := network.Mask.Size()
	hostBits := bits - ones
	if hostBits <= 1 {
		return "", errors.New("network is too small")
	}

	base := binary.BigEndian.Uint32(ipv4)
	totalHosts := uint64(1) << uint64(hostBits)
	for host := uint64(2); host < totalHosts-1; host++ {
		candidate := make(net.IP, len(ipv4))
		binary.BigEndian.PutUint32(candidate, base+uint32(host))
		if !network.Contains(candidate) {
			continue
		}
		addr := candidate.String()
		if _, exists := used[addr]; exists {
			continue
		}
		return addr, nil
	}

	return "", errors.New("no available addresses in network")
}

type CommandError struct {
	Command  string
	Args     []string
	Output   string
	ExitCode int
	Err      error
}

func (e *CommandError) Error() string {
	builder := strings.Builder{}
	builder.WriteString(e.Command)
	if len(e.Args) > 0 {
		builder.WriteString(" ")
		builder.WriteString(strings.Join(e.Args, " "))
	}
	if e.Output != "" {
		builder.WriteString(": ")
		builder.WriteString(e.Output)
	}
	if e.Err != nil {
		builder.WriteString(": ")
		builder.WriteString(e.Err.Error())
	}
	return builder.String()
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

func runCommand(cmd string, args ...string) ([]byte, error) {
	command := exec.Command(cmd, args...)
	output, err := command.CombinedOutput()
	if err == nil {
		return output, nil
	}

	var execErr *exec.Error
	if errors.As(err, &execErr) && execErr.Err == exec.ErrNotFound {
		return nil, ErrUnavailable
	}

	var exitErr *exec.ExitError
	exitCode := -1
	if errors.As(err, &exitErr) {
		exitCode = exitErr.ExitCode()
	}

	return nil, &CommandError{
		Command:  cmd,
		Args:     args,
		Output:   strings.TrimSpace(string(output)),
		ExitCode: exitCode,
		Err:      err,
	}
}

func ensureDevice(name string) error {
	if name == "" {
		return errors.New("device name is required")
	}

	if _, err := runCommand("ip", "link", "show", "dev", name); err == nil {
		return bringLinkUp(name)
	} else if errors.Is(err, ErrUnavailable) {
		return err
	} else {
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) {
			if cmdErr.ExitCode != 1 {
				return err
			}
		} else {
			return err
		}
	}

	if _, err := runCommand("ip", "link", "add", "dev", name, "type", "wireguard"); err != nil {
		return err
	}
	return bringLinkUp(name)
}

func bringLinkUp(name string) error {
	if _, err := runCommand("ip", "link", "set", "dev", name, "up"); err != nil {
		return err
	}
	return nil
}

func setPrivateKey(name, key string) error {
	file, err := os.CreateTemp("", "wg-key-")
	if err != nil {
		return fmt.Errorf("create temp file for key: %w", err)
	}
	defer os.Remove(file.Name())

	if _, err := file.WriteString(strings.TrimSpace(key)); err != nil {
		file.Close()
		return fmt.Errorf("write private key: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close private key file: %w", err)
	}

	if _, err := runCommand("wg", "set", name, "private-key", file.Name()); err != nil {
		return err
	}
	return nil
}

func configurePeer(name string, cfg PeerConfig) error {
	if cfg.PublicKey.IsZero() {
		return errors.New("peer public key is required")
	}

	args := []string{"set", name, "peer", cfg.PublicKey.String()}
	if cfg.Remove {
		args = append(args, "remove")
		_, err := runCommand("wg", args...)
		return err
	}

	if cfg.ReplaceAllowedIPs && len(cfg.AllowedIPs) > 0 {
		ips := make([]string, 0, len(cfg.AllowedIPs))
		for _, ipNet := range cfg.AllowedIPs {
			ips = append(ips, ipNet.String())
		}
		args = append(args, "allowed-ips", strings.Join(ips, ","))
	}

	if cfg.PersistentKeepaliveInterval != nil {
		secs := int(cfg.PersistentKeepaliveInterval.Round(time.Second) / time.Second)
		if secs < 0 {
			secs = 0
		}
		args = append(args, "persistent-keepalive", strconv.Itoa(secs))
	}

	_, err := runCommand("wg", args...)
	return err
}

func clearPeers(name string) error {
	output, err := runCommand("wg", "show", name, "dump")
	if err != nil {
		if errors.Is(err, ErrUnavailable) {
			return err
		}
		var cmdErr *CommandError
		if errors.As(err, &cmdErr) && cmdErr.ExitCode == 1 {
			return nil
		}
		return err
	}

	devices, err := parseDump(output)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return nil
	}
	for _, peer := range devices[0].Peers {
		if !peer.HasPublicKey {
			continue
		}
		if _, err := runCommand("wg", "set", name, "peer", peer.PublicKey.String(), "remove"); err != nil {
			return err
		}
	}
	return nil
}

func parseDump(data []byte) ([]*Device, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var devices []*Device
	var current *Device

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) == 0 {
			continue
		}

		switch fields[0] {
		case "interface":
			if len(fields) < 2 {
				continue
			}
			device := &Device{Name: fields[1]}
			if len(fields) > 2 && fields[2] != "" && fields[2] != "(none)" {
				if key, err := ParseKey(fields[2]); err == nil {
					device.PrivateKey = key
					device.HasPrivateKey = true
				}
			}
			if len(fields) > 3 && fields[3] != "" && fields[3] != "(none)" {
				if key, err := ParseKey(fields[3]); err == nil {
					device.PublicKey = key
					device.HasPublicKey = true
				}
			}
			if len(fields) > 4 {
				if port, err := strconv.Atoi(fields[4]); err == nil {
					device.ListenPort = port
				}
			}
			devices = append(devices, device)
			current = device

		case "peer":
			if current == nil {
				continue
			}
			if len(fields) < 2 {
				continue
			}
			peer := Peer{}
			if key, err := ParseKey(fields[1]); err == nil {
				peer.PublicKey = key
				peer.HasPublicKey = true
			}

			if len(fields) > 4 && fields[4] != "" && fields[4] != "(none)" {
				parts := strings.Split(fields[4], ",")
				for _, part := range parts {
					cidr := strings.TrimSpace(part)
					if cidr == "" {
						continue
					}
					_, network, err := net.ParseCIDR(cidr)
					if err != nil {
						continue
					}
					peer.AllowedIPs = append(peer.AllowedIPs, *network)
				}
			}

			if len(fields) > 5 {
				if sec, err := strconv.ParseInt(fields[5], 10, 64); err == nil && sec > 0 {
					peer.LastHandshakeTime = time.Unix(sec, 0)
				}
			}

			current.Peers = append(current.Peers, peer)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return devices, nil
}
