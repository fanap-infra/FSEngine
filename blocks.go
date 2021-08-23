package fsEngine

import (
	"encoding/binary"
	"fmt"
)

func (fse *FSEngine) NoSpace() uint32 {
	tryCounter := 0
	for {
		fileIndex, err := fse.header.FindOldestFile()
		if err != nil {
			fse.log.Errorv("can not find oldest file", "err", err.Error())
			return 0
		}
		blockIndex := fileIndex.FirstBlock
		n, err := fse.RemoveVirtualFile(fileIndex.Id)
		if err != nil {
			fse.log.Errorv("can not remove virtual file", "id", fileIndex.Id,
				"err", err.Error())
			return 0
		}
		fse.eventsHandler.VirtualFileDeleted(fileIndex.Id, "file deleted due to space requirements")
		if n == 0 {

			if tryCounter > 1 {
				fse.log.Warnv("can not make any block empty after multiple try", "Id", fileIndex.Id,
					"fileIndex.FirstBlock", fileIndex.FirstBlock, "tryCounter", tryCounter)
				return blockIndex
			}
			fse.log.Warnv("can not make any block empty, try again", "Id", fileIndex.Id,
				"fileIndex.FirstBlock", fileIndex.FirstBlock)
			tryCounter++
			continue
		}
		return blockIndex
	}
}

// BlockStructure
//	+===============+===============+===============+=======+
//	|    				 	  BLOCK 				   		|
//	+--------+------+--------+------+--------+------+-------+
//	|  blockID  |   fileID 	 |	 prevBlock	 | valid Size 	|
//	+--------+------+--------+------+--------+------+-------+
//  |  4 byte   |   4 byte   |    4 byte     |   4 byte     |   16 Byte
//	+--------+------+--------+------+--------+------+-------+
// Warning It is not thread safe
func (fse *FSEngine) prepareBlock(data []byte, fileID uint32, previousBlock uint32, blockID uint32) ([]byte, error) {
	dataTmp := make([]byte, 0)

	headerTmp := make([]byte, 4)
	binary.BigEndian.PutUint32(headerTmp, blockID)
	dataTmp = append(dataTmp, headerTmp...)
	binary.BigEndian.PutUint32(headerTmp, fileID)
	dataTmp = append(dataTmp, headerTmp...)
	binary.BigEndian.PutUint32(headerTmp, previousBlock)
	dataTmp = append(dataTmp, headerTmp...)
	binary.BigEndian.PutUint32(headerTmp, uint32(len(data)))
	dataTmp = append(dataTmp, headerTmp...)
	dataTmp = append(dataTmp, data...)

	return dataTmp, nil
}

func (fse *FSEngine) parseBlock(data []byte, blockID uint32, fileID uint32) ([]byte, error) {
	pBlockID := binary.BigEndian.Uint32(data[0:4])
	pFileID := binary.BigEndian.Uint32(data[4:8])
	dataSize := binary.BigEndian.Uint32(data[12:16])
	if dataSize > fse.blockSize-16 {
		return nil, fmt.Errorf("blockd data size is too large, dataSize: %v", dataSize)
	}

	if pBlockID != blockID {
		return nil, fmt.Errorf("blockd id is wrong, pBlockID: %v, blockID: %v ", pBlockID, blockID)
	}
	if pFileID != fileID {
		return nil, fmt.Errorf("file id is wrong, pFileID: %v, fileID: %v ", pBlockID, blockID)
	}
	return data[16 : dataSize+16], nil
}

func (fse *FSEngine) BAMUpdated(fileID uint32, bam []byte) error {
	// ToDo: because of file index,we use mutex
	fse.crudMutex.Lock()
	defer fse.crudMutex.Unlock()
	return fse.header.UpdateBAM(fileID, bam)
}

func (fse *FSEngine) UpdateFileIndexes(fileID uint32, firstBlock uint32, lastBlock uint32,
	fileSize uint32, bam []byte, info []byte) error {
	fse.crudMutex.Lock()
	defer fse.crudMutex.Unlock()
	return fse.header.UpdateFileIndexes(fileID, firstBlock, lastBlock, fileSize, bam, info)
}

func (fse *FSEngine) UpdateFileOptionalData(fileId uint32, info []byte) error {
	fse.crudMutex.Lock()
	defer fse.crudMutex.Unlock()
	err := fse.header.UpdateFSHeader()
	if err != nil {
		fse.log.Warnv("Can not updateHeader", "err", err.Error())
		return err
	}
	return fse.header.UpdateFileOptionalData(fileId, info)
}

//func (fse *FSEngine) GetFileOptionalData(fileId uint32) ([]byte, error) {
//	fse.crudMutex.Lock()
//	defer fse.crudMutex.Unlock()
//	return fse.header.UpdateFileOptionalData(fileId, info)
//}
