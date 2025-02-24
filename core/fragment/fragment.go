package fragment

import (
	rtpcc "cc/core/rtp"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	"github.com/pion/rtp"
)

const (
	// 扩展头ID
	FragmentHeaderID  = 1 // 分片信息头的ID
	FragmentPayloadID = 2 // 分片数据的ID

	// 单个分片的最大数据长度
	MaxFragmentSize = 251 // 考虑RTP包的总大小限制
)

// 分片信息
type FragmentInfo struct {
	FragmentID     uint16 // 当前分片ID
	TotalFragments uint16 // 总分片数
}

// 分片缓存结构
type FragmentBuffer struct {
	mu             sync.Mutex
	Fragments      map[uint16][]byte
	TotalFragments uint16
	Complete       bool
}

// 创建新的分片缓存
func NewFragmentBuffer() *FragmentBuffer {
	return &FragmentBuffer{
		Fragments: make(map[uint16][]byte),
	}
}

// RTP分片处理器
type RTPFragmenter struct {
	config         *rtpcc.RTPConfig
	fragmentBuffer *FragmentBuffer
}

// 分片处理器接口
type Fragmenter interface {
	Fragment(message []byte) ([]*rtp.Packet, error)
	Process(packet *rtp.Packet) ([]byte, bool, error)
}

// 创建新的RTP分片处理器
func NewRTPFragmenter(config *rtpcc.RTPConfig) *RTPFragmenter {
	return &RTPFragmenter{
		config:         config,
		fragmentBuffer: NewFragmentBuffer(),
	}
}

// 创建分片的RTP包
func (rf *RTPFragmenter) Fragment(message []byte) ([]*rtp.Packet, error) {
	if len(message) == 0 {
		return nil, errors.New("empty message")
	}

	// 如果消息长度小于最大分片大小，直接返回单个包
	// debug
	fmt.Println("len(message):", len(message))
	if len(message) <= MaxFragmentSize {
		packet, err := rtpcc.CreateRTPPacket(rf.config, nil, message)
		if err != nil {
			return nil, err
		}
		rf.config.SequenceNumber++
		rf.config.UpdateTimestamp()
		//debug
		fmt.Println("Packet num: ", 1)
		return []*rtp.Packet{packet}, nil
	}

	// 计算需要的分片数
	totalFragments := (len(message) + MaxFragmentSize - 1) / MaxFragmentSize
	if totalFragments > 65535 {
		return nil, errors.New("message too large, exceeds maximum fragments")
	}

	var packets []*rtp.Packet

	for i := 0; i < totalFragments; i++ {
		// 计算当前分片的数据范围
		start := i * MaxFragmentSize
		end := start + MaxFragmentSize
		if end > len(message) {
			end = len(message)
		}

		// 构造分片信息头
		fragmentInfo := FragmentInfo{
			FragmentID:     uint16(i),
			TotalFragments: uint16(totalFragments),
		}
		fragmentInfoBytes := make([]byte, 4)
		binary.BigEndian.PutUint16(fragmentInfoBytes[0:2], fragmentInfo.FragmentID)
		binary.BigEndian.PutUint16(fragmentInfoBytes[2:4], fragmentInfo.TotalFragments)

		// 创建RTP包
		packet := &rtp.Packet{
			Header: rtp.Header{
				Version:   rf.config.Version,
				Padding:   rf.config.Padding,
				Extension: true,
				Marker:    i == totalFragments-1,
				// Marker:           false,
				PayloadType:      rf.config.PayloadType,
				SequenceNumber:   rf.config.SequenceNumber,
				Timestamp:        rf.config.Timestamp,
				SSRC:             rf.config.SSRC,
				CSRC:             rf.config.CSRC,
				ExtensionProfile: rf.config.ExtensionProfile,
			},
		}

		// 设置分片信息扩展头
		packet.SetExtension(FragmentHeaderID, fragmentInfoBytes)

		// 设置分片数据扩展头
		packet.SetExtension(FragmentPayloadID, message[start:end])

		//debug
		fmt.Println("debug:")
		fmt.Println("message[start:end]:", string(message[start:end]))

		packets = append(packets, packet)
		rf.config.SequenceNumber++
		rf.config.UpdateTimestamp()
	}

	//debug
	fmt.Println("Total fragments: ", totalFragments)
	fmt.Println("Packet num: ", len(packets))

	return packets, nil
}

// 处理分片包并尝试重组
func (fb *FragmentBuffer) ProcessPacket(packet *rtp.Packet) ([]byte, bool, error) {
	fb.mu.Lock()
	defer fb.mu.Unlock()

	fragmentInfoBytes := packet.GetExtension(FragmentHeaderID)
	if len(fragmentInfoBytes) != 4 {
		//debug
		fragmentID := binary.BigEndian.Uint16(fragmentInfoBytes[0:2])
		totalFragments := binary.BigEndian.Uint16(fragmentInfoBytes[2:4])
		fmt.Println("fragmentID:", fragmentID, "totalFragments:", totalFragments)
		return nil, false, errors.New("invalid fragment info")
	}

	fragmentID := binary.BigEndian.Uint16(fragmentInfoBytes[0:2])
	totalFragments := binary.BigEndian.Uint16(fragmentInfoBytes[2:4])
	//debug
	fmt.Println("fragmentID:", fragmentID, "totalFragments:", totalFragments)

	if fb.TotalFragments == 0 {
		fb.TotalFragments = totalFragments
	} else if fb.TotalFragments != totalFragments {
		return nil, false, errors.New("inconsistent total fragments")
	}

	fragmentData := packet.GetExtension(FragmentPayloadID)
	if len(fragmentData) == 0 {
		return nil, false, errors.New("empty fragment data")
	}

	fb.Fragments[fragmentID] = make([]byte, len(fragmentData))
	copy(fb.Fragments[fragmentID], fragmentData)
	// fb.Fragments[fragmentID] = fragmentData
	//debug
	fmt.Println("id: ", fragmentID, "fragmentData:", string(fragmentData))

	if len(fb.Fragments) == int(fb.TotalFragments) {
		// message := make([]byte, 0, MaxFragmentSize*int(fb.TotalFragments))
		message := make([]byte, 0)
		for i := uint16(0); i < fb.TotalFragments; i++ {
			if data, ok := fb.Fragments[i]; ok {
				message = append(message, data...)
				//debug
				fmt.Println("~i:", i, "data:", string(data))
			} else {
				return nil, false, errors.New("missing fragment")
			}
		}
		fb.Complete = true
		// 清理缓存
		fb.Fragments = make(map[uint16][]byte)
		fb.TotalFragments = 0
		return message, true, nil
	}

	return nil, false, nil
}

// 处理接收到的RTP包
func (rf *RTPFragmenter) Process(packet *rtp.Packet) ([]byte, bool, error) {
	// 检查是否是分片包
	if !packet.Header.Extension {
		// 非分片包，直接返回payload
		payload, extension, err := rtpcc.ExtractData(packet)
		if err != nil {
			return nil, true, err
		}
		if len(extension) > 0 {
			return extension, true, nil
		}
		return payload, true, nil
	}

	return rf.fragmentBuffer.ProcessPacket(packet)
}
