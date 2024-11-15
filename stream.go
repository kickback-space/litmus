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

func stream(ctx context.Context, dc *webrtc.DataChannel, connID string, testDone chan struct{}, testError chan error, SessionTuner *SessionTuner, peerConnection *webrtc.PeerConnection) {
    startTime := time.Now()
    sequence := uint32(0)

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()

    defer func() {
        dc.Close()
        close(testDone)
        peerConnection.Close()
    }()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if SessionTuner.testComplete {
                Log(Info, "Test completed successfully", Entry{"connID", connID})
                return
            }

            profile, err := SessionTuner.getCurrentProfile()
            if err != nil {
                continue
            }

            size := profile.PacketSize
            packetsPerSecond := profile.PacketsPerSecond

            if packetsPerSecond > 0 {
                ticker.Reset(time.Second / time.Duration(packetsPerSecond))
            }

            packet := make([]byte, size)
            binary.BigEndian.PutUint32(packet[0:headerSize-8], sequence)
            binary.BigEndian.PutUint64(packet[headerSize-8:headerSize], uint64(time.Now().UnixNano()))

            _, err = rand.Read(packet[headerSize:])
            if err != nil {
                Log(Error, "Failed to generate random data",
                    Entry{"error", err},
                    Entry{"connID", connID})
                testError <- err
                return
            }

            if err := dc.Send(packet); err != nil {
                Log(Error, "Failed to send test packet",
                    Entry{"error", err},
                    Entry{"sequence", sequence},
                    Entry{"connID", connID})

                testError <- err
                return
            }
            sequence++

            // Check if maxTestDuration is reached
            if time.Since(startTime) >= maxTestDuration {
                Log(Info, "Max test duration reached", Entry{"connID", connID})
                return
            }
        }
    }
}
