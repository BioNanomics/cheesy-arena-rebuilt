// Copyright 2025 Team 254. All Rights Reserved.
// Author: kwaremburg
//
// LED controller interface and implementations for controlling RGB light bars in the hubs.

package led

import (
	"fmt"
	"log"
	"net"
	"time"
)

const (
	sACNPort          = 5568
	sACNSourceName    = "Cheesy Arena"
	sourceName        = "Cheesy Arena"
	pixelDataOffset   = 126
	heartbeatInterval = 1 * time.Second
)

// Color represents an RGB color value.
type Color struct {
	R, G, B uint8
}

func (c Color) Equals(other Color) bool {
	return c.R == other.R && c.G == other.G && c.B == other.B
}

// Predefined colors for different states
var (
	ColorOff    = Color{0, 0, 0}       // Off (inactive hub)
	ColorGreen  = Color{0, 255, 0}     // Green (field safe)
	ColorPurple = Color{128, 0, 128}   // Purple (counting after match)
	ColorRed    = Color{255, 0, 0}     // Red (red alliance active hub)
	ColorBlue   = Color{0, 0, 255}     // Blue (blue alliance active hub)
	ColorWhite  = Color{255, 255, 255} // White (for chase animations)
)

// LedController defines the interface for controlling LED lighting systems.
// Implementations include DMX/sACN controllers and Govee smart LED controllers.
type LedController interface {
	// SetAddress configures the controller's target address (IP for DMX, device ID for Govee)
	SetAddress(address string) error

	// SetColor sets the desired color for the LED (solid color mode)
	SetColor(color Color)

	// GetColor returns the current color setting
	GetColor() Color

	// SetPixels sets individual pixel colors (pixel-level control mode)
	// Only supported by controllers that return true for SupportsPixelControl()
	SetPixels(pixels []Color)

	// SupportsPixelControl returns whether this controller supports per-pixel color control
	SupportsPixelControl() bool

	// Update sends the current color to the physical device
	Update()

	// Close releases any resources held by the controller
	Close()

	// IsHealthy returns whether the controller is functioning properly
	IsHealthy() bool
}

const (
	// Number of pixels per LED strip for pixel-level control
	NumPixelsPerStrip = 50
)

// DmxController implements the LedController interface for DMX/sACN E1.31 over Ethernet.
// This was previously named "Controller" and provides backward compatibility.
type DmxController struct {
	Address      string
	Universe     int
	conn         net.Conn
	color        Color
	lastColor    Color
	pixels       []Color // Pixel array for pixel-level control
	lastPixels   []Color // Last sent pixel array
	usePixelMode bool    // True if using pixel-level control, false for solid color
	lastSend     time.Time
	StartChannel int // Starting channel for the hub
	packet       []byte
}

func (dmx *DmxController) SetAddress(address string) error {
	if dmx.conn != nil {
		dmx.conn.Close()
		dmx.conn = nil
	}

	dmx.Address = address
	if address != "" {
		var err error
		if dmx.conn, err = net.Dial("udp4", net.JoinHostPort(address, fmt.Sprintf("%d", sACNPort))); err != nil {
			return err
		}
	}

	return nil
}

func (dmx *DmxController) SetColor(color Color) {
	dmx.color = color
	dmx.usePixelMode = false // Switch to solid color mode
}

func (dmx *DmxController) GetColor() Color {
	return dmx.color
}

func (dmx *DmxController) SetPixels(pixels []Color) {
	dmx.pixels = pixels
	dmx.usePixelMode = true // Switch to pixel mode
}

func (dmx *DmxController) SupportsPixelControl() bool {
	return true // DMX supports per-pixel control
}

func (dmx *DmxController) Close() {
	if dmx.conn != nil {
		dmx.conn.Close()
		dmx.conn = nil
	}
}

func (dmx *DmxController) IsHealthy() bool {
	// DMX controller is healthy if it has a connection or is not configured
	return dmx.conn != nil || dmx.Address == ""
}

func (dmx *DmxController) Update() {
	if dmx.conn == nil {
		// This controller is not configured; do nothing.
		return
	}

	if dmx.usePixelMode {
		// Pixel-level control mode
		pixels := dmx.pixels

		// Create the template packet if it doesn't already exist or if size changed
		numChannels := len(pixels) * 3
		if len(dmx.packet) == 0 || len(dmx.packet) != pixelDataOffset+numChannels+3 {
			dmx.packet = createBlankPacket(numChannels)
			dmx.lastPixels = make([]Color, len(pixels))
		}

		// Send packets if the pixel values have changed
		if dmx.shouldSendPixelPacket(pixels) {
			dmx.populatePixelPacket(pixels, dmx.StartChannel)
			if err := dmx.sendPacket(dmx.Universe); err != nil {
				log.Printf("sACN error writing data to universe %d: %v", dmx.Universe, err)
				return
			}
			copy(dmx.lastPixels, pixels)
			dmx.lastSend = time.Now()
		}
	} else {
		// Solid color mode
		color := dmx.color

		// Create the template packet if it doesn't already exist
		if len(dmx.packet) == 0 {
			dmx.packet = createBlankPacket(3)
		}

		// Send packets if the pixel values have changed
		if dmx.shouldSendPacket(color) {
			dmx.populatePacket(color, dmx.StartChannel)
			if err := dmx.sendPacket(dmx.Universe); err != nil {
				log.Printf("sACN error writing data to universe %d: %v", dmx.Universe, err)
				return
			}
			dmx.lastColor = color
			dmx.lastSend = time.Now()
		}
	}
}

