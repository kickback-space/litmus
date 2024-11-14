// NetworkConfig.js
const NetworkConfig = {
  // WebRTC configuration
  rtcConfig: {
    iceServers: [
      {
        urls: "stun:stun.l.google.com:19302",
      },
    ],
  },
  
  // Data channel configuration
  dataChannelConfig: {
    ordered: false,
    maxRetransmits: 0,
  }
};
// Prevent modifications to the configuration
Object.freeze(NetworkConfig);