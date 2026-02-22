# Govee LED Control Integration Plan

## Executive Summary

### Overview
This document outlines the integration of Govee smart LED control capabilities from the AM Showcase Stream Overlay project into Cheesy Arena. The integration will provide a cost-effective alternative to professional DMX lighting systems while maintaining compatibility with existing DMX infrastructure.

### Goals
- Enable Cheesy Arena to control Govee smart LEDs via the Govee LAN protocol
- Maintain backward compatibility with existing DMX/sACN LED controllers
- Provide a unified interface for LED control regardless of hardware type
- Implement robust error handling and retry logic for consumer-grade WiFi devices
- Create user-friendly configuration UI for device discovery and setup

### Approach
Implement the Govee LAN protocol natively in Go as a new partner integration, refactor the LED controller to use an interface-based design pattern, and provide configuration options to select between DMX and Govee hardware.

### Key Benefits
- **Cost Reduction**: Govee LEDs cost significantly less than professional DMX fixtures
- **Flexibility**: Easier deployment for practice sessions and smaller events
- **No External Dependencies**: Pure Go implementation, no Node.js required
- **Dual Hardware Support**: Can run DMX and Govee simultaneously or independently

---

## Prerequisites

### Development Environment
- **Go Version**: 1.23.0 or newer (as specified in go.mod)
- **Operating System**: macOS, Linux, or Windows
- **Network Access**: Local network with multicast support for Govee device discovery
- **Git**: For version control

### Hardware Requirements (for testing)
- At least 2 Govee smart LED devices (WiFi-enabled models with LAN API support)
- Network router/switch supporting UDP multicast (239.255.255.250)
- Devices must be on same network as Cheesy Arena server

### Software Dependencies
- No new external Go dependencies required (using standard library `net` package)
- Existing Cheesy Arena dependencies remain unchanged

### Knowledge Requirements
- Go programming language
- UDP networking and multicast
- Cheesy Arena architecture and codebase
- Basic understanding of LED control systems

---

## Architecture Overview

### Current State
```
Arena
  └── RedHubLeds (led.Controller - DMX/sACN)
  └── BlueHubLeds (led.Controller - DMX/sACN)
```

### Target State
```
Arena
  └── RedHubLeds (led.LedController interface)
  │     ├── DmxController (sACN implementation)
  │     └── GoveeController (Govee LAN implementation)
  └── BlueHubLeds (led.LedController interface)
        ├── DmxController (sACN implementation)
        └── GoveeController (Govee LAN implementation)

Partner Integrations
  └── GoveeClient (device discovery and management)
```

### Component Responsibilities

**led.LedController Interface**
- Define common LED control operations (SetColor, Update, Close)
- Abstract hardware-specific implementation details

**led.DmxController**
- Existing sACN/E1.31 implementation
- Refactored from current led.Controller

**led.GoveeController**
- New Govee LAN protocol implementation
- Manages individual Govee device communication
- Implements retry logic and health monitoring

**partner.GoveeClient**
- Device discovery via UDP multicast
- Device registry and lifecycle management
- Provides device lookup for controllers

---

## Detailed Implementation Steps

### Phase 1: Core Infrastructure (Day 1)

#### Step 1.1: Create Govee Protocol Package
**File**: `cheesy-arena/partner/govee.go`

**Tasks**:
1. Create GoveeClient struct with device registry
2. Implement UDP multicast device discovery
3. Implement device control commands (turn on/off, set color)
4. Add retry logic for unreliable WiFi connections
5. Implement health monitoring and device timeout detection

**Key Structures**:
```go
type GoveeDevice struct {
    DeviceId  string    // MAC address format
    IP        string    // Current IP address
    SKU       string    // Device model
    LastSeen  time.Time // For health monitoring
}

type GoveeClient struct {
    devices      map[string]*GoveeDevice
    scanSocket   *net.UDPConn
    mutex        sync.RWMutex
}
```

**Methods to Implement**:
- `NewGoveeClient() *GoveeClient`
- `StartDiscovery() error`
- `StopDiscovery()`
- `GetDevice(deviceId string) (*GoveeDevice, error)`
- `SendCommand(deviceId, command string, data map[string]interface{}) error`
- `TurnOn(deviceId string) error`
- `TurnOff(deviceId string) error`
- `SetColor(deviceId string, r, g, b uint8) error`

**Protocol Details**:
- Multicast Address: `239.255.255.250`
- Scan Port: `4001` (outbound)
- Reply Port: `4002` (inbound)
- Control Port: `4003` (to device IP)
- Message Format: JSON with `{"msg": {"cmd": "...", "data": {...}}}`

#### Step 1.2: Create Govee Protocol Tests
**File**: `cheesy-arena/partner/govee_test.go`

**Test Cases**:
1. `TestGoveeClient_Discovery` - Mock UDP multicast discovery
2. `TestGoveeClient_SendCommand` - Verify command formatting
3. `TestGoveeClient_TurnOnOff` - Test on/off commands
4. `TestGoveeClient_SetColor` - Test color commands
5. `TestGoveeClient_RetryLogic` - Verify retry on failure
6. `TestGoveeClient_DeviceTimeout` - Test stale device removal

