// Copyright 2014 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventSettingsReadWrite(t *testing.T) {
	db := setupTestDb(t)
	defer db.Close()

	eventSettings, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(
		t,
		EventSettings{
			Id:                          1,
			Name:                        "Untitled Event",
			PlayoffType:                 DoubleEliminationPlayoff,
			NumPlayoffAlliances:         8,
			SelectionRound2Order:        "L",
			SelectionRound3Order:        "",
			SelectionShowUnpickedTeams:  true,
			TbaDownloadEnabled:          true,
			ApChannel:                   36,
			SCCUpCommands:               "configure terminal\ninterface range gigabitEthernet 1/2-4\nno shutdown\nexit\nexit\nexit",
			SCCDownCommands:             "configure terminal\ninterface range gigabitEthernet 1/2-4\nshutdown\nexit\nexit\nexit",
			WarmupDurationSec:           0,
			AutoDurationSec:             20,
			PauseDurationSec:            3,
			TeleopDurationSec:           140,
			WarningRemainingDurationSec: 20,
			EnergizedRPThreshold:        100,
			SuperchargedRPThreshold:     360,
			TraversalRPThreshold:        50,
			DMXAddress:                  "10.0.100.80",
			LedControllerType:           "dmx",
			RedLedAddress:               "",
			BlueLedAddress:              "",
			RedLedDeviceId:              "",
			BlueLedDeviceId:             "",
			CompanionAddress:            "",
			CompanionPort:               0,
		},
		*eventSettings,
	)

	eventSettings.Name = "Chezy Champs"
	eventSettings.NumPlayoffAlliances = 6
	eventSettings.SelectionRound2Order = "F"
	eventSettings.SelectionRound3Order = "L"
	err = db.UpdateEventSettings(eventSettings)
	assert.Nil(t, err)
	eventSettings2, err := db.GetEventSettings()
	assert.Nil(t, err)
	assert.Equal(t, eventSettings, eventSettings2)
}
