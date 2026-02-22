// Copyright 2026 Team 254. All Rights Reserved.
// Author: Integration Team
//
// Integration tests for LED controllers.
// These tests require actual hardware and are skipped by default.
// Run with: go test -tags=integration ./led

//go:build integration
// +build integration

package led

import (
	"os"
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/partner"
	"github.com/stretchr/testify/assert"
)

// TestGoveeIntegration tests the full Govee LED workflow with real hardware.
// Set environment variables:
//   GOVEE_TEST_DEVICE_ID - MAC address of test Govee device
func TestGoveeIntegration(t *testing.T) {
	deviceId := os.Getenv("GOVEE_TEST_DEVICE_ID")
	if deviceId == "" {
		t.Skip("Skipping integration test: GOVEE_TEST_DEVICE_ID not set")
	}

	// Create Govee client
	client := partner.NewGoveeClient()
	assert.NotNil(t, client)

	// Start discovery
	err := client.StartDiscovery()
	assert.NoError(t, err)
	defer client.StopDiscovery()

	// Wait for device discovery
	t.Log("Waiting for device discovery...")
	time.Sleep(3 * time.Second)

	// Check if device was discovered
	device, err := client.GetDevice(deviceId)
	if err != nil {
		t.Logf("Device not found. Discovered devices:")
		for _, d := range client.GetAllDevices() {
			t.Logf("  - %s (%s @ %s)", d.DeviceId, d.SKU, d.IP)
		}
		t.Fatalf("Test device %s not found: %v", deviceId, err)
	}
	assert.NotNil(t, device)
	t.Logf("Found device: %s (%s @ %s)", device.DeviceId, device.SKU, device.IP)

	// Create controller
	controller := NewGoveeController(deviceId, client)
	assert.NotNil(t, controller)

	// Test color sequence
	colors := []struct {
		name  string
		color Color
	}{
		{"Red", ColorRed},
		{"Blue", ColorBlue},
		{"Green", ColorGreen},
		{"Purple", ColorPurple},
		{"Off", ColorOff},
	}

	for _, tc := range colors {
		t.Logf("Testing color: %s", tc.name)
		controller.SetColor(tc.color)
		controller.Update()
		time.Sleep(1 * time.Second)

		// Verify color was set
		assert.Equal(t, tc.color, controller.GetColor())
		assert.True(t, controller.IsHealthy(), "Controller should be healthy after %s", tc.name)
	}

	// Test rapid color changes
	t.Log("Testing rapid color changes...")
	for i := 0; i < 10; i++ {
		controller.SetColor(ColorRed)
		controller.Update()
		time.Sleep(50 * time.Millisecond)
		controller.SetColor(ColorBlue)
		controller.Update()
		time.Sleep(50 * time.Millisecond)
	}
	assert.True(t, controller.IsHealthy(), "Controller should remain healthy after rapid changes")

	// Clean up
	controller.Close()
	t.Log("Integration test complete")
}

// TestDmxIntegration tests the DMX LED workflow with real hardware.
// Set environment variables:
//   DMX_TEST_ADDRESS - IP address of DMX controller
func TestDmxIntegration(t *testing.T) {
	address := os.Getenv("DMX_TEST_ADDRESS")
	if address == "" {
		t.Skip("Skipping integration test: DMX_TEST_ADDRESS not set")
	}

	// Create DMX controller
	controller := &DmxController{Universe: 1, StartChannel: 1}
	err := controller.SetAddress(address)
	assert.NoError(t, err)
	defer controller.Close()

	// Test color sequence
	colors := []struct {
		name  string
		color Color
	}{
		{"Red", ColorRed},
		{"Blue", ColorBlue},
		{"Green", ColorGreen},
		{"Purple", ColorPurple},
		{"Off", ColorOff},
	}

	for _, tc := range colors {
		t.Logf("Testing color: %s", tc.name)
		controller.SetColor(tc.color)
		controller.Update()
		time.Sleep(500 * time.Millisecond)

		// Verify color was set
		assert.Equal(t, tc.color, controller.GetColor())
		assert.True(t, controller.IsHealthy(), "Controller should be healthy")
	}

	// Test rapid updates
	t.Log("Testing rapid updates...")
	for i := 0; i < 20; i++ {
		controller.SetColor(ColorRed)
		controller.Update()
		time.Sleep(50 * time.Millisecond)
	}
	assert.True(t, controller.IsHealthy(), "Controller should remain healthy after rapid updates")

	t.Log("Integration test complete")
}