#### Step 1.3: Create LED Controller Interface
**File**: `cheesy-arena/led/controller.go` (modify existing)

**Tasks**:
1. Extract interface from existing Controller struct
2. Define common LED operations
3. Add documentation for interface contract

**Interface Definition**:
```go
// LedController defines the interface for controlling LED lighting systems
type LedController interface {
    SetAddress(address string) error
    SetColor(color Color)
    GetColor() Color
    Update()
    Close()
    IsHealthy() bool
}
```

**Changes Required**:
- Rename existing `Controller` to `DmxController`
- Ensure `DmxController` implements `LedController` interface
- Update all method signatures to match interface

### Phase 2: Govee LED Controller Implementation (Day 2)

#### Step 2.1: Create Govee LED Controller
**File**: `cheesy-arena/led/govee_controller.go`

**Tasks**:
1. Create GoveeController struct implementing LedController interface
2. Integrate with partner.GoveeClient for device communication
3. Implement color state management
4. Add retry logic for unreliable connections
5. Implement health monitoring

**Key Structure**:
```go
type GoveeController struct {
    deviceId     string
    goveeClient  *partner.GoveeClient
    color        Color
    lastColor    Color
    lastUpdate   time.Time
    isHealthy    bool
    retryCount   int
    mutex        sync.Mutex
}
```

**Methods to Implement**:
- `NewGoveeController(deviceId string, client *partner.GoveeClient) *GoveeController`
- `SetAddress(address string) error` - Store device ID
- `SetColor(color Color)` - Update desired color
- `GetColor() Color` - Return current color
- `Update()` - Send color to device with retry logic
- `Close()` - Cleanup resources
- `IsHealthy() bool` - Check device connectivity

**Retry Strategy**:
- OFF commands: Retry 3 times with 120ms delay (from AM Showcase pattern)
- ON commands: Single attempt
- Color commands: Single attempt with verification

#### Step 2.2: Create Govee Controller Tests
**File**: `cheesy-arena/led/govee_controller_test.go`

**Test Cases**:
1. `TestGoveeController_SetColor` - Verify color changes
2. `TestGoveeController_Update` - Test update logic
3. `TestGoveeController_RetryLogic` - Verify retry behavior
4. `TestGoveeController_HealthMonitoring` - Test health checks
5. `TestGoveeController_StateManagement` - Verify state tracking

#### Step 2.3: Refactor Existing DMX Controller
**File**: `cheesy-arena/led/dmx_controller.go` (rename from controller.go)

**Tasks**:
1. Rename `Controller` struct to `DmxController`
2. Ensure all methods match `LedController` interface
3. Add `IsHealthy()` method if not present
4. Update package documentation

**Note**: Keep all existing DMX/sACN logic intact, only rename and add interface compliance.

### Phase 3: Arena Integration (Day 2-3)

#### Step 3.1: Update Arena Structure
**File**: `cheesy-arena/field/arena.go`

**Changes Required**:

1. **Import Updates** (line ~19):
```go
import (
    // ... existing imports
    "github.com/Team254/cheesy-arena/partner"
)
```

2. **Arena Struct Updates** (line ~56):
```go
type Arena struct {
    // ... existing fields
    RedHubLeds       led.LedController  // Changed from led.Controller
    BlueHubLeds      led.LedController  // Changed from led.Controller
    GoveeClient      *partner.GoveeClient  // New field
    // ... rest of fields
}
```

3. **NewArena Function Updates** (line ~120):
```go
func NewArena(dbPath string) (*Arena, error) {
    arena := new(Arena)
    arena.configureNotifiers()
    arena.Plc = new(plc.ModbusPlc)

    // Initialize Govee client
    arena.GoveeClient = partner.NewGoveeClient()

    // LED controllers will be initialized based on settings
    // Default to DMX for backward compatibility
    arena.RedHubLeds = &led.DmxController{Universe: 1, StartChannel: 1}
    arena.BlueHubLeds = &led.DmxController{Universe: 2, StartChannel: 1}

    // ... rest of initialization
}
```

4. **Add LED Initialization Method**:
```go
// InitializeLedControllers sets up LED controllers based on event settings
func (arena *Arena) InitializeLedControllers() error {
    if arena.EventSettings.LedControllerType == "govee" {
        if arena.EventSettings.RedLedDeviceId != "" {
            arena.RedHubLeds = led.NewGoveeController(
                arena.EventSettings.RedLedDeviceId,
                arena.GoveeClient,
            )
        }
        if arena.EventSettings.BlueLedDeviceId != "" {
            arena.BlueHubLeds = led.NewGoveeController(
                arena.EventSettings.BlueLedDeviceId,
                arena.GoveeClient,
            )
        }
    } else {
        // DMX controllers (existing behavior)
        redDmx := &led.DmxController{Universe: 1, StartChannel: 1}
        blueDmx := &led.DmxController{Universe: 2, StartChannel: 1}

        if arena.EventSettings.RedLedAddress != "" {
            redDmx.SetAddress(arena.EventSettings.RedLedAddress)
        }
        if arena.EventSettings.BlueLedAddress != "" {
            blueDmx.SetAddress(arena.EventSettings.BlueLedAddress)
        }

        arena.RedHubLeds = redDmx
        arena.BlueHubLeds = blueDmx
    }
    return nil
}
```

