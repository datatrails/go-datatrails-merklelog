package snowflakeid

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"net"
)

var (
	ErrBadWorkerCIDR = errors.New("provided worker CIDR is invalid")
	ErrBadPodIP      = errors.New("pod ip invalid")
	ErrMaskRange     = errors.New("the specified CIDR mask allows for to many or too few private ip addresses")
)

const (
	// MaxWorkerBits allowed per worker id or sequence counter
	MaxWorkerBits = 24
	// MinWorkerBits allowed per worker id or sequence counter
	MinWorkerBits = 8
)

// workerIDSequenceBits returns the worker id and the number of bits permitted
// for the id sequence counter.
func workerIDSequenceBits(cfg Config) (uint16, int, error) {

	mask, err := parseMask(cfg.WorkerCIDR)
	if err != nil {
		return 0, 0, err
	}
	ip, err := parseIP(cfg.PodIP)
	if err != nil {
		return 0, 0, err
	}

	workerIDBits := bits.Len16(binary.BigEndian.Uint16(mask[2:]))

	sequenceBits := MaxWorkerBits - workerIDBits
	masked := ip.Mask(mask)
	id := binary.BigEndian.Uint16(masked[2:])

	return id, sequenceBits, nil
}

// parseMask parses the CIDR mask which configures how many bits to allocate to
// the worker id from the pod private ip address.  It errors if the
// configuration exceeds the assumptions of the id generator.
func parseMask(workerCIDR string) (net.IPMask, error) {
	_, ipNet, err := net.ParseCIDR(workerCIDR)
	if err != nil {
		return nil, fmt.Errorf("%s - issue parsing CIDR: %w", workerCIDR, err)
	}

	mask := invertIPMask(ipNet.Mask)
	if mask[0] != 0 || mask[1] != 0 {
		return nil, fmt.Errorf("%s - allows to many ips: %w", workerCIDR, ErrMaskRange)
	}
	if mask[2] == 0 && mask[3] < 255 {
		return nil, fmt.Errorf("%s - allows to few ips: %w", workerCIDR, ErrMaskRange)
	}
	return mask, nil
}

// invertIPMask  inverts the mask in place and also returns it
func invertIPMask(mask net.IPMask) net.IPMask {
	for i := range 4 {
		mask[i] = ^mask[i]
	}
	return mask
}

// parseIP parses a pod ip address and requires that it is allocated from a known private ip range.
func parseIP(podIP string) (net.IP, error) {
	ip := net.ParseIP(podIP)
	if ip == nil {
		return nil, fmt.Errorf("%s - issue parsing IP: %w", podIP, ErrBadPodIP)
	}
	if !ip.IsPrivate() {
		return nil, fmt.Errorf("%s - is not a private ip: %w", podIP, ErrBadPodIP)
	}
	return ip, nil
}
