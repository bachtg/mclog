package log

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
)

// Log represents a log file with file path, file size and mutex for concurrent access
type Log struct {
	file *os.File
	path string
	size uint64
	mu   sync.Mutex
}

func NewLog(path string) (*Log, error) {
	// O_CREATE: Create a new file if it does not exist
	// O_RDWR: Open the file for read and write
	// 0644: permission read/write
	file, err := os.OpenFile(
		path,
		os.O_CREATE|os.O_RDWR,
		0644,
	)
	if err != nil {
		fmt.Println("Failed to open log file:", err)
		return nil, err
	}

	size, err := getFileSize(file)
	if err != nil {
		return nil, err
	}

	return &Log{
		file: file,
		path: path,
		size: size,
	}, nil
}

func getFileSize(file *os.File) (uint64, error) {
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return uint64(info.Size()), nil
}

// Append writes []byte into log file and returns written bytes
// format: [len1][data1][len2][data2]...
func (l *Log) Append(data []byte) (uint64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// only allow data with lenth size of uint32
	if len(data) > int(^uint32(0)) {
		return 0, os.ErrInvalid
	}

	offset, err := l.file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}

	// write data length
	length := uint32(len(data))
	if err := binary.Write(l.file, binary.BigEndian, length); err != nil {
		return 0, err
	}

	// write data
	_, err = l.file.Write(data)
	if err != nil {
		return 0, err
	}

	return uint64(offset), nil
}

func (l *Log) Read(offset uint64) ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.file.Seek(int64(offset), io.SeekStart); err != nil {
		return nil, err
	}

	var length uint32
	if err := binary.Read(l.file, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	data := make([]byte, length)
	if _, err := l.file.Read(data); err != nil {
		return nil, err
	}

	return data, nil
}

func (l *Log) Dump() ([]byte, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := l.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return io.ReadAll(l.file)
}

func (l *Log) Load(data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if err := l.file.Truncate(0); err != nil {
		return err
	}

	if _, err := l.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if _, err := l.file.Write(data); err != nil {
		return err
	}

	l.size = uint64(len(data))
	return nil
}