#### Step 3.2: Update Event Settings Model
**File**: `cheesy-arena/model/event_settings.go`

**Tasks**:
1. Add new fields for LED controller configuration
2. Update database schema version if needed
3. Add migration logic for existing databases

**New Fields to Add**:
```go
type EventSettings struct {
    // ... existing fields

    // LED Controller Configuration
    LedControllerType  string `json:"ledControllerType"`  // "dmx" or "govee"
    RedLedAddress      string `json:"redLedAddress"`      // DMX: IP address
    BlueLedAddress     string `json:"blueLedAddress"`     // DMX: IP address
    RedLedDeviceId     string `json:"redLedDeviceId"`     // Govee: Device MAC
    BlueLedDeviceId    string `json:"blueLedDeviceId"`    // Govee: Device MAC

    // ... rest of fields
}
```

**Default Values**:
- `LedControllerType`: `"dmx"` (backward compatibility)
- All device IDs/addresses: `""` (empty, not configured)

#### Step 3.3: Update Arena Run Loop
**File**: `cheesy-arena/field/arena.go`

**Tasks**:
1. Start Govee discovery when arena starts
2. Stop Govee discovery when arena stops
3. Ensure LED controllers are properly initialized

**Changes in `Run()` method** (around line 200):
```go
func (arena *Arena) Run() {
    // Start Govee device discovery
    if arena.GoveeClient != nil {
        if err := arena.GoveeClient.StartDiscovery(); err != nil {
            log.Printf("Warning: Failed to start Govee discovery: %v", err)
        }
        defer arena.GoveeClient.StopDiscovery()
    }

    // Initialize LED controllers based on settings
    if err := arena.InitializeLedControllers(); err != nil {
        log.Printf("Warning: Failed to initialize LED controllers: %v", err)
    }

    // ... existing run loop code
}
```

### Phase 4: Configuration UI (Day 3-4)

#### Step 4.1: Create LED Setup Page Backend
**File**: `cheesy-arena/web/setup_leds.go`

**Tasks**:
1. Create GET handler for LED setup page
2. Create POST handler for saving LED configuration
3. Create API endpoint for device discovery
4. Create API endpoint for testing LED commands
5. Create WebSocket handler for real-time device status

**Handlers to Implement**:
```go
// GET /setup/leds - Display LED configuration page
func (web *Web) ledsGetHandler(w http.ResponseWriter, r *http.Request)

// POST /setup/leds - Save LED configuration
func (web *Web) ledsPostHandler(w http.ResponseWriter, r *http.Request)

// GET /setup/leds/discover - Get list of discovered Govee devices
func (web *Web) ledsDiscoverHandler(w http.ResponseWriter, r *http.Request)

// POST /setup/leds/test - Test LED command (turn on/off, color)
func (web *Web) ledsTestHandler(w http.ResponseWriter, r *http.Request)

// GET /setup/leds/websocket - WebSocket for real-time updates
func (web *Web) ledsWebsocketHandler(w http.ResponseWriter, r *http.Request)
```

**API Response Structures**:
```go
type LedSetupData struct {
    ControllerType    string         `json:"controllerType"`
    DmxConfig         DmxConfig      `json:"dmxConfig"`
    GoveeConfig       GoveeConfig    `json:"goveeConfig"`
    DiscoveredDevices []GoveeDevice  `json:"discoveredDevices"`
}

type DmxConfig struct {
    RedAddress  string `json:"redAddress"`
    BlueAddress string `json:"blueAddress"`
}

type GoveeConfig struct {
    RedDeviceId  string `json:"redDeviceId"`
    BlueDeviceId string `json:"blueDeviceId"`
}
```

#### Step 4.2: Create LED Setup Page Frontend
**File**: `cheesy-arena/templates/setup_leds.html`

**UI Components**:
1. **Controller Type Selection**
   - Radio buttons: DMX / Govee
   - Show/hide relevant configuration sections

2. **DMX Configuration Section**
   - Text inputs for Red/Blue LED IP addresses
   - Test buttons for each controller

3. **Govee Configuration Section**
   - Device discovery button
   - Table of discovered devices with:
     - Device ID (MAC address)
     - IP address
     - Model (SKU)
     - Last seen timestamp
     - Assign buttons (Red/Blue)
   - Selected device indicators
   - Test buttons for each assigned device

4. **Test Controls**
   - Color buttons: Off, Green, Purple, Red, Blue
   - Apply to: Red Only, Blue Only, Both
   - Real-time status indicators

5. **Save/Cancel Buttons**
   - Save configuration
   - Cancel and return to setup menu

**JavaScript Features**:
- WebSocket connection for real-time device updates
- AJAX calls for discovery and testing
- Form validation
- Visual feedback for device health

