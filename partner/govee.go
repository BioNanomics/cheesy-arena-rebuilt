// Copyright 2026 Team 254. All Rights Reserved.
// Author: Integration Team
//
// Client for interfacing with Govee smart LED devices via LAN protocol.

package partner

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"golang.org/x/net/ipv4"
)

const (
	goveeMulticastAddr      = "239.255.255.250"
	goveeScanPort           = 4001
	goveeScanReplyPort      = 4002
	goveeControlPort        = 4003
	goveeDiscoveryRefreshMs = 15000
	goveeDeviceTimeoutMs    = 60000
	goveeRetryCount         = 3
	goveeRetryDelayMs       = 120
)

// GoveeDevice represents a discovered Govee LED device on the network.
type GoveeDevice struct {
	DeviceId string    // MAC address format (e.g., "31:E1:DB:E6:45:46:08:46")
	IP       string    // Current IP address
	SKU      string    // Device model
	LastSeen time.Time // For health monitoring
}

// GoveeClient manages discovery and control of Govee LED devices.
type GoveeClient struct {
	devices        map[string]*GoveeDevice
	scanSocket     *net.UDPConn
	replySocket    *net.UDPConn
	stopChan       chan bool
	mutex          sync.RWMutex
	isRunning      bool
	discoveryReady bool // Set to true after initial discovery completes
	enableLogging  bool // Controls whether diagnostic messages are logged
}

// goveeMessage represents the JSON message format for Govee LAN protocol.
type goveeMessage struct {
	Msg goveeMessageContent `json:"msg"`
}

type goveeMessageContent struct {
	Cmd  string                 `json:"cmd"`
	Data map[string]interface{} `json:"data"`
}

// NewGoveeClient creates a new Govee client instance.
func NewGoveeClient() *GoveeClient {
	return &GoveeClient{
		devices:  make(map[string]*GoveeDevice),
		stopChan: make(chan bool),
	}
}

// SetLogging enables or disables diagnostic logging.
func (c *GoveeClient) SetLogging(enabled bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.enableLogging = enabled
}

// logf logs a message if logging is enabled.
func (c *GoveeClient) logf(format string, args ...interface{}) {
	c.mutex.RLock()
	enabled := c.enableLogging
	c.mutex.RUnlock()

	if enabled {
		log.Printf(format, args...)
	}
}

// StartDiscovery begins the device discovery process.
func (c *GoveeClient) StartDiscovery() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.isRunning {
		return fmt.Errorf("discovery already running")
	}

	// Recreate stopChan to make restart-safe (prevents panic on double-close and allows restart)
	c.stopChan = make(chan bool)

	// Start listening for scan replies
	if err := c.startReplyListener(); err != nil {
		return fmt.Errorf("failed to start reply listener: %v", err)
	}

	// Start broadcasting scan requests
	if err := c.startScanBroadcaster(); err != nil {
		c.replySocket.Close()
		return fmt.Errorf("failed to start scan broadcaster: %v", err)
	}

	c.isRunning = true
	c.logf("[Govee] Device discovery started")
	return nil
}

// StopDiscovery stops the device discovery process.
func (c *GoveeClient) StopDiscovery() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.isRunning {
		return
	}

	close(c.stopChan)
	c.isRunning = false

	if c.scanSocket != nil {
		c.scanSocket.Close()
		c.scanSocket = nil
	}

	if c.replySocket != nil {
		c.replySocket.Close()
		c.replySocket = nil
	}

	c.logf("[Govee] Device discovery stopped")
}

// GetDevice retrieves a device by its ID.
func (c *GoveeClient) GetDevice(deviceId string) (*GoveeDevice, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	device, exists := c.devices[deviceId]
	if !exists {
		return nil, fmt.Errorf("device not found: %s", deviceId)
	}

	// Check if device is stale
	if time.Since(device.LastSeen) > goveeDeviceTimeoutMs*time.Millisecond {
		return nil, fmt.Errorf("device timeout: %s", deviceId)
	}

	return device, nil
}

// GetAllDevices returns a list of all discovered devices.
func (c *GoveeClient) GetAllDevices() []*GoveeDevice {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.pruneStaleDevices()

	devices := make([]*GoveeDevice, 0, len(c.devices))
	for _, device := range c.devices {
		devices = append(devices, device)
	}
	return devices
}

// TurnOn turns on a Govee device.
func (c *GoveeClient) TurnOn(deviceId string) error {
	return c.SendCommand(deviceId, "turn", map[string]interface{}{"value": 1})
}