func (dmx *DmxController) shouldSendPacket(color Color) bool {
	if !color.Equals(dmx.lastColor) {
		return true
	}
	return time.Since(dmx.lastSend) >= heartbeatInterval
}

func (dmx *DmxController) shouldSendPixelPacket(pixels []Color) bool {
	// Check if pixel array changed
	if len(pixels) != len(dmx.lastPixels) {
		return true
	}
	for i := range pixels {
		if !pixels[i].Equals(dmx.lastPixels[i]) {
			return true
		}
	}
	return time.Since(dmx.lastSend) >= heartbeatInterval
}

func (dmx *DmxController) populatePacket(color Color, startChannel int) {
	// Clear DMX data area
	for i := pixelDataOffset; i < len(dmx.packet); i++ {
		dmx.packet[i] = 0
	}

	// Light mapping (3 channels):
	// 1: Red
	// 2: Green
	// 3: Blue
	dmx.packet[pixelDataOffset+startChannel-1+0] = color.R
	dmx.packet[pixelDataOffset+startChannel-1+1] = color.G
	dmx.packet[pixelDataOffset+startChannel-1+2] = color.B
}

func (dmx *DmxController) populatePixelPacket(pixels []Color, startChannel int) {
	// Clear DMX data area
	for i := pixelDataOffset; i < len(dmx.packet); i++ {
		dmx.packet[i] = 0
	}

	// Populate each pixel (3 channels per pixel: R, G, B)
	for pixelIndex, color := range pixels {
		channelOffset := pixelDataOffset + startChannel - 1 + (pixelIndex * 3)
		dmx.packet[channelOffset+0] = color.R
		dmx.packet[channelOffset+1] = color.G
		dmx.packet[channelOffset+2] = color.B
	}
}

func (dmx *DmxController) sendPacket(universe int) error {
	dmx.packet[111]++ // Sequence number
	dmx.packet[113] = byte(universe >> 8)
	dmx.packet[114] = byte(universe & 0xff)
	_, err := dmx.conn.Write(dmx.packet)
	return err
}

// Controller is an alias for DmxController for backward compatibility.
// Deprecated: Use DmxController directly.
type Controller = DmxController

func putFlagsLength(pkt []byte, off int, pduLen int) {
	fl := 0x7000 | (pduLen & 0x0FFF)
	pkt[off] = byte(fl >> 8)
	pkt[off+1] = byte(fl)
}

func createBlankPacket(numChannels int) []byte {
	size := pixelDataOffset + numChannels + 3
	packet := make([]byte, size)

	// Preamble size
	packet[0] = 0x00
	packet[1] = 0x10

	// Postamble size
	packet[2] = 0x00
	packet[3] = 0x00

	// ACN packet identifier
	// copy(packet[4:16], []byte("ACN-E1.17\x00\x00\x00"))

	// ACN packet identifier
	packet[4] = 0x41
	packet[5] = 0x53
	packet[6] = 0x43
	packet[7] = 0x2d
	packet[8] = 0x45
	packet[9] = 0x31
	packet[10] = 0x2e
	packet[11] = 0x31
	packet[12] = 0x37
	packet[13] = 0x00
	packet[14] = 0x00
	packet[15] = 0x00

	// Root PDU length and flags
	// rootPduLength := size - 16
	// packet[16] = 0x70 | byte(rootPduLength>>8)
	// packet[17] = byte(rootPduLength & 0xff)

	rootPduLength := size - 16
	putFlagsLength(packet, 16, rootPduLength)

	// E1.31 vector indicating that this is a data packet
	packet[21] = 0x04

	// CID
	// copy(packet[22:38], []byte("CheesyArena-LEDs"))

	copy(
		packet[22:38], []byte{
			0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0,
			0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		},
	)

	// Framing PDU length and flags
	// framingPduLength := size - 38
	// packet[38] = 0x70 | byte(framingPduLength>>8)
	// packet[39] = byte(framingPduLength & 0xff)

	framingPduLength := size - 38
	putFlagsLength(packet, 38, framingPduLength)

	// E1.31 vector indicating that this is a data packet
	packet[43] = 0x02

	// Source name
	copy(packet[44:108], []byte(sACNSourceName))

	// Priority
	packet[108] = 100

	// Reserved
	packet[109] = 0x00
	packet[110] = 0x00

	// Sequence number
	packet[111] = 0x00

	// Options flags
	packet[112] = 0x00

	// DMX universe
	packet[113] = 0x00
	packet[114] = 0x00

	// DMP layer PDU length
	// dmpPduLength := size - 115
	// packet[115] = 0x70 | byte(dmpPduLength>>8)
	// packet[116] = byte(dmpPduLength & 0xff)

	dmpPduLength := size - 115
	putFlagsLength(packet, 115, dmpPduLength)

	// E1.31 vector indicating set property
	packet[117] = 0x02

	// Address and data type
	packet[118] = 0xa1

	// First property address
	packet[119] = 0x00
	packet[120] = 0x00

	// Address increment
	packet[121] = 0x00
	packet[122] = 0x01

	// Property value count
	count := 1 + numChannels + 3 // Extra 3 channels to avoid malformed packet errors in wireshark.
	packet[123] = byte(count >> 8)
	packet[124] = byte(count & 0xff)

	// DMX start code
	packet[125] = 0

	return packet
}