#### Step 4.3: Update Web Router
**File**: `cheesy-arena/web/web.go`

**Changes Required** (in `newHandler()` method, around line 127):
```go
func (web *Web) newHandler() http.Handler {
    mux := http.NewServeMux()
    // ... existing routes

    // Add LED setup routes
    mux.HandleFunc("GET /setup/leds", web.ledsGetHandler)
    mux.HandleFunc("POST /setup/leds", web.ledsPostHandler)
    mux.HandleFunc("GET /setup/leds/discover", web.ledsDiscoverHandler)
    mux.HandleFunc("POST /setup/leds/test", web.ledsTestHandler)
    mux.HandleFunc("GET /setup/leds/websocket", web.ledsWebsocketHandler)

    // ... rest of routes
    return mux
}
```

#### Step 4.4: Update Setup Menu
**File**: `cheesy-arena/templates/setup/settings.html` (or main setup page)

**Tasks**:
1. Add "LED Controllers" link to setup menu
2. Add icon/description for LED configuration

### Phase 5: Testing and Validation (Day 4-5)

#### Step 5.1: Unit Tests

**Files to Create/Update**:
- `cheesy-arena/partner/govee_test.go`
- `cheesy-arena/led/govee_controller_test.go`
- `cheesy-arena/led/dmx_controller_test.go` (update existing)
- `cheesy-arena/field/arena_test.go` (update existing)
- `cheesy-arena/web/setup_leds_test.go`

**Test Coverage Goals**:
- Partner package: >80% coverage
- LED package: >80% coverage
- Web handlers: >70% coverage

**Key Test Scenarios**:
1. Govee device discovery with multiple devices
2. Govee command sending with network failures
3. Retry logic verification
4. Device timeout and health monitoring
5. LED controller interface compliance
6. Arena initialization with different controller types
7. Configuration save/load
8. WebSocket communication

#### Step 5.2: Integration Tests

**Test Scenarios**:

1. **DMX to Govee Migration**
   - Start with DMX configuration
   - Switch to Govee via UI
   - Verify controllers are replaced
   - Verify LED commands work

2. **Govee Device Discovery**
   - Start arena with no devices
   - Power on Govee devices
   - Verify devices appear in discovery list
   - Assign devices to red/blue
   - Verify assignment persists

3. **Match Play with Govee LEDs**
   - Configure Govee controllers
   - Load a match
   - Start match
   - Verify LEDs change during match states:
     - PreMatch: Green
     - Auto/Teleop: Red/Blue based on hub state
     - PostMatch: Off → Purple → Green
   - Verify flashing behavior during transitions

4. **Network Failure Recovery**
   - Start match with Govee LEDs working
   - Disconnect one device from WiFi
   - Verify health status updates
   - Reconnect device
   - Verify recovery

5. **Concurrent DMX and Govee**
   - Configure Red as DMX, Blue as Govee
   - Run match
   - Verify both controllers work independently

#### Step 5.3: Manual Testing Procedures

**Test Plan Document**: `cheesy-arena/docs/GOVEE_TESTING.md`

**Manual Test Cases**:

1. **Initial Setup**
   - [ ] Fresh install with no configuration
   - [ ] Default to DMX controller type
   - [ ] Navigate to LED setup page
   - [ ] UI loads without errors

2. **Govee Device Discovery**
   - [ ] Click "Discover Devices" button
   - [ ] Verify devices appear within 5 seconds
   - [ ] Verify device information is accurate (IP, MAC, model)
   - [ ] Verify "Last Seen" timestamp updates
   - [ ] Power off a device, verify it disappears after timeout

3. **Device Assignment**
   - [ ] Assign device to Red controller
   - [ ] Verify device shows as assigned
   - [ ] Assign different device to Blue controller
   - [ ] Verify both assignments persist
   - [ ] Try to assign same device to both (should warn/prevent)

4. **LED Testing**
   - [ ] Test Red LED: Off, Green, Red, Blue, Purple
   - [ ] Test Blue LED: Off, Green, Red, Blue, Purple
   - [ ] Test Both LEDs simultaneously
   - [ ] Verify colors match expected values
   - [ ] Verify response time < 500ms

5. **Configuration Persistence**
   - [ ] Save Govee configuration
   - [ ] Restart Cheesy Arena
   - [ ] Verify configuration loads correctly
   - [ ] Verify devices reconnect automatically

6. **Match Integration**
   - [ ] Load practice match
   - [ ] Verify PreMatch: Both LEDs green
   - [ ] Start match
   - [ ] Verify Auto period: LEDs show hub state
   - [ ] Verify Teleop: LEDs flash during transitions
   - [ ] End match
   - [ ] Verify PostMatch sequence: Off → Purple → Green

7. **Error Handling**
   - [ ] Configure non-existent device ID
   - [ ] Attempt to start match
   - [ ] Verify graceful degradation (no crash)
   - [ ] Verify error message in logs
   - [ ] Verify UI shows device as unhealthy