// TurnOff turns off a Govee device with retry logic.
func (c *GoveeClient) TurnOff(deviceId string) error {
	// OFF commands use retry logic to combat packet loss
	var lastErr error
	for i := 0; i < goveeRetryCount; i++ {
		err := c.SendCommand(deviceId, "turn", map[string]interface{}{"value": 0})
		if err == nil {
			return nil
		}
		lastErr = err
		if i < goveeRetryCount-1 {
			time.Sleep(goveeRetryDelayMs * time.Millisecond)
		}
	}
	return fmt.Errorf("failed to turn off after %d retries: %v", goveeRetryCount, lastErr)
}

// SetColor sets the RGB color of a Govee device.
func (c *GoveeClient) SetColor(deviceId string, r, g, b uint8) error {
	return c.SendCommand(deviceId, "color", map[string]interface{}{
		"r": r,
		"g": g,
		"b": b,
	})
}

// SendCommand sends a command to a specific Govee device.
func (c *GoveeClient) SendCommand(deviceId, command string, data map[string]interface{}) error {
	device, err := c.GetDevice(deviceId)
	if err != nil {
		return err
	}

	msg := goveeMessage{
		Msg: goveeMessageContent{
			Cmd:  command,
			Data: data,
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %v", err)
	}

	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", device.IP, goveeControlPort))
	if err != nil {
		return fmt.Errorf("failed to resolve address: %v", err)
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %v", err)
	}
	defer conn.Close()

	_, err = conn.Write(jsonData)
	if err != nil {
		return fmt.Errorf("failed to send command: %v", err)
	}

	return nil
}

// startReplyListener starts listening for device discovery replies.
func (c *GoveeClient) startReplyListener() error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", goveeMulticastAddr, goveeScanReplyPort))
	if err != nil {
		return err
	}

	c.logf("[Govee] Starting reply listener on %s:%d", goveeMulticastAddr, goveeScanReplyPort)

	// Listen on all interfaces by binding to 0.0.0.0:port
	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("0.0.0.0:%d", goveeScanReplyPort))
	if err != nil {
		return err
	}

	conn, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		return err
	}

	c.logf("[Govee] UDP socket bound to 0.0.0.0:%d", goveeScanReplyPort)

	// Increase receive buffer to reduce packet loss on Windows
	if err := conn.SetReadBuffer(1024 * 1024); err != nil {
		c.logf("[Govee] Warning: Failed to set read buffer size: %v", err)
	}

	c.replySocket = conn

	// Join multicast group on all suitable interfaces for better Windows compatibility
	if err := c.joinMulticastOnAllInterfaces(conn, addr); err != nil {
		conn.Close()
		return fmt.Errorf("failed to join multicast group: %v", err)
	}

	// Start listening goroutine
	go c.listenForReplies()

	c.logf("[Govee] Reply listener started successfully")
	return nil
}

// joinMulticastOnAllInterfaces joins the multicast group on all suitable network interfaces.
// This is critical for Windows compatibility where automatic interface selection often fails.
func (c *GoveeClient) joinMulticastOnAllInterfaces(conn *net.UDPConn, addr *net.UDPAddr) error {
	interfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	c.logf("[Govee] Enumerating network interfaces for multicast...")

	// Wrap the UDP connection with ipv4.PacketConn for multicast control
	p := ipv4.NewPacketConn(conn)

	joinedCount := 0
	var lastErr error

	for _, iface := range interfaces {
		// Log all interfaces for debugging
		c.logf("[Govee] Found interface: %s (Flags: %v, HW: %s)", iface.Name, iface.Flags, iface.HardwareAddr)

		// Skip interfaces that are down, loopback, or don't support multicast
		if iface.Flags&net.FlagUp == 0 {
			c.logf("[Govee]   Skipping %s: interface is down", iface.Name)
			continue
		}
		if iface.Flags&net.FlagLoopback != 0 {
			c.logf("[Govee]   Skipping %s: loopback interface", iface.Name)
			continue
		}
		if iface.Flags&net.FlagMulticast == 0 {
			c.logf("[Govee]   Skipping %s: no multicast support", iface.Name)
			continue
		}

		// Get interface addresses for additional debugging
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			c.logf("[Govee]   Interface %s has address: %s", iface.Name, a.String())
		}

		// Try to join the multicast group on this interface
		err := p.JoinGroup(&iface, &net.UDPAddr{IP: addr.IP})
		if err != nil {
			c.logf("[Govee] ERROR: Failed to join multicast group on interface %s: %v", iface.Name, err)
			lastErr = err
			continue
		}

		c.logf("[Govee] SUCCESS: Joined multicast group %s on interface %s (%s)", addr.IP, iface.Name, iface.HardwareAddr)
		joinedCount++
	}

	if joinedCount == 0 {
		if lastErr != nil {
			return fmt.Errorf("failed to join multicast group on any interface: %v", lastErr)
		}
		return fmt.Errorf("no suitable network interfaces found for multicast")
	}

	c.logf("[Govee] Successfully joined multicast group on %d interface(s)", joinedCount)
	return nil
}

