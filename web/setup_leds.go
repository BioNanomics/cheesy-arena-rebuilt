// Copyright 2026 Team 254. All Rights Reserved.
// Author: Integration Team
//
// Web routes for configuring LED controllers (DMX and Govee).

package web

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/Team254/cheesy-arena/led"
	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/websocket"
)

// Shows the LED configuration page.
func (web *Web) ledsGetHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	template, err := web.parseFiles("templates/setup_leds.html", "templates/base.html")
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Get current LED controller configuration
	settings := web.arena.EventSettings
	controllerType := settings.LedControllerType
	if controllerType == "" {
		controllerType = "dmx"
	}

	data := struct {
		*model.EventSettings
		ControllerType string
		RedHealthy     bool
		BlueHealthy    bool
	}{
		EventSettings:  settings,
		ControllerType: controllerType,
		RedHealthy:     web.arena.RedHubLeds.IsHealthy(),
		BlueHealthy:    web.arena.BlueHubLeds.IsHealthy(),
	}

	err = template.ExecuteTemplate(w, "base", data)
	if err != nil {
		handleWebErr(w, err)
		return
	}
}

// Saves the LED configuration to the database and reinitializes controllers.
func (web *Web) ledsPostHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	settings := web.arena.EventSettings

	// Update settings from form
	settings.LedControllerType = r.PostFormValue("ledControllerType")
	settings.DMXAddress = r.PostFormValue("dmxAddress")
	settings.RedLedAddress = r.PostFormValue("redLedAddress")
	settings.BlueLedAddress = r.PostFormValue("blueLedAddress")

	// Save to database
	err := web.arena.Database.UpdateEventSettings(settings)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Reinitialize LED controllers
	err = web.arena.InitializeLedControllers()
	if err != nil {
		log.Printf("Warning: Failed to reinitialize LED controllers: %v", err)
	}

	http.Redirect(w, r, "/setup/leds", http.StatusSeeOther)
}

