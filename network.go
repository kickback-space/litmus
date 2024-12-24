// litmus/network.go
package litmus

import (
	"sync"
	"time"
	"math"
)

const (
	defaultPacketSize          = 1200  // bytes, typical MTU size
	adaptInterval              = 200 * time.Millisecond
	requiredStableIntervals    = 8
	requiredFailureIntervals   = 4
	requiredDeviationIntervals = 4
	MinimumThroughputRatio     = 0.95   // 95% min throughput between server & client 
	MaxStablePacketLossRate    = 0.01   // 1% packet loss threshold
	MaxStableJitter            = 20.0   // milliseconds
	MaxEffectiveRateDeviation  = 30.0   // percent
	StepUpEffectiveDeviation   = 30.0   // percent threshold for stepping up
)

// NetworkCapability represents the measured network performance characteristics
type NetworkCapability struct {
	MaxStableBitrate  int     // kbps
	PacketLossRate    float64 // Measured packet loss rate
	Jitter           float64  // Measured jitter in milliseconds
}

// NetworkTuner manages the network capability discovery process
type NetworkTuner struct {
	currentBitrate      int
	maxBitrate         int
	stepSize           int
	lastAdjustment     time.Time
	stableCount        int
	failureCount       int
	deviationCount     int
	testComplete       bool
	bestStable         NetworkCapability
	mu                 sync.Mutex
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

	clientToServerEffectiveRatio := 0.0
	if serverEffectiveRate > 0 {
		clientToServerEffectiveRatio = (actualThroughput / serverEffectiveRate) * 100
	}

	targetBitrateInBps := float64(nt.currentBitrate) * 1000

	// Calculate deviation of server effective rate from target bitrate
	effectiveRateDeviation := 0.0
	if serverEffectiveRate > 0 {
		effectiveRateDeviation = math.Abs((targetBitrateInBps - serverEffectiveRate) / targetBitrateInBps * 100)
	}

	if effectiveRateDeviation > MaxEffectiveRateDeviation {
		nt.deviationCount++

		// If we see consistent high deviation, step down regardless of other metrics
		if nt.deviationCount >= requiredDeviationIntervals {
			// Step down bitrate
			newBitrate := int(serverEffectiveRate / 1000) // Convert to kbps
			
			if newBitrate < nt.currentBitrate-nt.stepSize {
				newBitrate = nt.currentBitrate - nt.stepSize
			}
			
			nt.currentBitrate = newBitrate
			nt.deviationCount = 0
			nt.stableCount = 0
			
			if nt.currentBitrate < 1000 { // Minimum 1 Mbps
				nt.testComplete = true
				return false
			}
			
			nt.bestStable = NetworkCapability{
				MaxStableBitrate: newBitrate,
				PacketLossRate:   lossRate,
				Jitter:           jitter,
			}
			
			return true
		}
	} else {
		nt.deviationCount = 0
	}

	// Check if current bitrate is stable
	if lossRate <= MaxStablePacketLossRate && 
	   jitter <= MaxStableJitter && 
	   effectiveRateDeviation <= MaxEffectiveRateDeviation &&
	   clientToServerEffectiveRatio >= MinimumThroughputRatio {
		nt.stableCount++
		nt.failureCount = 0

		if nt.stableCount >= requiredStableIntervals {
			currentCapability := NetworkCapability{
				MaxStableBitrate: nt.currentBitrate,
				PacketLossRate:   lossRate,
				Jitter:           jitter,
			}

			if nt.bestStable.MaxStableBitrate > 0 {
				lastThroughput := float64(nt.bestStable.MaxStableBitrate)
				currentThroughput := float64(currentCapability.MaxStableBitrate)

				if currentThroughput < lastThroughput*1.05 {
					nt.testComplete = true
					return false
				}
			}

			nt.bestStable = currentCapability

			// Try higher bitrate if not at max and deviation is low
			if nt.currentBitrate < nt.maxBitrate && effectiveRateDeviation < StepUpEffectiveDeviation {
				nt.currentBitrate += nt.stepSize
				nt.stableCount = 0
			} else {
				// We reached maxBitrate or high deviation
				nt.testComplete = true
				return false
			}
		}
	} else {
		nt.failureCount++
		nt.stableCount = 0

		if nt.failureCount >= requiredFailureIntervals {
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