8. **Performance**
   - [ ] Run 10 consecutive matches
   - [ ] Monitor LED response times
   - [ ] Check for memory leaks
   - [ ] Verify no degradation over time

---

## Database Schema Updates

### EventSettings Table

**New Columns**:
```sql
ALTER TABLE event_settings ADD COLUMN led_controller_type TEXT DEFAULT 'dmx';
ALTER TABLE event_settings ADD COLUMN red_led_address TEXT DEFAULT '';
ALTER TABLE event_settings ADD COLUMN blue_led_address TEXT DEFAULT '';
ALTER TABLE event_settings ADD COLUMN red_led_device_id TEXT DEFAULT '';
ALTER TABLE event_settings ADD COLUMN blue_led_device_id TEXT DEFAULT '';
```

**Migration Strategy**:
- Add columns with default values for backward compatibility
- Existing databases will default to DMX mode
- No data loss or manual migration required

**Implementation Location**: `cheesy-arena/model/event_settings.go`

**Migration Code**:
```go
func migrateEventSettings(db *Database) error {
    // Check if columns exist
    // Add columns if missing
    // Set defaults for existing records
}
```

---

## Testing Strategy

### Unit Testing

**Framework**: Go's built-in `testing` package + `github.com/stretchr/testify`

**Test Organization**:
- One test file per source file (`*_test.go`)
- Table-driven tests for multiple scenarios
- Mock UDP connections for network tests
- Mock GoveeClient for controller tests

**Coverage Requirements**:
- Minimum 75% code coverage for new code
- All public methods must have tests
- All error paths must be tested

**Running Tests**:
```bash
# Run all tests
cd cheesy-arena
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./partner
go test ./led

# Run with race detection
go test -race ./...
```

### Integration Testing

**Test Environment**:
- Local development machine
- 2+ Govee LED devices on same network
- Multicast-enabled network switch/router

**Test Data**:
- Sample event database with test matches
- Known Govee device IDs for testing

**Integration Test Scenarios**:
1. End-to-end match play with Govee LEDs
2. Configuration changes during runtime
3. Device discovery and assignment workflow
4. Network failure and recovery
5. DMX/Govee hybrid configuration

**Test Execution**:
```bash
# Run integration tests (requires hardware)
cd cheesy-arena
go test -tags=integration ./...
```

### Manual Testing

**Test Checklist**: See Phase 5, Step 5.3 above

**Test Environment**:
- Full Cheesy Arena setup
- Real Govee hardware
- Various network conditions (good/poor WiFi)

**Documentation**: `cheesy-arena/docs/GOVEE_TESTING.md`

### Performance Testing

**Metrics to Monitor**:
- LED command latency (target: <200ms)
- Device discovery time (target: <5s)
- Memory usage over extended operation
- CPU usage during match play
- Network bandwidth consumption

**Tools**:
- Go's built-in profiling (`pprof`)
- Network packet capture (Wireshark)
- System monitoring (Activity Monitor / htop)

**Performance Benchmarks**:
```bash
cd cheesy-arena/led
go test -bench=. -benchmem
```

---

## Rollback Plan

### Immediate Rollback (During Development)

**If issues arise during development**:

1. **Revert Git Commits**:
```bash
git log --oneline  # Find commit before integration
git revert <commit-hash>
# OR
git reset --hard <commit-hash>
```

2. **Remove New Files**:
```bash
rm cheesy-arena/partner/govee.go
rm cheesy-arena/partner/govee_test.go
rm cheesy-arena/led/govee_controller.go
rm cheesy-arena/led/govee_controller_test.go
rm cheesy-arena/web/setup_leds.go
rm cheesy-arena/templates/setup_leds.html
```

3. **Restore Original Files**:
```bash
git checkout HEAD -- cheesy-arena/led/controller.go
git checkout HEAD -- cheesy-arena/field/arena.go
git checkout HEAD -- cheesy-arena/model/event_settings.go
git checkout HEAD -- cheesy-arena/web/web.go
```

### Production Rollback

**If issues arise in production**:

1. **Database Rollback** (if schema changed):
```sql
-- Remove new columns (data will be lost)
ALTER TABLE event_settings DROP COLUMN led_controller_type;
ALTER TABLE event_settings DROP COLUMN red_led_address;
ALTER TABLE event_settings DROP COLUMN blue_led_address;
ALTER TABLE event_settings DROP COLUMN red_led_device_id;
ALTER TABLE event_settings DROP COLUMN blue_led_device_id;
```

2. **Configuration Rollback**:
   - Edit `event.db` to set `led_controller_type` back to `"dmx"`
   - Or delete database and restore from backup

3. **Binary Rollback**:
   - Replace `cheesy-arena` binary with previous version
   - Restart application

4. **Graceful Degradation**:
   - If Govee fails, system falls back to DMX
   - No match interruption required
   - Can switch controller type via UI without restart

### Rollback Testing

**Before Production Deployment**:
1. Test rollback procedure in staging environment
2. Verify database migration reversal works
3. Verify application starts with old binary
4. Document rollback time estimate (target: <5 minutes)

