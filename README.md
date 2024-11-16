# Litmus

Litmus is a Go package that provides adaptive network quality testing for WebRTC apps. It automatically determines the optimal video profile for a given network connection by testing different quality levels and measuring network performance.

## Features

- Adaptive video profile testing
- Real-time network metrics monitoring (packet loss, jitter)
- WebRTC-based testing using DataChannels
- Multiple pre-configured video profiles (540p to 1152p)
- Automatic quality adjustment based on network conditions

## Installation

```bash
go get github.com/kickback-space/litmus
```

## Quick Start StandAlone

```go
package main

import (
    "github.com/kickback-space/litmus"
)

func main() {
    // Start the Litmus server on port 8080
    server := litmus.NewServer(8000)

    err := server.ListenStandalone(path)
}
```

## How It Works

1. Starts with the highest quality video profile
2. Sends test packets matching the profile's bitrate and framerate
3. Monitors network performance (packet loss and jitter)
4. Automatically adjusts to lower quality profiles if network conditions are insufficient
5. Determines the highest sustainable quality level for the connection

The test completes when either:
- A profile is stable for N consecutive intervals
- A profile fails for M consecutive intervals
- The maximum test duration (X seconds) is reached

## Video Profiles

Includes pre-configured profiles ranging from:
- 1152p (2048x1152) at 30/24fps
- 1080p (1920x1080) at 30/24fps
- 960p (1440x960) at 30/24fps
- 720p (1280x720) at 30fps
- 540p (960x540) at 30/24fps

Each profile includes specific bitrate targets and network performance thresholds.

## API

### HTTP Endpoints

- `/litmus` - Main WebSocket endpoint for test connections
- `/litmus/health` - Health check endpoint

### WebSocket Messages

The server accepts the following message types:
- `offer` - WebRTC offer for connection establishment
- `candidate` - ICE candidates for peer connection
- `metrics_report` - Network performance metrics
