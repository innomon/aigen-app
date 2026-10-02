package compactenc

// PreencodeInt calculates byte size required to encode a signed int64 using ZigZag mapping.
func PreencodeInt(val int64) int {
	u := (uint64(val) << 1) ^ uint64(val>>63)
	return PreencodeUint(u)
}

// EncodeInt encodes a signed integer using ZigZag compact encoding.
func EncodeInt(s *State, val int64) error {
	u := (uint64(val) << 1) ^ uint64(val>>63)
	return EncodeUint(s, u)
}

// DecodeInt decodes a signed integer using ZigZag compact encoding.
func DecodeInt(s *State) (int64, error) {
	u, err := DecodeUint(s)
	if err != nil {
		return 0, err
	}
	return int64(u>>1) ^ -int64(u&1), nil
}
