package client

import (
	"bufio"
	rtpcc "cc/core/rtp"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/pion/rtp"
)

type Client struct {
	sendConn   *net.UDPConn
	recvConn   *net.UDPConn
	config     *rtpcc.RTPConfig
	buffer     []byte
	remoteAddr *net.UDPAddr
}

func GetClientConfigSSRC(cli *Client) (uint32, error) {
	rtpSSRC := cli.config.SSRC
	return rtpSSRC, nil
}

func NewClient(localPort int, remoteAddr string, remotePort int) (*Client, error) {
	// 创建接收连接
	recvAddr := &net.UDPAddr{
		IP:   net.ParseIP("0.0.0.0"),
		Port: localPort,
	}
	recvConn, err := net.ListenUDP("udp", recvAddr)
	if err != nil {
		return nil, fmt.Errorf("error creating receive connection: %v", err)
	}

	// 创建发送连接
	remoteUDPAddr := &net.UDPAddr{
		IP:   net.ParseIP(remoteAddr),
		Port: remotePort,
	}
	sendConn, err := net.DialUDP("udp", nil, remoteUDPAddr)
	if err != nil {
		recvConn.Close()
		return nil, fmt.Errorf("error creating send connection: %v", err)
	}

	return &Client{
		sendConn:   sendConn,
		recvConn:   recvConn,
		config:     rtpcc.NewRTPConfig(),
		buffer:     make([]byte, 1500),
		remoteAddr: remoteUDPAddr,
	}, nil
}

func (c *Client) Close() {
	if c.sendConn != nil {
		c.sendConn.Close()
	}
	if c.recvConn != nil {
		c.recvConn.Close()
	}
}

func (c *Client) SendMessage(message string) error {
	packet, err := rtpcc.CreateRTPPacket(c.config, nil, []byte(message))
	if err != nil {
		return fmt.Errorf("error creating RTP packet: %v", err)
	}

	rawPacket, err := packet.Marshal()
	if err != nil {
		return fmt.Errorf("error marshaling packet: %v", err)
	}

	_, err = c.sendConn.Write(rawPacket)
	if err != nil {
		return fmt.Errorf("error sending packet: %v", err)
	}

	c.config.SequenceNumber++
	c.config.UpdateTimestamp()
	return nil
}

func (c *Client) ReceiveMessage() (string, *rtp.Header, error) {
	n, _, err := c.recvConn.ReadFromUDP(c.buffer)
	if err != nil {
		return "", nil, fmt.Errorf("error reading from UDP: %v", err)
	}

	packet := &rtp.Packet{}
	if err := packet.Unmarshal(c.buffer[:n]); err != nil {
		return "", nil, fmt.Errorf("error unmarshaling packet: %v", err)
	}

	payload, extension, err := rtpcc.ExtractData(packet)
	if err != nil {
		return "", nil, fmt.Errorf("error extracting data: %v", err)
	}

	var message string
	if len(extension) > 0 {
		message = string(extension)
	} else if len(payload) > 0 {
		message = string(payload)
	}

	return message, &packet.Header, nil
}

func (c *Client) StartSending(wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Println("Type your message and press Enter to send (type 'quit' to exit):")

	for scanner.Scan() {
		message := scanner.Text()
		if message == "quit" {
			return
		}

		if err := c.SendMessage(message); err != nil {
			fmt.Printf("Error sending message: %v\n", err)
			continue
		}
	}
}

func (c *Client) StartReceiving(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		message, header, err := c.ReceiveMessage()
		if err != nil {
			// 检查是否是因为连接关闭导致的错误
			if opErr, ok := err.(*net.OpError); ok && opErr.Err.Error() == "use of closed network connection" {
				return
			}
			fmt.Printf("Error receiving message: %v\n", err)
			continue
		}

		if message != "" {
			fmt.Printf("\nReceived from SSRC 0x%X (seq=%d): %s\n",
				header.SSRC,
				header.SequenceNumber,
				message)
		}
	}
}
