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
    maxTestDuration = 15 * time.Second
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

            // Adjust send rate
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

            sequence++

            if time.Since(startTime) >= maxTestDuration {
                Log(Info, "Max test duration reached", Entry{"connID", connID})
                return
            }
        }
    }
}