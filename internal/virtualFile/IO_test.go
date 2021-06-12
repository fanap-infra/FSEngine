package virtualFile

import (
	"bytes"
	"github.com/fanap-infra/FSEngine/internal/blockAllocationMap"
	"github.com/fanap-infra/log"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"testing"
)

const blockSizeTest  =  5120
const maxNumberOfBlocks  =  3
const vfID  = 1

var byte2D [][]byte

type FSMock struct {
	vBuf []byte
	seekPointer int
	vBufBlocks [][]byte
	openFiles          map[uint32]*VirtualFile
}

func (fsMock *FSMock) Write(data []byte, fileID uint32) (int, error) {
	if bytes.Compare(data, byte2D[len(byte2D)-1]) == 0  {
		log.Errorv("fsMock data is not equal", "len(data)", len(data),
			"len(byte2D)", byte2D[len(byte2D)-1])
	}
	fsMock.vBuf = append(fsMock.vBuf, data...)
	counter := 0
	for {
		if (len(data) - counter) <  blockSizeTest {
			//log.Infov("FSMock Write smaller than blockSizeTest",
			//	"blockID", len(fsMock.vBufBlocks), "data size", len(data), "counter", counter)
			err := fsMock.openFiles[fileID].AddBlockID(uint32(len(fsMock.vBufBlocks)))
			if err != nil {
				return 0, err
			}
			fsMock.vBufBlocks = append(fsMock.vBufBlocks, data[counter:])
			counter = len(data)
		} else {
			//log.Infov("FSMock Write greater than blockSizeTest",
			//	"blockID", len(fsMock.vBufBlocks), "data size", len(data), "counter", counter)
			err := fsMock.openFiles[fileID].AddBlockID(uint32(len(fsMock.vBufBlocks)))
			if err != nil {
				return 0, err
			}
			fsMock.vBufBlocks = append(fsMock.vBufBlocks, data[counter:blockSizeTest])
			counter = counter + blockSizeTest
		}
		if counter >= len(data) {
			if counter != len(data) {
				log.Warnv("counter greater than data", "counter",counter,"len(data)",len(data))
			}
			return len(data), nil
		}
	}
}

func (fsMock *FSMock)  WriteAt(data []byte, off int64, fileID uint32) (int, error) {
	fsMock.vBuf = append(fsMock.vBuf, data...)
	return len(data), nil
}

func (fsMock *FSMock)  Read(data []byte, fileID uint32) (int, error){
	data = fsMock.vBuf[fsMock.seekPointer:fsMock.seekPointer+len(data)]
	fsMock.seekPointer = fsMock.seekPointer+len(data)
	return len(data) , nil
}

func (fsMock *FSMock) ReadAt(data []byte, off int64, fileID uint32) (int, error){
	return len(data), nil
}

func (fsMock *FSMock) ReadBlock(blockIndex uint32) ([]byte, error){
	return fsMock.vBufBlocks[blockIndex], nil
}

func (fsMock *FSMock) Closed(fileID uint32) error{
	return  nil
}

func (fsMock *FSMock) NoSpace() uint32 {
	return 0
}

func NewVBufMock() *FSMock {
	return &FSMock{seekPointer: 0, openFiles: make(map[uint32]*VirtualFile)}
}

func TestIO_WR(t *testing.T) {
	fsMock := NewVBufMock()
	blm  := blockAllocationMap.New(log.GetScope("test"), fsMock, maxNumberOfBlocks)
	vf := NewVirtualFile("test", vfID, blockSizeTest, fsMock, blm,
		int(blockSizeTest)*2, log.GetScope("test2"))
	fsMock.openFiles[vfID] = vf


	size := 0
	VFSize := int(1.5*blockSizeTest)
	MaxByteArraySize := int(blockSizeTest*0.5)
	for {
		token := make([]byte, uint32( rand.Intn(MaxByteArraySize))+1)
		m, err := rand.Read(token)
		assert.Equal(t, nil, err)
		byte2D  = append(byte2D, token)
		assert.Equal(t, m, len(token))
		size = size + m
		n, err := vf.Write(token)
		assert.Equal(t, nil, err)
		assert.Equal(t, m, n)

		if size > VFSize {
			break
		}
	}

	err := vf.Close()
	assert.Equal(t, nil, err)
	//log.Infov("writting finished ",
	//	"number of blocks", len(vf.blockAllocationMap.ToArray()), "size", size,
	//	"len(fsMock.vBuf)", len(fsMock.vBuf))
	assert.Equal(t, size, len(fsMock.vBuf))
	counter := 0
	var vBlocks []byte
	for _, v := range fsMock.vBufBlocks {
		vBlocks = append(vBlocks, v...)
	}
	for _, v := range byte2D {
		assert.Equal(t, v, vBlocks[counter:counter+len(v)])
		counter = counter + len(v)
	}

	for _, v := range byte2D {
		buf := make([]byte, len(v))
		for j, _ := range buf {
			buf[j] = 0
		}
		//log.Infov("read buf ", "len(buf)", len(buf), "i",i, "" +
		//	"fsMock.seekPointer", fsMock.seekPointer, "vf.seekPointer", vf.seekPointer,
		//	"vf.bufStart",vf.bufStart, "vf.bufEnd",vf.bufEnd)
		//if i+1 == len(byte2D) {
		//	log.Warn("last packet")
		//}
		_, err := vf.Read(buf)
		assert.Equal(t, nil, err)
		if err != nil {
			log.Warn("Test")
			break
		}
		//assert.Equal(t, v[0], buf[0])
		assert.Equal(t, 0,bytes.Compare(v, buf) )

	}


	assert.Equal(t, 0,bytes.Compare(vBlocks, vf.bufRX) )


}
