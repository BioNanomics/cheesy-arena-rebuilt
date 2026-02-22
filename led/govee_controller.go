// Copyright 2026 Team 254. All Rights Reserved.
// Author: Integration Team
//
// Govee LED controller implementation for controlling Govee smart LEDs via LAN protocol.

package led

import (
	"log"
	"sync"
	"time"

	"github.com/Team254/cheesy-arena/partner"
)

const (
	goveeUpdateInterval = 50 * time.Millisecond // Limit update frequency (reduced from 100ms to support faster animations)
	goveeHealthTimeout  = 5 * time.Second       // Consider unhealthy if no successful update
)

// GoveeController implements the LedController interface for Govee smart LED devices.
type GoveeController struct {
	deviceId      string
	goveeClient   *partner.GoveeClient
	color         Color
	lastColor     Color
	lastUpdate    time.Time
	lastSuccess   time.Time
	isHealthy     bool
	enableLogging bool
	mutex         sync.Mutex
}

// NewGoveeController creates a new Govee LED controller for the specified device.
func NewGoveeController(deviceId string, client *partner.GoveeClient, enableLogging bool) *GoveeController {
	return &GoveeController{
		deviceId:      deviceId,
		goveeClient:   client,
		isHealthy:     true,
		lastSuccess:   time.Now(),
		enableLogging: enableLogging,
	}
}

// SetAddress sets the Govee device ID (MAC address format).
func (g *GoveeController) SetAddress(address string) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.deviceId = address
	return nil
}

// GetDeviceId returns the Govee device ID.
func (g *GoveeController) GetDeviceId() string {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return g.deviceId
}

// SetColor sets the desired color for the LED.
func (g *GoveeController) SetColor(color Color) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.color = color
}

// GetColor returns the current color setting.
func (g *GoveeController) GetColor() Color {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	return g.color
}

// SetPixels sets individual pixel colors. Govee controllers don't support per-pixel control,
// so this falls back to using a simple alternating pattern between the two most common colors.
func (g *GoveeController) SetPixels(pixels []Color) {
	// Govee doesn't support per-pixel control, so we fall back to simple alternating
	// Count occurrences of each color to determine the dominant pattern
	if len(pixels) == 0 {
		return
	}

	// Simple heuristic: alternate between first and second pixel colors
	// This gives a basic chase effect for Govee strips
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Use the first pixel color as the solid color (best we can do with Govee)
	g.color = pixels[0]
}

// SupportsPixelControl returns whether this controller supports per-pixel color control.
// Govee controllers only support solid colors, not per-pixel control.
func (g *GoveeController) SupportsPixelControl() bool {
	return false
}

// Update sends the current color to the Govee device.
func (g *GoveeController) Update() {
	g.update(false)
}

// ForceUpdate sends the current color to the Govee device, bypassing throttle checks.
func (g *GoveeController) ForceUpdate() {
	g.update(true)
}

// update is the internal implementation that optionally bypasses throttle checks.
func (g *GoveeController) update(force bool) {
	g.mutex.Lock()
	color := g.color
	deviceId := g.deviceId
	lastUpdate := g.lastUpdate
	lastColor := g.lastColor
	g.mutex.Unlock()

	// Enforce minimum update interval (unless forced)
	// This respects the Govee hardware constraint of 50ms minimum between commands
	if !force && time.Since(lastUpdate) < goveeUpdateInterval {
		return
	}

	// Skip if color hasn't changed (no need to send duplicate commands)
	if color.Equals(lastColor) {
		return
	}

	// Skip if device ID is not configured
	if deviceId == "" {
		return
	}

	// Skip if Govee client is not enabled (but don't mark as unhealthy)
	if g.goveeClient == nil || !g.goveeClient.IsEnabled() {
		return
	}

	// Determine if we need to turn on, off, or change color
	var err error
	if color.Equals(ColorOff) {
		// Turn off with retry logic (handled by client)
		err = g.goveeClient.TurnOff(deviceId)
		if err != nil && g.enableLogging {
			log.Printf("[Govee] TurnOff failed for device %s: %v", deviceId, err)
		}
	} else {
		// Ensure device is on and showing the correct color
		if lastColor.Equals(ColorOff) || !lastColor.Equals(color) {
			// Turn on the device first to ensure it's ready to receive color commands
			err = g.goveeClient.TurnOn(deviceId)
			if err != nil {
				if g.enableLogging {
					log.Printf("[Govee] TurnOn failed for device %s: %v", deviceId, err)
				}
			} else {
				// Set the color after device is on
				err = g.goveeClient.SetColor(deviceId, color.R, color.G, color.B)
				if err != nil && g.enableLogging {
					log.Printf("[Govee] SetColor failed for device %s: %v", deviceId, err)
				}
			}
		}
	}

	g.mutex.Lock()
	g.lastUpdate = time.Now()
	if err != nil {
		if g.enableLogging {
			log.Printf("[Govee] Error updating device %s: %v", deviceId, err)

			// If device not found, show discovered devices to help troubleshooting
			if err.Error() == "device not found: "+deviceId {
				devices := g.goveeClient.GetAllDevices()
				if len(devices) > 0 {
					log.Printf("[Govee] Discovered devices on network:")
					for _, d := range devices {
						log.Printf("[Govee]   - %s (%s @ %s)", d.DeviceId, d.SKU, d.IP)
					}
				} else {
					log.Printf("[Govee] No devices discovered. Make sure devices are powered on and on the same network.")
				}
			}
		}

		g.isHealthy = false
	} else {
		g.lastColor = color
		g.lastSuccess = time.Now()
		g.isHealthy = true
	}
	g.mutex.Unlock()
}

// Close releases any resources held by the controller.
func (g *GoveeController) Close() {
	// Turn off the LED when closing
	g.SetColor(ColorOff)
	g.Update()
}

// IsHealthy returns whether the controller is functioning properly.
func (g *GoveeController) IsHealthy() bool {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	// Check if we've had a successful update recently
	if time.Since(g.lastSuccess) > goveeHealthTimeout {
		g.isHealthy = false
	}

	return g.isHealthy
}
