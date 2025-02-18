package rtpcc

import (
	"encoding/binary"
	"math/rand"
	"time"

	"github.com/pion/rtp"
)

type RTPConfig struct {
	// header
	Version          uint8
	Padding          bool
	Extension        bool
	Marker           bool
	PayloadType      uint8
	SequenceNumber   uint16
	Timestamp        uint32
	SSRC             uint32
	CSRC             []uint32
	ExtensionProfile uint16
	// Extensions       []Extension
}

func generateSSRC() uint32 {
	buf := make([]byte, 4)
	rand.Read(buf)
	return binary.BigEndian.Uint32(buf)
}

func (c *RTPConfig) UpdateTimestamp() {
	samplesPerPacket := uint32(48000 * 20 / 1000)
	c.Timestamp += samplesPerPacket
}

func NewRTPConfig() *RTPConfig {

	initialTimestamp := uint32(time.Now().UnixNano() / 1000000)

	return &RTPConfig{
		Version:          2,
		Padding:          false,
		Extension:        false,
		Marker:           false,
		PayloadType:      96,
		SequenceNumber:   0,
		Timestamp:        initialTimestamp,
		SSRC:             generateSSRC(),
		CSRC:             make([]uint32, 0),
		ExtensionProfile: 0xBEDE,
	}
}

func CreateRTPPacket(config *RTPConfig, payload []byte, message []byte) (*rtp.Packet, error) {
	packet := &rtp.Packet{
		Header: rtp.Header{
			Version:          config.Version,
			Padding:          config.Padding,
			Extension:        config.Extension,
			Marker:           config.Marker,
			PayloadType:      config.PayloadType,
			SequenceNumber:   config.SequenceNumber,
			Timestamp:        config.Timestamp,
			SSRC:             config.SSRC,
			CSRC:             config.CSRC,
			ExtensionProfile: config.ExtensionProfile,
		},
		Payload: payload,
	}

	// 存在扩展头数据时添加扩展头
	if len(message) > 0 {
		packet.Header.Extension = true
		packet.SetExtension(1, message)
	}

	return packet, nil
}

func ExtractData(packet *rtp.Packet) ([]byte, []byte, error) {
	IDs := packet.Header.GetExtensionIDs()
	var extensionPayload []byte

	if len(IDs) > 0 {
		extensionPayload = packet.GetExtension(IDs[0])
	}

	return packet.Payload, extensionPayload, nil
}
