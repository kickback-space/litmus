package litmus

import (
    "sync"
    "time"
)

const (
    defaultPacketSize = 1200  // bytes, typical MTU size
)

// NetworkCapability represents the measured network performance characteristics
type NetworkCapability struct {
    MaxStableBitrate  int     // kbps
    PacketLossRate    float64 // Measured packet loss rate
    Jitter           float64 // Measured jitter in milliseconds
}

// NetworkTuner manages the network capability discovery process
type NetworkTuner struct {
    currentBitrate    int
    maxBitrate        int
    stepSize          int
    lastAdjustment    time.Time
    stableCount       int
    failureCount      int
    testComplete      bool
    bestStable        NetworkCapability
    mu                sync.Mutex
}

func NewNetworkTuner(initialBitrate, maxBitrate, stepSize int) *NetworkTuner {
    return &NetworkTuner{
        currentBitrate: initialBitrate,
        maxBitrate:    maxBitrate,
        stepSize:      stepSize,
        lastAdjustment: time.Now(),
    }
}

func (nt *NetworkTuner) getCurrentBitrate() int {
    nt.mu.Lock()
    defer nt.mu.Unlock()
    return nt.currentBitrate
}

func (nt *NetworkTuner) IsTestComplete() bool {
    nt.mu.Lock()
    defer nt.mu.Unlock()
    return nt.testComplete
}

func (nt *NetworkTuner) GetCapability() NetworkCapability {
    nt.mu.Lock()
    defer nt.mu.Unlock()
    return nt.bestStable
}

func (nt *NetworkTuner) adjustBitrate(lossRate, jitter float64) bool {
    nt.mu.Lock()
    defer nt.mu.Unlock()

    now := time.Now()
    if now.Sub(nt.lastAdjustment) < adaptInterval {
        return !nt.testComplete
    }
    nt.lastAdjustment = now

    // Check if current bitrate is stable
    if lossRate <= 0.01 && jitter <= 20.0 {
        nt.stableCount++
        nt.failureCount = 0

        // Update best stable configuration
        if nt.stableCount >= requiredStableIntervals {
            nt.bestStable = NetworkCapability{
                MaxStableBitrate: nt.currentBitrate,
                PacketLossRate:   lossRate,
                Jitter:          jitter,
            }

            // Try higher bitrate if not at max
            if nt.currentBitrate < nt.maxBitrate {
                nt.currentBitrate += nt.stepSize
                nt.stableCount = 0
            } else {
                nt.testComplete = true
                return false
            }
        }
    } else {
        nt.failureCount++
        nt.stableCount = 0

        if nt.failureCount >= requiredFailureIntervals {
            // Step down bitrate
            nt.currentBitrate -= nt.stepSize
            if nt.currentBitrate < 1000 { // Minimum 1 Mbps
                nt.testComplete = true
                return false
            }
            nt.failureCount = 0
        }
    }

    return !nt.testComplete
}