### Rollback Decision Criteria

**Trigger rollback if**:
- Critical bugs preventing match play
- Data corruption or loss
- Unacceptable performance degradation (>1s LED latency)
- Security vulnerabilities discovered
- Incompatibility with existing hardware

**Do NOT rollback for**:
- Minor UI issues (can be fixed forward)
- Govee-specific issues (can disable Govee, keep DMX)
- Non-critical bugs (can be patched)

---

## Timeline Estimates

### Detailed Schedule

| Phase | Tasks | Duration | Dependencies |
|-------|-------|----------|--------------|
| **Phase 1: Core Infrastructure** | | **1 day** | |
| 1.1 | Create Govee protocol package | 4 hours | None |
| 1.2 | Create Govee protocol tests | 2 hours | 1.1 |
| 1.3 | Create LED controller interface | 2 hours | None |
| **Phase 2: Govee LED Controller** | | **1 day** | |
| 2.1 | Create Govee LED controller | 4 hours | 1.1, 1.3 |
| 2.2 | Create Govee controller tests | 2 hours | 2.1 |
| 2.3 | Refactor existing DMX controller | 2 hours | 1.3 |
| **Phase 3: Arena Integration** | | **1 day** | |
| 3.1 | Update Arena structure | 2 hours | 2.1, 2.3 |
| 3.2 | Update Event Settings model | 2 hours | None |
| 3.3 | Update Arena run loop | 2 hours | 3.1, 3.2 |
| 3.4 | Integration testing | 2 hours | 3.1-3.3 |
| **Phase 4: Configuration UI** | | **1.5 days** | |
| 4.1 | Create LED setup page backend | 4 hours | 3.2 |
| 4.2 | Create LED setup page frontend | 6 hours | 4.1 |
| 4.3 | Update web router | 1 hour | 4.1 |
| 4.4 | Update setup menu | 1 hour | 4.2 |
| **Phase 5: Testing & Validation** | | **1.5 days** | |
| 5.1 | Unit tests | 4 hours | All phases |
| 5.2 | Integration tests | 4 hours | All phases |
| 5.3 | Manual testing | 4 hours | All phases |
| **Documentation** | | **0.5 days** | |
| | Code documentation | 2 hours | All phases |
| | User guide | 2 hours | Phase 4 |
| **Buffer** | | **0.5 days** | |
| | Unexpected issues | 4 hours | - |

### Total Estimated Time: **6-7 days**

### Milestone Schedule

| Milestone | Completion Date | Deliverables |
|-----------|----------------|--------------|
| M1: Core Protocol | End of Day 1 | Govee client, LED interface |
| M2: LED Controllers | End of Day 2 | Govee & DMX controllers |
| M3: Arena Integration | End of Day 3 | Full backend integration |
| M4: UI Complete | End of Day 5 | Configuration UI functional |
| M5: Testing Complete | End of Day 6 | All tests passing |
| M6: Production Ready | End of Day 7 | Documentation, deployment |

### Critical Path

```
Day 1: Govee Protocol → LED Interface
  ↓
Day 2: Govee Controller → DMX Refactor
  ↓
Day 3: Arena Integration → Event Settings
  ↓
Day 4-5: Configuration UI
  ↓
Day 6-7: Testing & Documentation
```

---

## Risk Mitigation

### Risk Matrix

| Risk | Probability | Impact | Severity | Mitigation Strategy |
|------|-------------|--------|----------|---------------------|
| Govee protocol changes | Low | High | Medium | Version detection, fallback to DMX |
| WiFi reliability issues | High | Medium | High | Retry logic, health monitoring, user warnings |
| Device discovery failures | Medium | Medium | Medium | Manual device ID entry, cached devices |
| Performance degradation | Low | High | Medium | Async operations, performance testing |
| Database migration issues | Low | High | Medium | Backward-compatible schema, migration tests |
| UI complexity | Medium | Low | Low | Iterative design, user testing |
| Integration bugs | Medium | High | High | Comprehensive testing, staged rollout |
| Hardware compatibility | Medium | Medium | Medium | Document supported models, test matrix |

### Detailed Risk Mitigation

#### Risk 1: WiFi Reliability Issues
**Description**: Govee devices use WiFi, which is less reliable than wired DMX.

**Mitigation**:
- Implement 3x retry for critical commands (OFF)
- Add health monitoring with visual indicators
- Provide clear UI warnings when devices are unhealthy
- Document network requirements (dedicated WiFi, no interference)
- Allow fallback to DMX mid-match if needed

**Contingency**:
- If reliability is unacceptable, disable Govee option
- Recommend DMX for official competitions
- Use Govee only for practice/scrimmages

#### Risk 2: Govee Protocol Changes
**Description**: Govee may update their LAN protocol, breaking compatibility.

**Mitigation**:
- Document current protocol version
- Implement protocol version detection
- Add logging for protocol errors
- Monitor Govee firmware updates
- Maintain contact with Govee developer community

**Contingency**:
- Freeze Govee device firmware at known-good version
- Provide protocol update mechanism
- Fall back to DMX if protocol breaks

