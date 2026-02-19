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
	deviceId    string
	goveeClient *partner.GoveeClient
	color       Color
	lastColor   Color
	lastUpdate  time.Time
	lastSuccess time.Time
	isHealthy   bool
	mutex       sync.Mutex
}

// NewGoveeController creates a new Govee LED controller for the specified device.
func NewGoveeController(deviceId string, client *partner.GoveeClient) *GoveeController {
	return &GoveeController{
		deviceId:    deviceId,
		goveeClient: client,
		isHealthy:   true,
		lastSuccess: time.Now(),
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

	// Don't send updates too frequently (unless forced)
	if !force && time.Since(lastUpdate) < goveeUpdateInterval && color.Equals(lastColor) {
		return
	}

	// Skip if device ID is not configured
	if deviceId == "" {
		return
	}

	// Skip if Govee client is not enabled
	if g.goveeClient == nil {
		return
	}
	if !g.goveeClient.IsEnabled() {
		g.mutex.Lock()
		g.isHealthy = false
		g.mutex.Unlock()
		return
	}

	// Determine if we need to turn on, off, or change color
	var err error
	if color.Equals(ColorOff) {
		// Turn off with retry logic (handled by client)
		err = g.goveeClient.TurnOff(deviceId)
		if err != nil {
			log.Printf("[Govee] TurnOff failed for device %s: %v", deviceId, err)
		}
	} else {
		// First ensure device is on
		if lastColor.Equals(ColorOff) || !lastColor.Equals(color) {
			// Set color first
			err = g.goveeClient.SetColor(deviceId, color.R, color.G, color.B)
			if err != nil {
				log.Printf("[Govee] SetColor failed for device %s: %v", deviceId, err)
			} else {
				// Then turn on
				err = g.goveeClient.TurnOn(deviceId)
				if err != nil {
					log.Printf("[Govee] TurnOn failed for device %s: %v", deviceId, err)
				}
			}
		}
	}

	g.mutex.Lock()
	g.lastUpdate = time.Now()
	if err != nil {
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
