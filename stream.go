// litmus/stream.go
package litmus

import (
    "context"
    "crypto/rand"
    "encoding/binary"
    "time"

    "github.com/pion/webrtc/v3"
    . "github.com/blitz-frost/log"
)

const (
    headerSize      = 12
    maxTestDuration = 200 * time.Second
)

func stream(ctx context.Context, dc *webrtc.DataChannel, connID string, testDone chan struct{}, testError chan error, networkTuner *NetworkTuner, peerConnection *webrtc.PeerConnection) {
    startTime := time.Now()
    sequence := uint32(0)

    ticker := time.NewTicker(time.Millisecond * 100)
    defer ticker.Stop()

    defer func() {
        dc.Close()
        close(testDone)
        peerConnection.Close()
    }()

    calculatePacketRate := func(bitrate int) (packetSize int, packetsPerSecond int) {
        packetSize = defaultPacketSize
        bitsPerPacket := packetSize * 8
        packetsPerSecond = (bitrate * 1000) / bitsPerPacket
        return
    }

   var lastBufferedAmount uint64
   var totalBytesSent uint64
   lastCheckTime := time.Now()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if networkTuner.IsTestComplete() {
                Log(Info, "Network testing complete", Entry{"connID", connID})
                return
            }

            currentBitrate := networkTuner.getCurrentBitrate()
            packetSize, packetsPerSecond := calculatePacketRate(currentBitrate)

            // Adjust send rate interval
            if packetsPerSecond > 0 {
                newInterval := time.Second / time.Duration(packetsPerSecond)
                ticker.Reset(newInterval)
            }

            // Create and send test packet
            packet := make([]byte, packetSize)
            binary.BigEndian.PutUint32(packet[0:headerSize-8], sequence)
            binary.BigEndian.PutUint64(packet[headerSize-8:headerSize], uint64(time.Now().UnixNano()))

            if _, err := rand.Read(packet[headerSize:]); err != nil {
                Log(Error, "Failed to generate random data",
                    Entry{"error", err},
                    Entry{"connID", connID})
                testError <- err
                return
            }

            if err := dc.Send(packet); err != nil {
                Log(Error, "Failed to send test packet",
                    Entry{"error", err},
                    Entry{"connID", connID})
                testError <- err
                return
            }
            totalBytesSent += uint64(packetSize)

            sequence++

            // Check buffered amount to see if data is actually going out or backing up
            currentBuffered := dc.BufferedAmount()

            elapsed := time.Since(lastCheckTime).Milliseconds()
            if elapsed >= 100 {
                // Calculate actual bytes transmitted (accounting for buffer changes)
                bufferChange := int64(currentBuffered) - int64(lastBufferedAmount)
                actualBytesSent := int64(totalBytesSent)
                
                // If buffer decreased, those bytes were sent too
                if bufferChange < 0 {
                    actualBytesSent += -bufferChange
                }
                
                // Calculate effective rate in bits per second
                effectiveSentBitsPerSec := float64(actualBytesSent) * 8.0 / (float64(elapsed)/1000.0)
                
                Log(Info, "Throughput calculation", Entries{
                    {"totalBytesSent", totalBytesSent},
                    {"bufferChange", bufferChange},
                    {"actualBytesSent", actualBytesSent},
                    {"effectiveRate", effectiveSentBitsPerSec},
                })

                networkTuner.SetServerEffectiveRate(effectiveSentBitsPerSec)

                // Reset counters for next interval
                lastBufferedAmount = currentBuffered
                totalBytesSent = 0
                lastCheckTime = time.Now()
            }


            if time.Since(startTime) >= maxTestDuration {
                Log(Info, "Max test duration reached", Entry{"connID", connID})
                return
            }
        }
    }
}
