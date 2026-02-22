// Copyright 2022 Team 254. All Rights Reserved.
// Author: pat@patfairbank.com (Patrick Fairbank)

package game

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScoreSummaryDetermineMatchStatus(t *testing.T) {
	redScoreSummary := &ScoreSummary{Score: 10}
	blueScoreSummary := &ScoreSummary{Score: 10}
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	redScoreSummary.Score = 11
	assert.Equal(t, RedWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, RedWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	blueScoreSummary.Score = 12
	assert.Equal(t, BlueWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, BlueWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Test playoff tiebreakers: NumOpponentMajorFouls → AutoPoints → TowerPoints → ActiveFuel
	redScoreSummary.Score = 12
	redScoreSummary.NumOpponentMajorFouls = 11
	redScoreSummary.AutoPoints = 11
	redScoreSummary.AutoClimbPoints = 30
	redScoreSummary.TeleopClimbPoints = 60
	redScoreSummary.ActiveFuel = 50
	blueScoreSummary.NumOpponentMajorFouls = 10
	blueScoreSummary.AutoPoints = 10
	blueScoreSummary.AutoClimbPoints = 30
	blueScoreSummary.TeleopClimbPoints = 60
	blueScoreSummary.ActiveFuel = 50
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, RedWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Blue has more opponent major fouls (wins tiebreaker)
	blueScoreSummary.NumOpponentMajorFouls = 12
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, BlueWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Tied on major fouls, red has more auto points (wins tiebreaker)
	redScoreSummary.NumOpponentMajorFouls = 12
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, RedWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Tied on major fouls, blue has more auto points (wins tiebreaker)
	blueScoreSummary.AutoPoints = 12
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, BlueWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Tied on major fouls and auto points, red has more tower points (wins tiebreaker)
	redScoreSummary.AutoPoints = 12
	redScoreSummary.TeleopClimbPoints = 70 // Red total tower: 30+70=100, Blue: 30+60=90
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, RedWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Tied on major fouls, auto points, and tower points, blue has more active fuel (wins tiebreaker)
	redScoreSummary.TeleopClimbPoints = 60 // Red total tower: 30+60=90, Blue: 30+60=90
	blueScoreSummary.ActiveFuel = 60
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, BlueWonMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))

	// Tied on all tiebreakers
	redScoreSummary.ActiveFuel = 60
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, false))
	assert.Equal(t, TieMatch, DetermineMatchStatus(redScoreSummary, blueScoreSummary, true))
}
