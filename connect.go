// litmus/connect.go
package litmus

import (
    "context"
    "net/http"
    "sync"
    "strconv"
    "time"

    "github.com/gorilla/websocket"
    "github.com/pion/webrtc/v3"
    . "github.com/blitz-frost/log"
    "errors"
)

var ErrConnectionFailed = errors.New("webrtc connection closed")

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true
    },
}

func (s *Server) handleConnection(w http.ResponseWriter, r *http.Request) error {
    ws, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        Log(Error, "network litmus websocket upgrade failed", Entry{"error", err})
        return err
    }
    defer ws.Close()

    var wsWriteMutex sync.Mutex
    writeJSON := func(v interface{}) error {
        wsWriteMutex.Lock()
        defer wsWriteMutex.Unlock()
        return ws.WriteJSON(v)
    }

    config := webrtc.Configuration{
        ICEServers: []webrtc.ICEServer{
            {
                URLs: []string{"stun:stun.l.google.com:19302"},
            },
        },
    }

    peerConnection, err := webrtc.NewPeerConnection(config)
    if err != nil {
        Log(Error, "network litmus peer connection failed", Entry{"error", err})
        return err
    }
    defer peerConnection.Close()

    connID := randomConnID()
    s.connections.Store(connID, peerConnection)
    defer s.connections.Delete(connID)

    testDone := make(chan struct{})
    testError := make(chan error, 1)
    
    // Initialize NetworkTuner with default settings
    networkTuner := NewNetworkTuner(
        2000,    // Start at 1 Mbps
        20000,   // Max 20 Mbps
        1000,    // 1 Mbps steps
    )

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
        if state == webrtc.PeerConnectionStateClosed ||
           state == webrtc.PeerConnectionStateFailed ||
           state == webrtc.PeerConnectionStateDisconnected {
            s.connections.Delete(connID)
            if state == webrtc.PeerConnectionStateFailed {
                select {
                case testError <- ErrConnectionFailed:
                default:
                    // Channel might be full or closed
                }
            }
            cancel()
        }
    })

    peerConnection.OnICECandidate(func(i *webrtc.ICECandidate) {
        if i == nil {
            return
        }
        
        if err := writeJSON(map[string]interface{}{
            "type": "candidate",
            "candidate": i.ToJSON(),
        }); err != nil {
            Log(Error, "failed to send ICE candidate", 
                Entry{"error", err},
                Entry{"connID", connID})
        }
    })
    
    peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
        go stream(ctx, dc, connID, testDone, testError, networkTuner, peerConnection)
    
        dc.OnClose(func() {
            cancel()
        })
    })

    for {
        select {
        case err := <-testError:
            return err
        case <-testDone:
            return nil
        default:
            var msg map[string]interface{}
            if err := ws.ReadJSON(&msg); err != nil {
                if websocket.IsUnexpectedCloseError(err, 
                    websocket.CloseGoingAway,
                    websocket.CloseNoStatusReceived) {
                    Log(Error, "unexpected websocket close", 
                        Entry{"error", err},
                        Entry{"connID", connID})
                    return err
                }
                return nil
            }

            msgType, ok := msg["type"].(string)
            if !ok {
                Log(Error, "invalid message type", 
                    Entry{"connID", connID})
                continue
            }

            switch msgType {
            case "metrics_report":
                lossRate, _ := msg["loss_rate"].(float64)
                jitter, _ := msg["jitter"].(float64)
                actualThroughput, _ := msg["actual_throughput"].(float64)


                serverEffectiveRate := networkTuner.GetServerEffectiveRate()
                shouldContinue := networkTuner.adjustBitrate(lossRate, jitter, actualThroughput, serverEffectiveRate)
                
                // Send current state back to client
                if err := writeJSON(map[string]interface{}{
                    "type":    "bitrate_update",
                    "bitrate": networkTuner.getCurrentBitrate(),
                    "final":   !shouldContinue,
                }); err != nil {
                    Log(Error, "Failed to send bitrate update",
                        Entry{"error", err},
                        Entry{"connID", connID})
                    return err
                }

                // If test is complete, send final results
                if !shouldContinue {
                    capability := networkTuner.GetCapability()
                    if err := writeJSON(map[string]interface{}{
                        "type":    "test_complete",
                        "bitrate": capability.MaxStableBitrate,
                        "final":   true,
                    }); err != nil {
                        Log(Error, "Failed to send test complete message",
                            Entry{"error", err},
                            Entry{"connID", connID})
                        return err
                    }
                }

            case "offer":
                if err := peerConnection.SetRemoteDescription(
                    webrtc.SessionDescription{
                        Type: webrtc.SDPTypeOffer,
                        SDP:  msg["sdp"].(string),
                    },
                ); err != nil {
                    Log(Error, "network test set remote description failed", 
                        Entry{"error", err},
                        Entry{"connID", connID})
                    return err
                }

                answer, err := peerConnection.CreateAnswer(nil)
                if err != nil {
                    Log(Error, "network test create answer failed", 
                        Entry{"error", err},
                        Entry{"connID", connID})
                    return err
                }

                if err = peerConnection.SetLocalDescription(answer); err != nil {
                    Log(Error, "network test set local description failed", 
                        Entry{"error", err},
                        Entry{"connID", connID})
                    return err
                }

                if err := writeJSON(map[string]interface{}{
                    "type": "answer",
                    "sdp":  answer.SDP,
                }); err != nil {
                    return err
                }

            case "candidate":
                candidate, ok := msg["candidate"].(map[string]interface{})
                if !ok {
                    Log(Error, "invalid candidate format",
                        Entry{"connID", connID})
                    continue
                }
                if err := peerConnection.AddICECandidate(webrtc.ICECandidateInit{
                    Candidate: candidate["candidate"].(string),
                }); err != nil {
                    Log(Error, "network test add ice candidate failed", 
                        Entry{"error", err},
                        Entry{"connID", connID})
                    return err
                }
            }
        }
    }
}

func randomConnID() string {
    return strconv.FormatInt(time.Now().UnixNano(), 36)
}