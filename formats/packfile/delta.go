package packfile

// See https://github.com/git/git/blob/49fa3dc76179e04b0833542fa52d0f287a4955ac/delta.h
// https://github.com/git/git/blob/c2c5f6b1e479f2c38e0e01345350620944e3527f/patch-delta.c,
// and https://github.com/tarruda/node-git-core/blob/master/src/js/delta.js
// for details about the delta format.

const deltaSizeMin = 4

// PatchDelta returns the result of applying the modification deltas in delta to src.
func PatchDelta(src, delta []byte) []byte {
	if len(delta) < deltaSizeMin {
		return nil
	}

	srcSize, delta := decodeLEB128(delta)
	if srcSize != uint(len(src)) {
		return nil
	}

	targetSize, delta := decodeLEB128(delta)
	remainingTargetSize := targetSize

	var dest []byte
	var cmd byte
	for {
		cmd = delta[0]
		delta = delta[1:]
		if isCopyFromSrc(cmd) {
			var offset, size uint
			offset, delta = decodeOffset(cmd, delta)
			size, delta = decodeSize(cmd, delta)
			if invalidSize(size, targetSize) ||
				invalidOffsetSize(offset, size, srcSize) {
				break
			}
			dest = append(dest, src[offset:offset+size]...)
			remainingTargetSize -= size
		} else if isCopyFromDelta(cmd) {
			size := uint(cmd) // cmd is the size itself
			if invalidSize(size, targetSize) {
				break
			}
			dest = append(dest, delta[0:size]...)
			remainingTargetSize -= size
			delta = delta[size:]
		} else {
			return nil
		}

		if remainingTargetSize <= 0 {
			break
		}
	}

	return dest
}

// Decodes a number encoded as an unsigned LEB128 at the start of some
// binary data and returns the decoded number and the rest of the
// stream.
//
// This must be called twice on the delta data buffer, first to get the
// expected source buffer size, and again to get the target buffer size.
func decodeLEB128(input []byte) (uint, []byte) {
	var result, bytesDecoded uint
	var b byte
	for {
		b = input[bytesDecoded]
		result |= (uint(b) & payload) << (bytesDecoded * 7) // concats 7 bits chunks
		bytesDecoded++

		if uint(b)&continuation == 0 || bytesDecoded == uint(len(input)) {
			break
		}
	}

	return result, input[bytesDecoded:]
}

const (
	payload      = 0x7f // 0111 1111
	continuation = 0x80 // 1000 0000
)

func isCopyFromSrc(cmd byte) bool {
	return (cmd & 0x80) != 0
}

func isCopyFromDelta(cmd byte) bool {
	return (cmd&0x80) == 0 && cmd != 0
}

func decodeOffset(cmd byte, delta []byte) (uint, []byte) {
	var offset uint
	if (cmd & 0x01) != 0 {
		offset = uint(delta[0])
		delta = delta[1:]
	}
	if (cmd & 0x02) != 0 {
		offset |= uint(delta[0]) << 8
		delta = delta[1:]
	}
	if (cmd & 0x04) != 0 {
		offset |= uint(delta[0]) << 16
		delta = delta[1:]
	}
	if (cmd & 0x08) != 0 {
		offset |= uint(delta[0]) << 24
		delta = delta[1:]
	}

	return offset, delta
}

func decodeSize(cmd byte, delta []byte) (uint, []byte) {
	var size uint
	if (cmd & 0x10) != 0 {
		size = uint(delta[0])
		delta = delta[1:]
	}
	if (cmd & 0x20) != 0 {
		size |= uint(delta[0]) << 8
		delta = delta[1:]
	}
	if (cmd & 0x40) != 0 {
		size |= uint(delta[0]) << 16
		delta = delta[1:]
	}
	if size == 0 {
		size = 0x10000
	}

	return size, delta
}

func invalidSize(size, targetSize uint) bool {
	return size > targetSize
}

func invalidOffsetSize(offset, size, srcSize uint) bool {
	return sumOverflows(offset, size) ||
		offset+size > srcSize
}

func sumOverflows(a, b uint) bool {
	return a+b < a
}
