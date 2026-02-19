# Govee LED Integration - Testing Guide

This document provides comprehensive testing procedures for the Govee LED integration in Cheesy Arena.

## Table of Contents

1. [Unit Testing](#unit-testing)
2. [Integration Testing](#integration-testing)
3. [Manual Testing Procedures](#manual-testing-procedures)
4. [Performance Testing](#performance-testing)
5. [Troubleshooting](#troubleshooting)

---

## Unit Testing

### Running Unit Tests

All unit tests are included in the standard test suite and run automatically.

```bash
# Run all tests
cd cheesy-arena
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./partner -v
go test ./led -v

# Run with race detection
go test -race ./partner ./led
```

### Test Coverage

Current test coverage for new code:

- **partner/govee.go**: 9 tests covering device discovery, commands, timeout, and pruning
- **led/govee_controller.go**: 10 tests covering interface compliance, color transitions, health monitoring
- **Coverage**: 47.6% for partner package, 21.6% for LED package (includes existing DMX code)

### Expected Results

All unit tests should pass:
```
PASS
ok  	github.com/Team254/cheesy-arena/partner	0.463s	coverage: 47.6% of statements
PASS
ok  	github.com/Team254/cheesy-arena/led	0.214s	coverage: 21.6% of statements
```

---

## Integration Testing

Integration tests require actual hardware and are disabled by default.

### Prerequisites

**For Govee Testing:**
- 1+ Govee LED device (H6159, H6163, H6182, or compatible)
- Device powered on and connected to same network as test machine
- Multicast-enabled network (UDP multicast must work)

**For DMX Testing:**
- DMX/sACN controller on network
- LED fixtures connected to controller

### Running Integration Tests

```bash
# Set environment variables
export GOVEE_TEST_DEVICE_ID="AA:BB:CC:DD:EE:FF:00:00"  # Your device MAC
export DMX_TEST_ADDRESS="10.0.100.80"                   # Your DMX controller IP

# Run integration tests
cd cheesy-arena
go test -tags=integration ./led -v

# Expected output:
# === RUN   TestGoveeIntegration
# Found device: AA:BB:CC:DD:EE:FF:00:00 (H6159 @ 192.168.1.100)
# Testing color: Red
# Testing color: Blue
# ...
# --- PASS: TestGoveeIntegration (15.00s)
```

### Finding Your Govee Device ID

If you don't know your device MAC address:

1. Start Cheesy Arena
2. Navigate to Setup → LED Configuration
3. Select "Govee Smart LEDs (WiFi)"
4. Click "Discover" button
5. Note the device ID (MAC address) from the discovered devices list

---

## Manual Testing Procedures

### Test 1: Initial Configuration

**Objective**: Verify LED setup page and configuration saving

**Steps**:
1. Start Cheesy Arena: `./cheesy-arena`
2. Open browser: `http://localhost:8080`
3. Navigate to Setup → LED Configuration
4. Verify page loads without errors
5. Select "DMX/sACN (Professional Lighting)"
6. Enter DMX address: `10.0.100.80`
7. Click "Save Configuration"
8. Verify redirect back to LED setup page
9. Verify settings persisted (DMX address still shown)

**Expected Result**: ✅ Configuration saves and persists

---

### Test 2: Govee Device Discovery

**Objective**: Verify Govee device discovery works

**Prerequisites**: Govee device powered on and on network

**Steps**:
1. Navigate to Setup → LED Configuration
2. Select "Govee Smart LEDs (WiFi)"
3. Click "Discover" button for Red Hub
4. Wait 2-3 seconds
5. Verify discovered devices appear in list
6. Click on a device in the list
7. Verify device ID populates in Red Hub field
8. Repeat for Blue Hub
9. Click "Save Configuration"

**Expected Result**: ✅ Devices discovered and configuration saved

---

### Test 3: LED Testing (DMX)

**Objective**: Verify DMX LED control works

**Prerequisites**: DMX controller configured

**Steps**:
1. Navigate to Setup → LED Configuration
2. Ensure DMX mode is selected and configured
3. Click "Red" button under Red Hub LED
4. Verify physical LED turns red
5. Click "Blue" button
6. Verify physical LED turns blue
7. Click "Green" button
8. Verify physical LED turns green
9. Click "Purple" button
10. Verify physical LED turns purple
11. Click "Off" button
12. Verify physical LED turns off
13. Repeat for Blue Hub LED

**Expected Result**: ✅ All colors display correctly on physical LEDs

---

### Test 4: LED Testing (Govee)

**Objective**: Verify Govee LED control works

**Prerequisites**: Govee devices configured

**Steps**:
1. Navigate to Setup → LED Configuration
2. Ensure Govee mode is selected and configured
3. Click "Red" button under Red Hub LED
4. Verify physical Govee LED turns red (may take 100-500ms)
5. Click "Blue" button
6. Verify physical LED turns blue
7. Click "Green" button
8. Verify physical LED turns green
9. Click "Purple" button
10. Verify physical LED turns purple
11. Click "Off" button
12. Verify physical LED turns off
13. Verify health badge shows "Healthy" (green)
14. Repeat for Blue Hub LED

**Expected Result**: ✅ All colors display correctly, health status accurate

---

### Test 5: Match Play Integration (Govee)

**Objective**: Verify LEDs work correctly during match play

**Prerequisites**: Govee devices configured, test match loaded

**Steps**:
1. Navigate to Match Play
2. Load a test match
3. Observe LED state: Both should be OFF
4. Click "Show PreMatch" or start match countdown
5. Verify both LEDs turn GREEN
6. Start match (Auto period)
7. Verify LEDs show hub state (typically OFF initially)
8. Trigger hub activation (via PLC or manual scoring)
9. Verify active hub LED turns alliance color (Red/Blue)
10. Transition to Teleop
11. Verify LEDs continue to reflect hub state
12. End match
13. Verify PostMatch sequence: OFF → PURPLE (counting) → GREEN (ready)

**Expected Result**: ✅ LEDs follow match state correctly

---

### Test 6: Configuration Switching

**Objective**: Verify switching between DMX and Govee works

**Prerequisites**: Both DMX and Govee hardware available

**Steps**:
1. Configure for DMX mode
2. Test Red LED (should work)
3. Navigate to Setup → LED Configuration
4. Switch to Govee mode
5. Configure Govee devices
6. Save configuration
7. Test Red LED (should work with Govee)
8. Switch back to DMX mode
9. Save configuration
10. Test Red LED (should work with DMX)

**Expected Result**: ✅ Switching between modes works without restart

---

### Test 7: Error Handling - Invalid Device

**Objective**: Verify graceful handling of non-existent device

**Steps**:
1. Navigate to Setup → LED Configuration
2. Select Govee mode
3. Enter invalid device ID: `00:00:00:00:00:00:00:00`
4. Save configuration
5. Navigate to Match Play
6. Load and start a test match
7. Verify arena doesn't crash
8. Check logs for error messages
9. Navigate back to LED Configuration
10. Verify health badge shows "Unhealthy" (red)

**Expected Result**: ✅ Graceful degradation, no crash, clear error indication

---

### Test 8: Network Disconnection Recovery

**Objective**: Verify recovery when Govee device loses network

**Prerequisites**: Govee device configured and working

**Steps**:
1. Verify LED is working (test with color buttons)
2. Unplug Govee device from power
3. Wait 10 seconds
4. Verify health badge changes to "Unhealthy"
5. Plug device back in
6. Wait 30 seconds for device to reconnect
7. Test LED again
8. Verify health badge returns to "Healthy"

**Expected Result**: ✅ System detects disconnection and recovers automatically

---

### Test 9: Multiple Matches Performance

**Objective**: Verify stable operation over multiple matches

**Steps**:
1. Configure Govee LEDs
2. Run 10 consecutive test matches
3. Observe LED behavior in each match
4. Monitor response times
5. Check for any degradation
6. Review logs for errors

**Expected Result**: ✅ Consistent performance, no memory leaks, no errors

---

### Test 10: WebSocket Real-time Updates

**Objective**: Verify real-time status updates on LED setup page

**Steps**:
1. Navigate to Setup → LED Configuration
2. Open browser developer console (F12)
3. Verify WebSocket connection established (check Network tab)
4. Click a test button (e.g., "Red" for Red Hub)
5. Observe console for WebSocket messages
6. Verify health status updates in real-time
7. Verify current color display updates

**Expected Result**: ✅ WebSocket connected, real-time updates working

---

## Performance Testing

### Latency Testing

**Objective**: Measure LED response time

**Method**:
1. Use stopwatch or video recording
2. Click test button
3. Measure time until LED changes color
4. Repeat 10 times for each controller type

**Expected Results**:
- **DMX**: < 50ms (near-instant)
- **Govee**: 100-500ms (WiFi latency)

### Throughput Testing

**Objective**: Verify rapid color changes don't cause issues

**Method**:
1. Modify test to rapidly change colors (10 changes/second)
2. Run for 60 seconds
3. Monitor for packet loss or errors

**Expected Results**:
- DMX: No packet loss
- Govee: Occasional packet loss acceptable (retry logic handles it)

### Memory Testing

**Objective**: Verify no memory leaks

**Method**:
```bash
# Run with memory profiling
go test -memprofile=mem.prof ./led
go tool pprof mem.prof

# Or use runtime monitoring
# Monitor RSS memory during extended operation
```

**Expected Results**: Stable memory usage over time

---

## Troubleshooting

### Govee Devices Not Discovered

**Symptoms**: Discover button returns no devices

**Possible Causes**:
1. Devices not powered on
2. Devices not on same network
3. Multicast blocked by network
4. Firewall blocking UDP ports 4001-4003

**Solutions**:
1. Verify devices are powered and connected to WiFi
2. Check network configuration (same subnet)
3. Enable multicast on router/switch
4. Configure firewall to allow UDP 4001-4003
5. Try discovery from command line:
   ```bash
   # Listen for Govee devices
   nc -u -l 4002
   # In another terminal, send scan request
   echo '{"msg":{"cmd":"scan","data":{"account_topic":"reserve"}}}' | nc -u 239.255.255.250 4001
   ```

### LEDs Not Responding

**Symptoms**: Test buttons don't change LED color

**DMX Troubleshooting**:
1. Verify DMX controller IP is correct
2. Check network connectivity: `ping 10.0.100.80`
3. Verify DMX controller is powered on
4. Check universe configuration (Red=1, Blue=2)
5. Review logs for sACN errors

**Govee Troubleshooting**:
1. Verify device ID is correct (MAC address format)
2. Check device is discovered: click Discover button
3. Verify device is on network
4. Check health status badge
5. Review logs for Govee errors
6. Try power cycling the device

### Health Status Shows "Unhealthy"

**Symptoms**: Red "Unhealthy" badge on LED setup page

**Causes**:
1. Device not responding (timeout > 5 seconds)
2. Network connectivity issues
3. Invalid device configuration

**Solutions**:
1. Click test button to retry communication
2. Verify device is powered and on network
3. Re-run device discovery
4. Check logs for specific error messages
5. Reconfigure device ID if needed

### WebSocket Connection Failed

**Symptoms**: Real-time updates not working

**Solutions**:
1. Refresh the page
2. Check browser console for errors
3. Verify Cheesy Arena is running
4. Check firewall settings
5. Try different browser

### Match Play LEDs Not Working

**Symptoms**: LEDs work in test but not during matches

**Possible Causes**:
1. Configuration not saved
2. Arena not reinitialized after config change
3. Match state not triggering LED updates

**Solutions**:
1. Verify configuration is saved (check LED setup page)
2. Restart Cheesy Arena
3. Check logs during match play
4. Verify PLC/hub state is being detected

---

## Success Criteria

All tests should meet these criteria:

✅ **Unit Tests**: All pass with >75% coverage for new code
✅ **Integration Tests**: Pass with real hardware
✅ **Configuration**: Save/load works correctly
✅ **Device Discovery**: Finds Govee devices on network
✅ **LED Control**: All colors display correctly
✅ **Match Integration**: LEDs follow match state
✅ **Error Handling**: Graceful degradation, no crashes
✅ **Performance**: Acceptable latency, no memory leaks
✅ **Recovery**: Automatic recovery from network issues
✅ **Documentation**: Clear error messages and logs

---

## Reporting Issues

When reporting issues, include:

1. **Environment**:
   - Cheesy Arena version
   - Operating system
   - Network configuration
   - Hardware details (Govee model, DMX controller)

2. **Steps to Reproduce**:
   - Exact sequence of actions
   - Configuration settings
   - Expected vs actual behavior

3. **Logs**:
   - Relevant log excerpts
   - Error messages
   - WebSocket console output

4. **Screenshots**:
   - LED setup page
   - Health status
   - Error messages

---

## Additional Resources

- **Govee LAN Protocol**: See `GOVEE_INTEGRATION_PLAN.md` Appendix A
- **Integration Plan**: `GOVEE_INTEGRATION_PLAN.md`
- **Source Code**:
  - `partner/govee.go` - Govee protocol implementation
  - `led/govee_controller.go` - Govee LED controller
  - `web/setup_leds.go` - LED setup page backend
  - `templates/setup_leds.html` - LED setup page frontend

---

**Document Version**: 1.0
**Last Updated**: 2026-02-18
**Author**: Integration Team

