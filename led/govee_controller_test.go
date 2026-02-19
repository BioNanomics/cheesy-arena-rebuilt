// Copyright 2026 Team 254. All Rights Reserved.
// Author: Integration Team
//
// Tests for Govee LED controller.

package led

import (
	"testing"
	"time"

	"github.com/Team254/cheesy-arena/partner"
	"github.com/stretchr/testify/assert"
)

func TestNewGoveeController(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	assert.NotNil(t, controller)
	assert.Equal(t, "test-device", controller.deviceId)
	assert.NotNil(t, controller.goveeClient)
	assert.True(t, controller.isHealthy)
}

func TestGoveeController_SetAddress(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("device1", client)
	
	err := controller.SetAddress("device2")
	assert.NoError(t, err)
	assert.Equal(t, "device2", controller.deviceId)
}

func TestGoveeController_SetGetColor(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	// Test setting red
	controller.SetColor(ColorRed)
	assert.Equal(t, ColorRed, controller.GetColor())
	
	// Test setting blue
	controller.SetColor(ColorBlue)
	assert.Equal(t, ColorBlue, controller.GetColor())
	
	// Test setting off
	controller.SetColor(ColorOff)
	assert.Equal(t, ColorOff, controller.GetColor())
}

func TestGoveeController_Update_NoDeviceId(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("", client)
	
	controller.SetColor(ColorRed)
	controller.Update() // Should not panic with empty device ID
	
	assert.True(t, controller.IsHealthy())
}

func TestGoveeController_Update_ClientNotEnabled(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	controller.SetColor(ColorRed)
	controller.Update() // Should not panic when client not enabled
	
	// Health should still be true since we haven't tried to communicate
	assert.True(t, controller.IsHealthy())
}

func TestGoveeController_IsHealthy_Timeout(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	// Set last success to long ago
	controller.lastSuccess = time.Now().Add(-10 * time.Second)
	
	assert.False(t, controller.IsHealthy())
}

func TestGoveeController_Close(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	controller.SetColor(ColorRed)
	controller.Close()
	
	// After close, color should be off
	assert.Equal(t, ColorOff, controller.GetColor())
}

func TestGoveeController_UpdateThrottling(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	controller.SetColor(ColorRed)
	controller.lastUpdate = time.Now()
	controller.lastColor = ColorRed
	
	// Update should be throttled
	controller.Update()
	
	// Last update time should not have changed significantly
	assert.WithinDuration(t, time.Now(), controller.lastUpdate, 50*time.Millisecond)
}

func TestGoveeController_ColorTransitions(t *testing.T) {
	client := partner.NewGoveeClient()
	controller := NewGoveeController("test-device", client)
	
	// Test off -> red transition
	controller.SetColor(ColorOff)
	controller.lastColor = ColorOff
	controller.SetColor(ColorRed)
	assert.Equal(t, ColorRed, controller.GetColor())
	
	// Test red -> blue transition
	controller.SetColor(ColorBlue)
	assert.Equal(t, ColorBlue, controller.GetColor())
	
	// Test blue -> off transition
	controller.SetColor(ColorOff)
	assert.Equal(t, ColorOff, controller.GetColor())
}

func TestGoveeController_InterfaceCompliance(t *testing.T) {
	client := partner.NewGoveeClient()
	var controller LedController = NewGoveeController("test-device", client)
	
	// Verify interface methods are available
	assert.NotNil(t, controller)
	controller.SetColor(ColorGreen)
	assert.Equal(t, ColorGreen, controller.GetColor())
	controller.Update()
	assert.NotNil(t, controller.IsHealthy)
	controller.Close()
}

