package fragment

import (
	"fmt"
	"testing"
)

func TestMap(t *testing.T) {
	fb := make(map[uint16][]byte)
	fb[uint16(0)] = []byte("AA")
	fb[uint16(1)] = []byte("BB")
	fb[uint16(2)] = []byte("CC")
	fb[uint16(3)] = []byte("DD")
	data := []byte("EE")
	fb[uint16(4)] = make([]byte, len(data))
	copy(fb[uint16(4)], data)

	message := make([]byte, 0)
	for i := uint16(0); i < 4; i++ {
		if data, ok := fb[i]; ok {
			message = append(message, data...)
			//debug
			fmt.Println("i:", i, "data:", string(data))
		} else {
			fmt.Println("missing fragment")
		}
	}
	fmt.Println(len(fb))
	fmt.Println(message)
}
