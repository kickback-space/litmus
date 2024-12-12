// ConnectionManager.js
class ConnectionManager {
  constructor() {
    this.peerConnection = null;
    this.dataChannel = null;
    this.webSocket = null;
    this.pendingCandidates = [];
    this.onMetricsCallback = null;
    this.onStateChangeCallback = null;
    this.onTestCompleteCallback = null;
    this.onBitrateUpdateCallback = null;
    this.connectionState = 'disconnected';
  }

  sendMetricsReport(report) {
    if (this.webSocket?.readyState === WebSocket.OPEN) {
      this.webSocket.send(JSON.stringify(report));
    }
  }
  
  async connect(hostAddress, useSsl = false) {
    try {
      this.updateState('connecting');
      await this.setupPeerConnection();
      await this.setupWebSocket(hostAddress, useSsl);
      return true;
    } catch (error) {
      console.error('Connection failed:', error);
      this.updateState('failed');
      throw error;
    }
  }

  async setupPeerConnection() {
    this.peerConnection = new RTCPeerConnection(NetworkConfig.rtcConfig);
    
    this.dataChannel = this.peerConnection.createDataChannel(
      "networkTest",
      NetworkConfig.dataChannelConfig
    );

    this.setupDataChannelHandlers();
    this.setupPeerConnectionHandlers();

    const offer = await this.peerConnection.createOffer();
    await this.peerConnection.setLocalDescription(offer);
    
    return offer;
  }

  setupWebSocket(hostAddress, useSsl) {
    const protocol = useSsl ? 'wss' : 'ws';
    this.webSocket = new WebSocket(`${protocol}://${hostAddress}/litmus`);

    return new Promise((resolve, reject) => {
      this.webSocket.onopen = () => {
        this.sendOffer();
        this.sendPendingCandidates();
        resolve();
      };

      this.webSocket.onmessage = async (event) => {
        const response = JSON.parse(event.data);
        await this.handleWebSocketMessage(response);
      };

      this.webSocket.onerror = (error) => {
        this.updateState('failed');
        reject(error);
      };

      this.webSocket.onclose = () => {
        this.updateState('disconnected');
      };
    });
  }

  async handleWebSocketMessage(response) {
    try {
      switch (response.type) {
        case 'answer':
          await this.peerConnection.setRemoteDescription(
            new RTCSessionDescription(response)
          );
          break;

        case 'candidate':
          await this.peerConnection.addIceCandidate(
            new RTCIceCandidate(response.candidate)
          );
          break;

        case 'bitrate_update':
          if (this.onBitrateUpdateCallback) {
            this.onBitrateUpdateCallback(response.bitrate);
          }
          break;

        case 'test_complete':
          if (this.onTestCompleteCallback) {
            this.onTestCompleteCallback(response.bitrate);
          }
          break;

        default:
          console.warn('Unknown message type:', response.type);
      }
    } catch (error) {
      console.error('Error handling WebSocket message:', error);
      this.updateState('failed');
    }
  }

  setupDataChannelHandlers() {
    this.dataChannel.onopen = () => {
      this.updateState('connected');
    };

    this.dataChannel.onclose = () => {
      this.updateState('disconnected');
    };

    this.dataChannel.onmessage = (event) => {
      if (this.onMetricsCallback) {
        this.processMetrics(event.data);
      }
    };

    this.dataChannel.onerror = (error) => {
      console.error('Data channel error:', error);
      this.updateState('failed');
    };
  }

  setupPeerConnectionHandlers() {
    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        this.handleIceCandidate(event.candidate);
      }
    };

    this.peerConnection.onconnectionstatechange = () => {
      if (['disconnected', 'failed', 'closed'].includes(this.peerConnection.connectionState)) {
        this.disconnect();
      }
    };
  }

  handleIceCandidate(candidate) {
    const candidateData = {
      type: 'candidate',
      candidate: candidate.toJSON(),
    };

    if (this.webSocket?.readyState === WebSocket.OPEN) {
      this.webSocket.send(JSON.stringify(candidateData));
    } else {
      this.pendingCandidates.push(candidateData);
    }
  }

  sendOffer() {
    if (this.peerConnection.localDescription) {
      this.webSocket.send(JSON.stringify({
        type: 'offer',
        sdp: this.peerConnection.localDescription.sdp,
      }));
    }
  }

  sendPendingCandidates() {
    while (this.pendingCandidates.length > 0) {
      const candidate = this.pendingCandidates.shift();
      this.webSocket.send(JSON.stringify(candidate));
    }
  }

  processMetrics(data) {
    const metrics = {
      timestamp: Date.now(),
      data: new Uint8Array(data),
    };
    this.onMetricsCallback(metrics);
  }

  updateState(state) {
    this.connectionState = state;
    if (this.onStateChangeCallback) {
      this.onStateChangeCallback(state);
    }
  }

  disconnect() {
    if (this.dataChannel) {
      this.dataChannel.close();
      this.dataChannel = null;
    }

    if (this.peerConnection) {
      this.peerConnection.close();
      this.peerConnection = null;
    }

    if (this.webSocket) {
      this.webSocket.close();
      this.webSocket = null;
    }

    this.pendingCandidates = [];
    this.updateState('disconnected');
  }

  onMetrics(callback) {
    this.onMetricsCallback = callback;
  }

  onTestComplete(callback) {
    this.onTestCompleteCallback = callback;
  }

  onBirateUpdate(callback) { 
    this.onBitrateUpdateCallback = callback;
  }

  onStateChange(callback) {
    this.onStateChangeCallback = callback;
  }
}
