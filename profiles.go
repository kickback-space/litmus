package litmus

type VideoProfile struct {
	Name                 string
	Resolution           string
	Width                int
	Height               int
	FrameRate            int
	Codec                string
	Bitrate              int     // in kbps
	AcceptablePacketLoss float64 // acceptable packet loss (e.g., 0.01 for 1%)
	AcceptableJitter     float64 // acceptable jitter in milliseconds
	PacketSize           int     // calculated packet size in bytes
	PacketsPerSecond     int     // calculated packets per second
}

func CalculatePacketSize(bitrateKbps int, frameRate int) (int, int) {
	totalBitsPerSecond := bitrateKbps * 1000
	bitsPerFrame := totalBitsPerSecond / frameRate
	// Assume each frame is divided into packets of approximately 1200 bytes (typical MTU)
	maxPacketPayloadSize := 1200 * 8 // in bits
	packetsPerFrame := bitsPerFrame / maxPacketPayloadSize
	if packetsPerFrame == 0 {
		packetsPerFrame = 1
	}
	packetSize := bitsPerFrame / packetsPerFrame / 8 // in bytes
	packetsPerSecond := packetsPerFrame * frameRate
	return packetSize, packetsPerSecond
}

var VideoProfiles = []VideoProfile{
	{
		Name:                 "1152p30fps",
		Resolution:           "2048x1152",
		Width:                2048,
		Height:               1152,
		FrameRate:            30,
		Codec:                "H.264",
		Bitrate:              9000,  // in kbps
		AcceptablePacketLoss: 0.004, // 0.4%
		AcceptableJitter:     18.0,  // milliseconds
	},
	{
		Name:                 "1152p24fps",
		Resolution:           "2048x1152",
		Width:                2048,
		Height:               1152,
		FrameRate:            24,
		Codec:                "H.264",
		Bitrate:              7500,
		AcceptablePacketLoss: 0.005,
		AcceptableJitter:     22.0,
	},
	{
		Name:                 "1080p30fps",
		Resolution:           "1920x1080",
		Width:                1920,
		Height:               1080,
		FrameRate:            30,
		Codec:                "H.264",
		Bitrate:              8000,
		AcceptablePacketLoss: 0.005,
		AcceptableJitter:     20.0,
	},
	{
		Name:                 "1080p24fps",
		Resolution:           "1920x1080",
		Width:                1920,
		Height:               1080,
		FrameRate:            24,
		Codec:                "H.264",
		Bitrate:              6000,
		AcceptablePacketLoss: 0.007,
		AcceptableJitter:     25.0,
	},
	{
		Name:                 "960p30fps",
		Resolution:           "1440x960",
		Width:                1440,
		Height:               960,
		FrameRate:            30,
		Codec:                "H.264",
		Bitrate:              5000,
		AcceptablePacketLoss: 0.008,
		AcceptableJitter:     30.0,
	},
	{
		Name:                 "960p24fps",
		Resolution:           "1440x960",
		Width:                1440,
		Height:               960,
		FrameRate:            24,
		Codec:                "H.264",
		Bitrate:              4000,
		AcceptablePacketLoss: 0.01,
		AcceptableJitter:     35.0,
	},
	{
		Name:                 "720p30fps",
		Resolution:           "1280x720",
		Width:                1280,
		Height:               720,
		FrameRate:            30,
		Codec:                "H.264",
		Bitrate:              3000,
		AcceptablePacketLoss: 0.015,
		AcceptableJitter:     40.0,
	},
	{
		Name:                 "540p30fps",
		Resolution:           "960x540",
		Width:                960,
		Height:               540,
		FrameRate:            30,
		Codec:                "H.264",
		Bitrate:              2000,
		AcceptablePacketLoss: 0.02,
		AcceptableJitter:     50.0,
	},
	{
		Name:                 "540p24fps",
		Resolution:           "960x540",
		Width:                960,
		Height:               540,
		FrameRate:            24,
		Codec:                "H.264",
		Bitrate:              1800,
		AcceptablePacketLoss: 0.022,
		AcceptableJitter:     55.0,
	},
}

func initProfiles() {
	for i := range VideoProfiles {
		packetSize, packetsPerSecond := CalculatePacketSize(VideoProfiles[i].Bitrate, VideoProfiles[i].FrameRate)
		VideoProfiles[i].PacketSize = packetSize
		VideoProfiles[i].PacketsPerSecond = packetsPerSecond
	}
}