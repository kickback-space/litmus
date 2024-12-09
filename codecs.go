// - VideoCodecCapability
// - VideoCodecRanking
// - VideoCodecProfile
// - Shared enums and constants

package litmus

type VideoCodec string

const (
    CodecH264 VideoCodec = "H264"
	CodecH265 VideoCodec = "H265"
    CodecVP8  VideoCodec = "VP8"
    CodecVP9  VideoCodec = "VP9"
    CodecAV1  VideoCodec = "AV1"
)

// VideoCodecCapability represents a codec's capabilities for a specific profile
type VideoCodecCapability struct {
    Codec        VideoCodec
    MimeType     string   // e.g., "video/H264"
    Profile      string   // e.g., "constrained-baseline"
    Level        string   // e.g., "3.1"
    SupportsSend bool
    SupportsRecv bool
}

// VideoCodecQuality represents quality metrics for a codec at a specific profile
type VideoCodecQuality struct {
    Profile            *VideoProfile // Reference to the profile being tested
    SmoothPlayback     bool          // Can play smoothly at this quality
    PowerEfficient     bool          // Power efficient at this quality
	HardwareAccelerated bool         // if both Smooth and Power are true it likely means there's a chip
}

// VideoCodecRanking represents a codec's ranked capabilities for sending and receiving
type VideoCodecRanking struct {
    Capability     VideoCodecCapability
    SendRank       int                    // Higher is better for sending
    ReceiveRank    int                    // Higher is better for receiving
    QualityResults []VideoCodecQuality    // Results for each tested profile
    Parameters     map[string]interface{} // Additional codec-specific parameters
}

// CodecPreference indicates whether the ranking is optimized for sending or receiving
type CodecPreference int

const (
    PreferSend CodecPreference = iota
    PreferReceive
)

// VideoCodecSelection represents the final codec selection for a session
type VideoCodecSelection struct {
    Codec         VideoCodec
    Profile       *VideoProfile  // Selected quality profile
    Preference    CodecPreference
    Parameters    map[string]interface{}
}

// String returns the string representation of a VideoCodec
func (c VideoCodec) String() string {
    return string(c)
}

// MimeType returns the MIME type for a given codec
func (c VideoCodec) MimeType() string {
    switch c {
    case CodecH264:
        return "video/H264"
    case CodecVP8:
        return "video/VP8"
    case CodecVP9:
        return "video/VP9"
    case CodecAV1:
        return "video/AV1"
    default:
        return "video/unknown"
    }
}

// ParseMimeType converts a MIME type string to a VideoCodec
func ParseMimeType(mimeType string) VideoCodec {
    switch mimeType {
    case "video/H264":
        return CodecH264
	case "video/h265":
		return CodecH265
    case "video/VP8":
        return CodecVP8
    case "video/VP9":
        return CodecVP9
    case "video/AV1":
        return CodecAV1
    default:
        return ""
    }
}

// ValidateProfile checks if a codec supports a given profile configuration
func (c VideoCodec) ValidateProfile(p *VideoProfile) bool {
    if p == nil {
        return false
    }
    
    // Check for reasonable frame rate
    if p.FrameRate <= 0 || p.FrameRate > 31 {
        return false
    }

    // Check for reasonable dimensions
    if p.Width <= 0 || p.Height <= 0 {
        return false
    }

    // Check for reasonable bitrate
    if p.Bitrate <= 0 {
        return false
    }

    return true
}