# Govee Device ID Guide

## The Problem

The Govee app shows MAC addresses in the standard 6-byte format:
```
5C:E7:53:80:A7:62
5C:E7:53:56:BE:A8
```

However, the Govee LAN protocol uses an **8-byte device ID format**:
```
31:E1:DB:E6:45:46:08:46
AA:BB:CC:DD:EE:FF:00:00
```

**You cannot use the MAC address from the Govee app directly.** You must discover the devices using Cheesy Arena to get the correct 8-byte device IDs.

---

## How to Find Your Device IDs

### Method 1: Using Cheesy Arena UI (Recommended)

1. **Start Cheesy Arena**:
   ```bash
   cd cheesy-arena
   ./cheesy-arena
   ```

2. **Open the LED Setup Page**:
   - Navigate to `http://localhost:8080`
   - Click **Setup** → **LED Configuration**

3. **Discover Devices**:
   - Select "Govee Smart LEDs (WiFi)" as controller type
   - Click the **"Discover"** button
   - Wait 2-3 seconds for devices to appear

4. **Copy Device IDs**:
   - Discovered devices will appear in a list showing:
     ```
     AA:BB:CC:DD:EE:FF:00:00 (H6159 @ 192.168.1.100)
     ```
   - Click on a device to auto-populate the field
   - Or manually copy the 8-byte device ID

5. **Save Configuration**:
   - Click "Save Configuration"
   - Test the LEDs using the test buttons

---

### Method 2: Using the Logs

If the device discovery isn't working in the UI, you can check the logs:

1. **Start Cheesy Arena** and watch the console output

2. **Look for discovery messages**:
   ```
   [Govee] Discovered device: AA:BB:CC:DD:EE:FF:00:00 (H6159 @ 192.168.1.100)
   ```

3. **If you see "device not found" errors**, the logs will now show all discovered devices:
   ```
   [Govee] Error updating device 5C:E7:53:80:A7:62: device not found: 5C:E7:53:80:A7:62
   [Govee] Discovered devices on network:
   [Govee]   - 31:E1:DB:E6:45:46:08:46 (H6159 @ 192.168.1.100)
   [Govee]   - 5C:E7:53:80:A7:62:00:00 (H6163 @ 192.168.1.101)
   ```

4. **Copy the correct device IDs** from the log output

---

### Method 3: Manual Network Scan (Advanced)

If you want to manually discover devices using command-line tools:

1. **Listen for scan replies**:
   ```bash
   # In one terminal
   nc -u -l 4002
   ```

2. **Send scan request** (in another terminal):
   ```bash
   echo '{"msg":{"cmd":"scan","data":{"account_topic":"reserve"}}}' | nc -u 239.255.255.250 4001
   ```

3. **Look for responses** like:
   ```json
   {
     "msg": {
       "cmd": "scan",
       "data": {
         "device": "31:E1:DB:E6:45:46:08:46",
         "ip": "192.168.1.100",
         "sku": "H6159"
       }
     }
   }
   ```

4. **Use the "device" field** as your device ID

---

## Troubleshooting

### No Devices Discovered

**Possible causes**:
1. Devices not powered on
2. Devices not connected to WiFi
3. Devices on different network/subnet
4. Multicast blocked by router/firewall
5. Firewall blocking UDP ports 4001-4003

**Solutions**:
1. Verify devices are powered and showing solid WiFi indicator
2. Check that devices are on the same network as Cheesy Arena
3. Enable multicast on your router/switch
4. Disable firewall temporarily to test
5. Try connecting Cheesy Arena computer and Govee devices to same WiFi network

### Wrong Device ID Format

**Symptom**: Error message like:
```
[Govee] Error updating device 5C:E7:53:80:A7:62: device not found: 5C:E7:53:80:A7:62
```

**Solution**: This means you're using the 6-byte MAC from the Govee app. Use the discovery feature to get the correct 8-byte device ID.

### Devices Discovered But Not Responding

**Possible causes**:
1. Device went to sleep
2. WiFi connection dropped
3. IP address changed

**Solutions**:
1. Power cycle the device
2. Click "Discover" again to refresh device list
3. Check device WiFi connection in Govee app
4. Ensure devices have static IP or DHCP reservation

---

## Example Configuration

Here's what a correct configuration looks like:

**Incorrect** (6-byte MAC from Govee app):
```
Red Hub Device ID:  5C:E7:53:80:A7:62  ❌
Blue Hub Device ID: 5C:E7:53:56:BE:A8  ❌
```

**Correct** (8-byte device ID from discovery):
```
Red Hub Device ID:  31:E1:DB:E6:45:46:08:46  ✅
Blue Hub Device ID: 5C:E7:53:80:A7:62:00:00  ✅
```

Notice the 8-byte format has **8 hex pairs** separated by colons, not 6.

---

## Quick Reference

| Source | Format | Example | Use in Cheesy Arena? |
|--------|--------|---------|---------------------|
| Govee App | 6-byte MAC | `5C:E7:53:80:A7:62` | ❌ No |
| LAN Discovery | 8-byte Device ID | `31:E1:DB:E6:45:46:08:46` | ✅ Yes |

**Always use the device ID from the discovery feature, not the MAC address from the Govee app.**

---

**Last Updated**: 2026-02-18

