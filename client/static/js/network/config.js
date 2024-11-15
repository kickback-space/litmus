// NetworkConfig.js
const NetworkConfig = {
  rtcConfig: {
    iceServers: [
      {
        urls: "stun:stun.l.google.com:19302",
      },
    ],
  },
  
  dataChannelConfig: {
    ordered: false,
    maxRetransmits: 0,
  }
};

Object.freeze(NetworkConfig);