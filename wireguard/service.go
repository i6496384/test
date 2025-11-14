package wireguard

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/vishvananda/netlink"
	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

type Service struct {
	mu     sync.Mutex
	client *wgctrl.Client
}

func NewService() (*Service, error) {
	client, err := wgctrl.New()
	if err != nil {
		return nil, fmt.Errorf("create wgctrl client: %w", err)
	}

	return &Service{client: client}, nil
}

func (s *Service) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil
	}

	err := s.client.Close()
	s.client = nil
	return err
}

func (s *Service) Devices() ([]*wgtypes.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil, errors.New("wireguard client not initialized")
	}

	devices, err := s.client.Devices()
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	return devices, nil
}

func (s *Service) ConfigureServer(name string, privateKey string, listenPort int, replacePeers bool, peers []wgtypes.PeerConfig) error {
	if name == "" {
		return errors.New("device name is required")
	}

	if err := ensureDevice(name); err != nil {
		return err
	}

	var cfg wgtypes.Config
	cfg.ReplacePeers = replacePeers

	if privateKey != "" {
		key, err := wgtypes.ParseKey(privateKey)
		if err != nil {
			return fmt.Errorf("parse private key: %w", err)
		}
		cfg.PrivateKey = &key
	}

	if listenPort != 0 {
		cfg.ListenPort = &listenPort
	}

	cfg.Peers = peers

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return errors.New("wireguard client not initialized")
	}

	if err := s.client.ConfigureDevice(name, cfg); err != nil {
		return fmt.Errorf("configure device %s: %w", name, err)
	}
	return nil
}

func (s *Service) RemovePeer(deviceName, peerPublicKey string) error {
	if deviceName == "" {
		return errors.New("device name is required")
	}
	key, err := wgtypes.ParseKey(peerPublicKey)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	peerCfg := wgtypes.PeerConfig{
		PublicKey: key,
		Remove:    true,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return errors.New("wireguard client not initialized")
	}

	if err := s.client.ConfigureDevice(deviceName, wgtypes.Config{Peers: []wgtypes.PeerConfig{peerCfg}}); err != nil {
		return fmt.Errorf("remove peer: %w", err)
	}
	return nil
}

func (s *Service) Device(deviceName string) (*wgtypes.Device, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil, errors.New("wireguard client not initialized")
	}

	device, err := s.client.Device(deviceName)
	if err != nil {
		return nil, fmt.Errorf("get device %s: %w", deviceName, err)
	}
	return device, nil
}

func ensureDevice(name string) error {
	_, err := netlink.LinkByName(name)
	if err == nil {
		return bringLinkUp(name)
	}

	var notFound netlink.LinkNotFoundError
	if !errors.As(err, &notFound) {
		return fmt.Errorf("lookup link %s: %w", name, err)
	}

	link := &netlink.GenericLink{
		LinkAttrs: netlink.LinkAttrs{Name: name},
		LinkType:  "wireguard",
	}
	if err := netlink.LinkAdd(link); err != nil {
		return fmt.Errorf("add link %s: %w", name, err)
	}

	return bringLinkUp(name)
}

func bringLinkUp(name string) error {
	link, err := netlink.LinkByName(name)
	if err != nil {
		return fmt.Errorf("lookup link %s: %w", name, err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return fmt.Errorf("set link %s up: %w", name, err)
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
