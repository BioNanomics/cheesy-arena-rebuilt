// Copyright 2026 Team 254. All Rights Reserved.
// Author: Integration Team
//
// Tests for Govee client.

package partner

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewGoveeClient(t *testing.T) {
	client := NewGoveeClient()
	assert.NotNil(t, client)
	assert.NotNil(t, client.devices)
	assert.NotNil(t, client.stopChan)
	assert.False(t, client.isRunning)
}

func TestGoveeClient_GetDevice_NotFound(t *testing.T) {
	client := NewGoveeClient()
	device, err := client.GetDevice("nonexistent")
	assert.Nil(t, device)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device not found")
}

func TestGoveeClient_GetDevice_Timeout(t *testing.T) {
	client := NewGoveeClient()
	
	// Add a device with old timestamp
	client.devices["test-device"] = &GoveeDevice{
		DeviceId: "test-device",
		IP:       "192.168.1.100",
		SKU:      "H6159",
		LastSeen: time.Now().Add(-2 * time.Minute), // 2 minutes ago
	}

	device, err := client.GetDevice("test-device")
	assert.Nil(t, device)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "device timeout")
}

func TestGoveeClient_GetDevice_Success(t *testing.T) {
	client := NewGoveeClient()
	
	// Add a fresh device
	client.devices["test-device"] = &GoveeDevice{
		DeviceId: "test-device",
		IP:       "192.168.1.100",
		SKU:      "H6159",
		LastSeen: time.Now(),
	}

	device, err := client.GetDevice("test-device")
	assert.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "test-device", device.DeviceId)
	assert.Equal(t, "192.168.1.100", device.IP)
	assert.Equal(t, "H6159", device.SKU)
}

func TestGoveeClient_GetAllDevices(t *testing.T) {
	client := NewGoveeClient()
	
	// Add multiple devices
	client.devices["device1"] = &GoveeDevice{
		DeviceId: "device1",
		IP:       "192.168.1.100",
		SKU:      "H6159",
		LastSeen: time.Now(),
	}
	client.devices["device2"] = &GoveeDevice{
		DeviceId: "device2",
		IP:       "192.168.1.101",
		SKU:      "H6163",
		LastSeen: time.Now(),
	}
	// Add a stale device
	client.devices["device3"] = &GoveeDevice{
		DeviceId: "device3",
		IP:       "192.168.1.102",
		SKU:      "H6182",
		LastSeen: time.Now().Add(-2 * time.Minute),
	}

	devices := client.GetAllDevices()
	assert.Len(t, devices, 2) // Stale device should be pruned
}

func TestGoveeClient_PruneStaleDevices(t *testing.T) {
	client := NewGoveeClient()
	
	// Add fresh device
	client.devices["fresh"] = &GoveeDevice{
		DeviceId: "fresh",
		IP:       "192.168.1.100",
		LastSeen: time.Now(),
	}
	
	// Add stale device
	client.devices["stale"] = &GoveeDevice{
		DeviceId: "stale",
		IP:       "192.168.1.101",
		LastSeen: time.Now().Add(-2 * time.Minute),
	}

	client.pruneStaleDevices()

	assert.Len(t, client.devices, 1)
	assert.Contains(t, client.devices, "fresh")
	assert.NotContains(t, client.devices, "stale")
}

func TestGoveeMessage_Marshal(t *testing.T) {
	msg := goveeMessage{
		Msg: goveeMessageContent{
			Cmd: "turn",
			Data: map[string]interface{}{
				"value": 1,
			},
		},
	}

	jsonData, err := json.Marshal(msg)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), "\"cmd\":\"turn\"")
	assert.Contains(t, string(jsonData), "\"value\":1")
}

func TestGoveeMessage_Unmarshal(t *testing.T) {
	jsonStr := `{"msg":{"cmd":"scan","data":{"device":"AA:BB:CC:DD:EE:FF","ip":"192.168.1.100","sku":"H6159"}}}`
	
	var msg goveeMessage
	err := json.Unmarshal([]byte(jsonStr), &msg)
	assert.NoError(t, err)
	assert.Equal(t, "scan", msg.Msg.Cmd)
	assert.Equal(t, "AA:BB:CC:DD:EE:FF", msg.Msg.Data["device"])
	assert.Equal(t, "192.168.1.100", msg.Msg.Data["ip"])
	assert.Equal(t, "H6159", msg.Msg.Data["sku"])
}

func TestGoveeClient_IsEnabled(t *testing.T) {
	client := NewGoveeClient()
	assert.False(t, client.IsEnabled())

	client.isRunning = true
	assert.True(t, client.IsEnabled())
}