#### Risk 3: Device Discovery Failures
**Description**: UDP multicast may not work on all networks.

**Mitigation**:
- Provide manual device ID entry option
- Cache discovered devices between sessions
- Document network requirements (multicast support)
- Add network diagnostics tool
- Support static IP configuration

**Contingency**:
- Manual configuration mode
- Pre-configure devices before event
- Use wired network for Govee devices if possible

#### Risk 4: Performance Degradation
**Description**: Additional network I/O may slow down arena loop.

**Mitigation**:
- Use async/non-blocking operations
- Separate goroutine for LED updates
- Performance benchmarking before release
- Monitor arena loop timing
- Set timeout limits for LED operations

**Contingency**:
- Disable Govee if loop timing exceeds threshold
- Optimize update frequency
- Reduce retry attempts if needed

#### Risk 5: Database Migration Issues
**Description**: Schema changes may cause issues with existing databases.

**Mitigation**:
- Use backward-compatible schema changes (ADD COLUMN with defaults)
- Test migration on sample databases
- Provide database backup before migration
- Implement migration rollback
- Version database schema

**Contingency**:
- Restore from backup
- Manual schema fix
- Skip migration, use defaults

#### Risk 6: Integration Bugs
**Description**: New code may introduce bugs in existing functionality.

**Mitigation**:
- Comprehensive unit and integration tests
- Code review before merge
- Staged rollout (dev → staging → production)
- Regression testing of existing features
- Feature flag to disable Govee

**Contingency**:
- Quick rollback procedure (documented above)
- Hotfix process
- Disable Govee via configuration

#### Risk 7: Hardware Compatibility
**Description**: Not all Govee models may support LAN API.

**Mitigation**:
- Document supported models (H6159, etc.)
- Test with multiple Govee models
- Provide compatibility check in UI
- Clear error messages for unsupported devices
- Maintain compatibility matrix

**Contingency**:
- Recommend specific models
- Provide alternative (DMX)
- Community testing for new models

### Risk Monitoring

**During Development**:
- Daily standup to discuss blockers
- Track issues in GitHub/issue tracker
- Weekly risk review

**Post-Deployment**:
- Monitor error logs for Govee-related issues
- Collect user feedback
- Track device health metrics
- Performance monitoring

---

## Success Criteria

### Functional Requirements

**Must Have** (P0):
- [ ] Govee devices can be discovered via UDP multicast
- [ ] Govee devices can be controlled (on/off, color)
- [ ] LED controller interface supports both DMX and Govee
- [ ] Arena can use Govee controllers for hub lights
- [ ] Configuration UI allows selecting controller type
- [ ] Configuration persists across restarts
- [ ] Existing DMX functionality remains unchanged
- [ ] All existing tests pass
- [ ] New code has >75% test coverage

**Should Have** (P1):
- [ ] Device health monitoring and status display
- [ ] Retry logic for unreliable connections
- [ ] Manual device ID entry option
- [ ] Test controls in configuration UI
- [ ] Real-time device discovery updates
- [ ] Graceful degradation on device failure
- [ ] Clear error messages and logging

**Nice to Have** (P2):
- [ ] Device naming/labeling
- [ ] Network diagnostics tool
- [ ] Performance metrics dashboard
- [ ] Automatic device reconnection
- [ ] Firmware version detection

### Performance Requirements

- [ ] LED command latency < 200ms (95th percentile)
- [ ] Device discovery completes within 5 seconds
- [ ] Arena loop timing unaffected (<1ms overhead)
- [ ] Memory usage increase < 10MB
- [ ] No memory leaks over 24-hour operation
- [ ] Supports at least 10 Govee devices simultaneously

### Quality Requirements

- [ ] Zero critical bugs in production
- [ ] Zero data loss or corruption
- [ ] No crashes or panics
- [ ] All error conditions handled gracefully
- [ ] Code passes `go vet` and `golint`
- [ ] Documentation complete and accurate

### User Experience Requirements

- [ ] Configuration UI is intuitive (no training required)
- [ ] Device discovery "just works" on supported networks
- [ ] Clear feedback for all user actions
- [ ] Error messages are actionable
- [ ] Setup time < 5 minutes for new users
- [ ] No manual network configuration required

### Compatibility Requirements

- [ ] Works on macOS, Linux, and Windows
- [ ] Compatible with Go 1.23.0+
- [ ] No breaking changes to existing APIs
- [ ] Existing event databases work without modification
- [ ] DMX users unaffected by Govee addition

### Acceptance Testing

**Test Scenarios**:

1. **Fresh Installation**
   - Install Cheesy Arena on new machine
   - Configure Govee LEDs from scratch
   - Run a complete match
   - **Pass Criteria**: Match completes successfully with correct LED behavior

2. **Upgrade from Previous Version**
   - Start with existing Cheesy Arena installation (DMX configured)
   - Upgrade to version with Govee support
   - Verify DMX still works
   - Switch to Govee
   - **Pass Criteria**: Smooth upgrade, no data loss, both modes work

