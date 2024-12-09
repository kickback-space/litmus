class MetricsManager {
  constructor() {
    this.metrics = {
      jitter: { current: 0 },
      packetLoss: { current: 0 }
    };

    this.receivedPacketsWindow = [];
    this.lastPacketTimestamp = null;
    this.lastInterarrival = 0;
    this.totalPackets = 0;
    this.lastReport = null;
    this.bytesSinceLastReport = 0;
    this.onMetricsUpdateCallback = null;
    this.onMetricsReportCallback = null;
  }

  processPacketData(metricsData) {
    const { data, timestamp } = metricsData;
    const sequence = new DataView(data.buffer).getUint32(0);
    const updateInterval = 200; // ms

    this.bytesSinceLastReport += data.byteLength;

    this.updatePacketLoss(sequence, timestamp);
    this.updateJitter(timestamp);

    if (this.onMetricsUpdateCallback) {
      this.onMetricsUpdateCallback(this.metrics);
    }

    if (!this.lastReport || timestamp - this.lastReport >= updateInterval) {
      const elapsedMs = this.lastReport ? (timestamp - this.lastReport) : updateInterval;
      const bitsReceived = this.bytesSinceLastReport * 8;
      const actualThroughput = (bitsReceived * 1000) / elapsedMs; // bits/second

      console.log("actual throughput")
      console.log(actualThroughput)
      if (this.onMetricsReportCallback) {
        this.onMetricsReportCallback({
          type: "metrics_report",
          loss_rate: this.metrics.packetLoss.current,
          jitter: this.metrics.jitter.current,
          sequence,
          actual_throughput: actualThroughput
        });
      }
      this.lastReport = timestamp;
      this.bytesSinceLastReport = 0;
    }
  }

  updateJitter(currentTimestamp) {
    if (this.lastPacketTimestamp !== null) {
      const arrivalDelta = currentTimestamp - this.lastPacketTimestamp;
      const jitterDelta = Math.abs(arrivalDelta - this.lastInterarrival);

      // RFC 3550 jitter formula with exponential moving average
      this.metrics.jitter.current += 
        (jitterDelta - this.metrics.jitter.current) / 16;
    }

    this.lastInterarrival = currentTimestamp - (this.lastPacketTimestamp || currentTimestamp);
    this.lastPacketTimestamp = currentTimestamp;
  }

  updatePacketLoss(sequence, timestamp) {
    this.receivedPacketsWindow.push({ sequence, timestamp });

    const windowStart = timestamp - 1000;
    this.receivedPacketsWindow = this.receivedPacketsWindow.filter(packet => packet.timestamp > windowStart);

    const uniqueSequences = [...new Set(this.receivedPacketsWindow.map(packet => packet.sequence))];

    if (uniqueSequences.length === 0) {
      this.metrics.packetLoss.current = 0;
      return;
    }

    const highestSequence = Math.max(...uniqueSequences);
    const lowestSequence = Math.min(...uniqueSequences);
    const expectedPackets = highestSequence - lowestSequence + 1;
    const receivedPackets = uniqueSequences.length;

    this.metrics.packetLoss.current = Math.max(0, (expectedPackets - receivedPackets) / expectedPackets);
  }

  reset() {
    this.metrics = {
      jitter: { current: 0 },
      packetLoss: { current: 0 }
    };

    this.receivedPacketsWindow = [];
    this.lastPacketTimestamp = null;
    this.lastInterarrival = 0;
    this.totalPackets = 0;
    this.lastReport = null;
    this.bytesSinceLastReport = 0;
  }

  onMetricsUpdate(callback) {
    this.onMetricsUpdateCallback = callback;
  }

  onMetricsReport(callback) {
    this.onMetricsReportCallback = callback;
  }

  async detectNetworkCapabilities() {
    if (!("connection" in navigator)) {
      console.warn("Network Information API not supported in this browser.");
    }
  }
}
