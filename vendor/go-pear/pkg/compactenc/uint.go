package compactenc

import (
	"encoding/binary"
	"fmt"
)

// PreencodeUint calculates byte size required to encode a uint64.
func PreencodeUint(val uint64) int {
	if val <= 0xfc {
		return 1
	}
	if val <= 0xffff {
		return 3
	}
	if val <= 0xffffffff {
		return 5
	}
	return 9
}

// EncodeUint encodes a variable-length unsigned integer into state.
func EncodeUint(s *State, val uint64) error {
	if val <= 0xfc {
		if s.Start+1 > len(s.Buffer) {
			return ErrBufferTooSmall
		}
		s.Buffer[s.Start] = byte(val)
		s.Start += 1
		return nil
	}

	if val <= 0xffff {
		if s.Start+3 > len(s.Buffer) {
			return ErrBufferTooSmall
		}
		s.Buffer[s.Start] = 0xfd
		binary.LittleEndian.PutUint16(s.Buffer[s.Start+1:], uint16(val))
		s.Start += 3
		return nil
	}

	if val <= 0xffffffff {
		if s.Start+5 > len(s.Buffer) {
			return ErrBufferTooSmall
		}
		s.Buffer[s.Start] = 0xfe
		binary.LittleEndian.PutUint32(s.Buffer[s.Start+1:], uint32(val))
		s.Start += 5
		return nil
	}

	if s.Start+9 > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	s.Buffer[s.Start] = 0xff
	binary.LittleEndian.PutUint64(s.Buffer[s.Start+1:], val)
	s.Start += 9
	return nil
}

// DecodeUint decodes a variable-length unsigned integer from state.
func DecodeUint(s *State) (uint64, error) {
	if s.Start >= s.End {
		return 0, ErrUnexpectedEOF
	}

	tag := s.Buffer[s.Start]
	if tag <= 0xfc {
		s.Start += 1
		return uint64(tag), nil
	}

	switch tag {
	case 0xfd:
		if s.Start+3 > s.End {
			return 0, ErrUnexpectedEOF
		}
		val := binary.LittleEndian.Uint16(s.Buffer[s.Start+1:])
		s.Start += 3
		return uint64(val), nil

	case 0xfe:
		if s.Start+5 > s.End {
			return 0, ErrUnexpectedEOF
		}
		val := binary.LittleEndian.Uint32(s.Buffer[s.Start+1:])
		s.Start += 5
		return uint64(val), nil

	case 0xff:
		if s.Start+9 > s.End {
			return 0, ErrUnexpectedEOF
		}
		val := binary.LittleEndian.Uint64(s.Buffer[s.Start+1:])
		s.Start += 9
		return val, nil

	default:
		return 0, fmt.Errorf("%w: unrecognized tag 0x%x", ErrInvalidEncoding, tag)
	}
}

// PreencodeUint64 returns the fixed 8-byte size for a uint64.
func PreencodeUint64(_ uint64) int {
	return 8
}

// EncodeUint64 encodes a fixed 8-byte little-endian unsigned integer.
func EncodeUint64(s *State, val uint64) error {
	if s.Start+8 > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	binary.LittleEndian.PutUint64(s.Buffer[s.Start:], val)
	s.Start += 8
	return nil
}

// DecodeUint64 decodes a fixed 8-byte little-endian unsigned integer.
func DecodeUint64(s *State) (uint64, error) {
	if s.Start+8 > s.End {
		return 0, ErrUnexpectedEOF
	}
	val := binary.LittleEndian.Uint64(s.Buffer[s.Start:])
	s.Start += 8
	return val, nil
}

// PreencodeUint32Array calculates byte size required to encode a uint32 array.
func PreencodeUint32Array(arr []uint32) int {
	return PreencodeUint(uint64(len(arr))) + len(arr)*4
}

// EncodeUint32Array encodes a uint32 slice prefixed by compact uint length.
func EncodeUint32Array(s *State, arr []uint32) error {
	if err := EncodeUint(s, uint64(len(arr))); err != nil {
		return err
	}
	if s.Start+len(arr)*4 > len(s.Buffer) {
		return ErrBufferTooSmall
	}
	for _, val := range arr {
		binary.LittleEndian.PutUint32(s.Buffer[s.Start:], val)
		s.Start += 4
	}
	return nil
}

// DecodeUint32Array decodes a compact uint length prefixed uint32 array.
func DecodeUint32Array(s *State) ([]uint32, error) {
	count, err := DecodeUint(s)
	if err != nil {
		return nil, err
	}
	intCount := int(count)
	if intCount < 0 || s.Start+intCount*4 > s.End {
		return nil, ErrUnexpectedEOF
	}
	res := make([]uint32, intCount)
	for i := 0; i < intCount; i++ {
		res[i] = binary.LittleEndian.Uint32(s.Buffer[s.Start:])
		s.Start += 4
	}
	return res, nil
}