3. **Network Failure Recovery**
   - Start match with Govee LEDs
   - Disconnect WiFi during match
   - Reconnect WiFi
   - **Pass Criteria**: System detects failure, shows warning, recovers automatically

4. **Performance Under Load**
   - Run 50 consecutive matches
   - Monitor system resources
   - **Pass Criteria**: No degradation, no memory leaks, consistent performance

5. **Multi-Device Support**
   - Configure 4+ Govee devices (2 red, 2 blue)
   - Run match
   - **Pass Criteria**: All devices respond correctly and in sync

### Sign-Off Criteria

**Development Complete**:
- [ ] All P0 and P1 requirements implemented
- [ ] All tests passing (unit, integration, manual)
- [ ] Code reviewed and approved
- [ ] Documentation complete

**Ready for Staging**:
- [ ] Acceptance tests pass
- [ ] Performance requirements met
- [ ] No known critical bugs
- [ ] Rollback plan tested

**Ready for Production**:
- [ ] Staging deployment successful
- [ ] User acceptance testing complete
- [ ] Production deployment plan approved
- [ ] Monitoring and alerting configured

---

## Appendices

### Appendix A: Govee Protocol Reference

**Discovery Protocol**:
```json
// Broadcast to 239.255.255.250:4001
{
  "msg": {
    "cmd": "scan",
    "data": {
      "account_topic": "reserve"
    }
  }
}

// Response on port 4002
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

**Control Protocol**:
```json
// Send to device_ip:4003

// Turn On
{
  "msg": {
    "cmd": "turn",
    "data": {
      "value": 1
    }
  }
}

// Turn Off
{
  "msg": {
    "cmd": "turn",
    "data": {
      "value": 0
    }
  }
}

// Set Color
{
  "msg": {
    "cmd": "color",
    "data": {
      "r": 255,
      "g": 0,
      "b": 0
    }
  }
}
```

### Appendix B: Supported Govee Models

**Confirmed Compatible**:
- H6159 (RGB LED Strip)
- H6163 (RGB LED Strip)
- H6182 (RGBIC LED Strip)

**Likely Compatible** (LAN API support):
- H6199
- H6601
- H6602

**Not Compatible** (Bluetooth-only):
- H6104
- H6105
- H6106

**Testing Required**:
- Any model not listed above
- Check for LAN API support in Govee app

### Appendix C: Network Requirements

**Minimum Requirements**:
- UDP multicast support (IGMP)
- Ports 4001-4003 open for UDP
- Same subnet for Cheesy Arena server and Govee devices
- WiFi or wired network (WiFi for Govee devices)

**Recommended Setup**:
- Dedicated WiFi network for Govee devices
- 5GHz WiFi for lower latency
- Wired connection for Cheesy Arena server
- Managed switch with IGMP snooping enabled
- No firewall blocking UDP multicast

**Network Diagram**:
```
[Cheesy Arena Server] (Wired)
         |
    [Network Switch]
         |
    [WiFi AP] ---- [Govee Device 1]
         |
         +-------- [Govee Device 2]
```

### Appendix D: Troubleshooting Guide

**Problem**: Devices not discovered

**Solutions**:
1. Check network supports multicast (ping 239.255.255.250)
2. Verify devices are on same subnet
3. Check firewall settings
4. Try manual device ID entry
5. Restart Govee devices

**Problem**: High latency (>500ms)

**Solutions**:
1. Check WiFi signal strength
2. Reduce WiFi interference
3. Use 5GHz WiFi instead of 2.4GHz
4. Reduce number of devices
5. Check network congestion

**Problem**: Devices disconnect during match

**Solutions**:
1. Check WiFi stability
2. Verify power supply to devices
3. Update Govee firmware
4. Reduce retry count
5. Switch to DMX for critical events

**Problem**: Colors don't match expected

**Solutions**:
1. Run color preset initialization
2. Check device color calibration in Govee app
3. Verify RGB values in code
4. Test with different colors
5. Check device model compatibility

### Appendix E: References

**Govee LAN API**:
- Community documentation: https://github.com/egold555/Govee-Reverse-Engineering
- Protocol analysis from AM Showcase Stream Overlay project

**Cheesy Arena**:
- GitHub: https://github.com/Team254/cheesy-arena
- Documentation: https://github.com/Team254/cheesy-arena/wiki

**DMX/sACN Protocol**:
- E1.31 Specification: https://tsp.esta.org/tsp/documents/docs/ANSI_E1-31-2018.pdf
- sACN Overview: https://www.lightjams.com/sacn.html

**Go Networking**:
- net package: https://pkg.go.dev/net
- UDP multicast: https://pkg.go.dev/net#UDPConn

---

## Document History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-02-18 | Integration Team | Initial plan created |

---

## Approval

**Plan Created By**: AI Assistant
**Date**: 2026-02-18
**Status**: Awaiting Review

**Reviewers**:
- [ ] Technical Lead
- [ ] Project Manager
- [ ] QA Lead

**Approval Required Before**: Code implementation begins

---

*End of Integration Plan*