// Returns a list of discovered Govee devices as JSON.
func (web *Web) ledsDiscoverHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	if web.arena.GoveeClient == nil {
		http.Error(w, "Govee client not initialized", http.StatusInternalServerError)
		return
	}

	devices := web.arena.GoveeClient.GetAllDevices()

	// Convert to JSON-friendly format
	type DeviceInfo struct {
		DeviceId string `json:"deviceId"`
		IP       string `json:"ip"`
		SKU      string `json:"sku"`
	}

	deviceList := make([]DeviceInfo, 0, len(devices))
	for _, device := range devices {
		deviceList = append(deviceList, DeviceInfo{
			DeviceId: device.DeviceId,
			IP:       device.IP,
			SKU:      device.SKU,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(deviceList)
}

// Tests an LED controller by setting it to a specific color.
func (web *Web) ledsTestHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	hub := r.PostFormValue("hub")        // "red" or "blue"
	colorStr := r.PostFormValue("color") // "red", "blue", "green", "purple", "off"

	log.Printf("[LED Test] Request received: hub=%s, color=%s", hub, colorStr)

	var controller led.LedController
	if hub == "red" {
		controller = web.arena.RedHubLeds
	} else if hub == "blue" {
		controller = web.arena.BlueHubLeds
	} else {
		log.Printf("[LED Test] Invalid hub: %s", hub)
		http.Error(w, "Invalid hub", http.StatusBadRequest)
		return
	}

	var color led.Color
	switch colorStr {
	case "red":
		color = led.ColorRed
	case "blue":
		color = led.ColorBlue
	case "green":
		color = led.ColorGreen
	case "purple":
		color = led.ColorPurple
	case "off":
		color = led.ColorOff
	default:
		log.Printf("[LED Test] Invalid color: %s", colorStr)
		http.Error(w, "Invalid color", http.StatusBadRequest)
		return
	}

	// For Govee controllers, wait for discovery to complete before testing
	if web.arena.EventSettings.LedControllerType == "govee" && web.arena.GoveeClient != nil {
		log.Printf("[LED Test] Waiting for discovery to be ready...")
		// Wait up to 5 seconds for discovery to be ready
		timeout := time.After(5 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for !web.arena.GoveeClient.IsDiscoveryReady() {
			select {
			case <-timeout:
				log.Printf("[LED Test] Discovery timeout - not ready after 5 seconds")
				http.Error(w, "Discovery not ready. Please wait a few seconds and try again.", http.StatusServiceUnavailable)
				return
			case <-ticker.C:
				// Continue waiting
			}
		}
		log.Printf("[LED Test] Discovery is ready")
	}

	log.Printf("[LED Test] Setting color and calling Update()")
	log.Printf("[LED Test] Controller type: %T", controller)
	log.Printf("[LED Test] Color to set: RGB(%d,%d,%d)", color.R, color.G, color.B)

	// Enable test mode to prevent arena loop from overwriting LED colors
	web.arena.SetLedTestMode(true)
	defer func() {
		web.arena.SetLedTestMode(false)
		log.Printf("[LED Test] Test mode disabled")
	}()

	// Wait for arena loop to see the flag (arena loop runs every 10ms)
	time.Sleep(20 * time.Millisecond)
	log.Printf("[LED Test] Test mode enabled, arena loop should be paused")

	// For Govee controllers, send commands directly to the Govee client to bypass the controller entirely
	// This avoids race conditions with the arena loop
	if web.arena.EventSettings.LedControllerType == "govee" {
		goveeController, ok := controller.(*led.GoveeController)
		if !ok {
			log.Printf("[LED Test] Error: controller is not a GoveeController")
			http.Error(w, "Invalid controller type", http.StatusInternalServerError)
			return
		}

		deviceId := goveeController.GetDeviceId()
		if deviceId == "" {
			log.Printf("[LED Test] Error: device ID not configured for %s hub", hub)
			http.Error(w, "Device ID not configured", http.StatusBadRequest)
			return
		}

		// Send the test color directly to the Govee client, bypassing the controller
		log.Printf("[LED Test] Sending test commands directly to Govee client for device %s", deviceId)
		for i := 0; i < 5; i++ {
			if color.Equals(led.ColorOff) {
				err := web.arena.GoveeClient.TurnOff(deviceId)
				if err != nil {
					log.Printf("[LED Test] TurnOff failed: %v", err)
				} else {
					log.Printf("[LED Test] TurnOff sent successfully")
				}
			} else {
				// Set color first
				err := web.arena.GoveeClient.SetColor(deviceId, color.R, color.G, color.B)
				if err != nil {
					log.Printf("[LED Test] SetColor failed: %v", err)
				} else {
					log.Printf("[LED Test] SetColor sent successfully")
					// Then turn on
					err = web.arena.GoveeClient.TurnOn(deviceId)
					if err != nil {
						log.Printf("[LED Test] TurnOn failed: %v", err)
					} else {
						log.Printf("[LED Test] TurnOn sent successfully")
					}
				}
			}
			time.Sleep(200 * time.Millisecond)
		}
		log.Printf("[LED Test] Sent 5 test commands directly to Govee client")

		// Keep the test color displayed for 2 seconds before allowing arena loop to resume
		time.Sleep(2 * time.Second)
	} else {
		controller.SetColor(color)
		controller.Update()

		// Keep the test color displayed for 2 seconds
		time.Sleep(2 * time.Second)
	}

	log.Printf("[LED Test] Test complete")
	w.WriteHeader(http.StatusOK)
}

// The websocket endpoint for sending realtime updates to the LED setup page.
func (web *Web) ledsWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	if !web.userIsAdmin(w, r) {
		return
	}

	ws, err := websocket.NewWebsocket(w, r)
	if err != nil {
		handleWebErr(w, err)
		return
	}
	defer ws.Close()

	// Subscribe to LED health notifications
	go ws.HandleNotifiers(web.arena.HubLedNotifier)

	// Loop, waiting for commands and responding to them, until the client closes the connection.
	for {
		messageType, data, err := ws.Read()
		if err != nil {
			if err == io.EOF {
				// Client has closed the connection; nothing to do here.
				return
			}
			log.Println(err)
			return
		}

		switch messageType {
		case "discover":
			// Trigger device discovery
			if web.arena.GoveeClient != nil && web.arena.GoveeClient.IsEnabled() {
				devices := web.arena.GoveeClient.GetAllDevices()
				ws.Write("devices", devices)
			} else {
				ws.WriteError("Govee client not enabled")
			}
		case "testLed":
			// Test LED command
			testData, ok := data.(map[string]interface{})
			if !ok {
				ws.WriteError("Invalid test data")
				continue
			}
			hub, _ := testData["hub"].(string)
			colorStr, _ := testData["color"].(string)

			var controller led.LedController
			if hub == "red" {
				controller = web.arena.RedHubLeds
			} else if hub == "blue" {
				controller = web.arena.BlueHubLeds
			} else {
				ws.WriteError("Invalid hub")
				continue
			}

			var color led.Color
			switch colorStr {
			case "red":
				color = led.ColorRed
			case "blue":
				color = led.ColorBlue
			case "green":
				color = led.ColorGreen
			case "purple":
				color = led.ColorPurple
			case "off":
				color = led.ColorOff
			default:
				ws.WriteError("Invalid color")
				continue
			}

			controller.SetColor(color)
			controller.Update()
			ws.Write("testComplete", true)
		case "getStatus":
			// Return current LED status
			status := map[string]interface{}{
				"controllerType": web.arena.EventSettings.LedControllerType,
				"redHealthy":     web.arena.RedHubLeds.IsHealthy(),
				"blueHealthy":    web.arena.BlueHubLeds.IsHealthy(),
				"redColor":       web.arena.RedHubLeds.GetColor(),
				"blueColor":      web.arena.BlueHubLeds.GetColor(),
			}
			ws.Write("status", status)
		default:
			ws.WriteError(fmt.Sprintf("Invalid message type '%s'.", messageType))
			continue
		}
	}
}