// listenForReplies processes incoming device discovery replies.
func (c *GoveeClient) listenForReplies() {
	buffer := make([]byte, 1024)
	c.logf("[Govee] Reply listener goroutine started, waiting for packets...")

	packetCount := 0
	for {
		select {
		case <-c.stopChan:
			c.logf("[Govee] Reply listener stopping (received %d packets total)", packetCount)
			return
		default:
			c.replySocket.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, remoteAddr, err := c.replySocket.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				c.logf("[Govee] Error reading reply: %v", err)
				continue
			}

			packetCount++
			c.logf("[Govee] Received packet #%d (%d bytes) from %s", packetCount, n, remoteAddr)

			var msg goveeMessage
			if err := json.Unmarshal(buffer[:n], &msg); err != nil {
				c.logf("[Govee] Failed to parse JSON from packet: %v", err)
				continue
			}

			c.logf("[Govee] Parsed message: cmd=%s", msg.Msg.Cmd)

			if msg.Msg.Cmd != "scan" {
				c.logf("[Govee] Ignoring non-scan message: %s", msg.Msg.Cmd)
				continue
			}

			deviceId, ok := msg.Msg.Data["device"].(string)
			if !ok {
				c.logf("[Govee] Scan reply missing device ID")
				continue
			}

			ip, ok := msg.Msg.Data["ip"].(string)
			if !ok {
				c.logf("[Govee] Scan reply missing IP address")
				continue
			}

			sku, _ := msg.Msg.Data["sku"].(string)

			c.logf("[Govee] Discovered device: %s (%s @ %s)", deviceId, sku, ip)

			c.mutex.Lock()
			c.devices[deviceId] = &GoveeDevice{
				DeviceId: deviceId,
				IP:       ip,
				SKU:      sku,
				LastSeen: time.Now(),
			}
			c.mutex.Unlock()

			log.Printf("[Govee] Discovered device: %s (%s @ %s)", deviceId, sku, ip)
		}
	}
}

// startScanBroadcaster starts broadcasting device discovery requests.
func (c *GoveeClient) startScanBroadcaster() error {
	addr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%d", goveeMulticastAddr, goveeScanPort))
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp4", nil, addr)
	if err != nil {
		return err
	}

	// Set multicast TTL and loop for better compatibility
	p := ipv4.NewPacketConn(conn)
	if err := p.SetMulticastTTL(2); err != nil {
		c.logf("[Govee] Warning: Failed to set multicast TTL: %v", err)
	}
	if err := p.SetMulticastLoopback(true); err != nil {
		c.logf("[Govee] Warning: Failed to set multicast loopback: %v", err)
	}

	c.scanSocket = conn

	c.logf("[Govee] Scan broadcaster started, will send to %s", addr)

	// Start broadcasting goroutine
	go c.broadcastScans()

	return nil
}

// broadcastScans periodically sends device discovery requests.
func (c *GoveeClient) broadcastScans() {
	ticker := time.NewTicker(goveeDiscoveryRefreshMs * time.Millisecond)
	defer ticker.Stop()

	// Send initial scan
	c.sendScanRequest()

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			c.sendScanRequest()
		}
	}
}

// sendScanRequest sends a single device discovery request.
func (c *GoveeClient) sendScanRequest() {
	msg := goveeMessage{
		Msg: goveeMessageContent{
			Cmd: "scan",
			Data: map[string]interface{}{
				"account_topic": "reserve",
			},
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		c.logf("[Govee] Failed to marshal scan request: %v", err)
		return
	}

	if c.scanSocket != nil {
		n, err := c.scanSocket.Write(jsonData)
		if err != nil {
			c.logf("[Govee] ERROR: Failed to send scan request: %v", err)
		} else {
			c.logf("[Govee] Sent scan request (%d bytes) to %s", n, c.scanSocket.RemoteAddr())
		}
	} else {
		c.logf("[Govee] ERROR: Scan socket is nil, cannot send scan request")
	}
}

// pruneStaleDevices removes devices that haven't been seen recently.
func (c *GoveeClient) pruneStaleDevices() {
	cutoff := time.Now().Add(-goveeDeviceTimeoutMs * time.Millisecond)
	for id, device := range c.devices {
		if device.LastSeen.Before(cutoff) {
			delete(c.devices, id)
		}
	}
}

// IsEnabled returns whether the Govee client is enabled (running).
func (c *GoveeClient) IsEnabled() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.isRunning
}

// IsDiscoveryReady returns whether initial discovery has completed.
func (c *GoveeClient) IsDiscoveryReady() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.discoveryReady
}

// SetDiscoveryReady marks discovery as ready.
func (c *GoveeClient) SetDiscoveryReady() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.discoveryReady = true
}
