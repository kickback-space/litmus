// litmus/network.go
package litmus

import (
    "sync"
    "time"
    "fmt"
    . "github.com/blitz-frost/log"

)

const (
    defaultPacketSize = 1200  // bytes, typical MTU size
    adaptInterval              = 200 * time.Millisecond
    requiredStableIntervals    = 8
    requiredFailureIntervals   = 4
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
    serverEffectiveRate float64
}

func NewNetworkTuner(initialBitrate, maxBitrate, stepSize int) *NetworkTuner {
    return &NetworkTuner{
        currentBitrate: initialBitrate,
        maxBitrate:    maxBitrate,
        stepSize:      stepSize,
        lastAdjustment: time.Now(),
    }
}

func (nt *NetworkTuner) SetServerEffectiveRate(rate float64) {
    nt.mu.Lock()
    defer nt.mu.Unlock()
    nt.serverEffectiveRate = rate
}

func (nt *NetworkTuner) GetServerEffectiveRate() float64 {
    nt.mu.Lock()
    defer nt.mu.Unlock()
    return nt.serverEffectiveRate
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

func (nt *NetworkTuner) adjustBitrate(lossRate, jitter, actualThroughput, serverEffectiveRate float64) bool {
    nt.mu.Lock()
    defer nt.mu.Unlock()

    now := time.Now()
    if now.Sub(nt.lastAdjustment) < adaptInterval {
        return !nt.testComplete
    }
    nt.lastAdjustment = now

 //   expectedBitsPerSec := nt.currentBitrate * 1000
  //  serverRatio := (serverEffectiveRate / float64(expectedBitsPerSec)) * 100


    //percentageThroughput := (actualThroughput / float64(expectedBitsPerSec)) * 100

    clientToServerEffectiveRatio := (actualThroughput / serverEffectiveRate) * 100

    // Log(Info, "loss rate", Entry{"loss rate", lossRate})
    // Log(Info, "jitter", Entry{"jitter", jitter})
    // Log(Info, "expected_bits_per_sec", Entry{"value", expectedBitsPerSec})
    // Log(Info, "server effective rate", Entry{"value", serverEffectiveRate})

    // Log(Info, "server ratio", Entry{"value", fmt.Sprintf("%.2f%%", serverRatio)})
    // Log(Info, "actual_throughput", Entry{"value", actualThroughput})
    // Log(Info, "percentage_throughput", Entry{"value", fmt.Sprintf("%.2f%%", percentageThroughput)})
    Log(Info, "diff from through throughput", Entry{"diff", fmt.Sprintf("%.2f%%", clientToServerEffectiveRatio)})
    

    // if actualThroughput < float64(expectedBitsPerSec)*0.8 {
    //     nt.failureCount++
    //     nt.stableCount = 0
    //     if nt.failureCount >= requiredFailureIntervals {
    //         nt.currentBitrate -= nt.stepSize
    //         if nt.currentBitrate < 1000 {
    //             // nt.test_complete = true
    //             // return false
    //             return true
    //         }
    //     }
    // }

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
                // TODO uncomment
                // nt.testComplete = true
                return true
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