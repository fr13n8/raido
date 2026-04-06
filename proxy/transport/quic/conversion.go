package quic

import (
	"strconv"
	"time"

	"github.com/quic-go/quic-go/qlog"
)

// Helper to convert logging.ByteCount(alias for int64) to float64 used in prometheus
func byteCountToPromCount(count int64) float64 {
	return float64(count)
}

// Helper to convert Duration to float64 used in prometheus
func durationToPromGauge(duration time.Duration) float64 {
	return float64(duration.Milliseconds())
}

// Helper to convert https://pkg.go.dev/github.com/quic-go/quic-go@vv0.47.0/logging#PacketType into string
func packetTypeString(pt qlog.PacketType) string {
	switch pt {
	case qlog.PacketTypeInitial:
		return "initial"
	case qlog.PacketTypeHandshake:
		return "handshake"
	case qlog.PacketTypeRetry:
		return "retry"
	case qlog.PacketType0RTT:
		return "0_rtt"
	case qlog.PacketTypeVersionNegotiation:
		return "version_negotiation"
	case qlog.PacketType1RTT:
		return "1_rtt"
	case qlog.PacketTypeStatelessReset:
		return "stateless_reset"
	default:
		return "unknown_packet_type"
	}
}

// Helper to convert https://pkg.go.dev/github.com/quic-go/quic-go@vv0.47.0/logging#PacketDropReason into string
func packetDropReasonString(reason qlog.PacketDropReason) string {
	switch reason {
	case qlog.PacketDropKeyUnavailable:
		return "key_unavailable"
	case qlog.PacketDropUnknownConnectionID:
		return "unknown_conn_id"
	case qlog.PacketDropHeaderParseError:
		return "header_parse_err"
	case qlog.PacketDropPayloadDecryptError:
		return "payload_decrypt_err"
	case qlog.PacketDropProtocolViolation:
		return "protocol_violation"
	case qlog.PacketDropDOSPrevention:
		return "dos_prevention"
	case qlog.PacketDropUnsupportedVersion:
		return "unsupported_version"
	case qlog.PacketDropUnexpectedPacket:
		return "unexpected_packet"
	case qlog.PacketDropUnexpectedSourceConnectionID:
		return "unexpected_src_conn_id"
	case qlog.PacketDropUnexpectedVersion:
		return "unexpected_version"
	case qlog.PacketDropDuplicate:
		return "duplicate"
	default:
		return "unknown_reason"
	}
}

// Helper to convert https://pkg.go.dev/github.com/quic-go/quic-go@v0.47.0/logging#PacketLossReason into string
func packetLossReasonString(reason qlog.PacketLossReason) string {
	switch reason {
	case qlog.PacketLossReorderingThreshold:
		return "reordering"
	case qlog.PacketLossTimeThreshold:
		return "timeout"
	default:
		return "unknown_loss_reason"
	}
}

func uint8ToString(input uint8) string {
	return strconv.FormatUint(uint64(input), 10)
}
