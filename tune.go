// adaptive_session.go
package litmus

import (
    "time"
    "errors"

    . "github.com/blitz-frost/log"
)

const (
    adaptInterval              = 500 * time.Millisecond
    requiredStableIntervals    = 4
    requiredFailureIntervals   = 2
)

type SessionTuner struct {
    profiles         []VideoProfile
    currentProfile   int
    stats            PacketStats
    lastAdjustment   time.Time
    stableCount      int
    failureCount     int
    testComplete     bool
    profileTestStart time.Time
}

type PacketStats struct {
    sent             uint32
    expectedSequence uint32
    lost             uint32
    timestamp        time.Time
    windowStart      uint32
}

func NewSessionTuner(profiles []VideoProfile) *SessionTuner {
    return &SessionTuner{
        profiles:         profiles,
        currentProfile:   0, // Start with the highest-quality profile
        lastAdjustment:   time.Now(),
        profileTestStart: time.Now(),
        stats: PacketStats{
            timestamp: time.Now(),
        },
        failureCount: 0,
    }
}

func (as *SessionTuner) getCurrentProfile() (VideoProfile, error) {
    if as.currentProfile >= len(as.profiles) {
        Log(Info, "getCurrentProfile called with out-of-bounds index",
            Entry{"currentProfile", as.currentProfile},
            Entry{"profilesLength", len(as.profiles)})
        return VideoProfile{}, errors.New("no suitable profiles")
    }
    profile := as.profiles[as.currentProfile]

    return profile, nil
}

func (as *SessionTuner) adjustProfile(currentLoss, currentJitter float64) {
    now := time.Now()
    if now.Sub(as.lastAdjustment) < adaptInterval {
        return
    }
    as.lastAdjustment = now

    profile, err := as.getCurrentProfile()
    if err != nil {
        // All profiles have been tested; select the smallest profile
        if len(as.profiles) > 0 {
            as.currentProfile = len(as.profiles) - 1
            profile = as.profiles[as.currentProfile]
            as.testComplete = true
            Log(Info, "All profiles tested; selecting smallest profile",
                Entry{"selected_profile", profile.Name})
        } 
    }

    // Check if the network conditions meet the acceptable thresholds
    if currentLoss <= profile.AcceptablePacketLoss && currentJitter <= profile.AcceptableJitter {
        as.stableCount++
        as.failureCount = 0 // Reset failureCount on success

        // Require sustained stability before confirming profile support
        if as.stableCount >= requiredStableIntervals {
            as.testComplete = true
        }
    } else {
        as.failureCount++
        as.stableCount = 0 // Reset stableCount on failure

        if as.failureCount >= requiredFailureIntervals {
            // Move to the next lower-quality profile if available
            as.currentProfile++
            if as.currentProfile >= len(as.profiles) {
                if len(as.profiles) > 0 {
                    as.currentProfile = len(as.profiles) - 1
                    profile = as.profiles[as.currentProfile]
                } else {
                    Log(Error, "No profiles available to select")
                }
                as.testComplete = true
                return
            }

            // Reset counters for the new profile
            as.failureCount = 0
            as.stableCount = 0
            as.profileTestStart = now
        }
    }
}

