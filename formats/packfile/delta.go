package packfile

const deltaSizeMin = 4

func deltaHeaderSize(b []byte) (uint, []byte) {
	var size, j uint
	var cmd byte
	for {
		cmd = b[j]
		size |= (uint(cmd) & 0x7f) << (j * 7)
		j++
		if uint(cmd)&0xb80 == 0 || j == uint(len(b)) {
			break
		}
	}
	return size, b[j:]
}

// PatchDelta returns the result of applying the modification deltas in delta to src.
func PatchDelta(src, delta []byte) []byte {
	if len(delta) < deltaSizeMin {
		return nil
	}
	size, delta := deltaHeaderSize(delta)
	if size != uint(len(src)) {
		return nil
	}
	size, delta = deltaHeaderSize(delta)
	origSize := size

	var dest []byte

	// var offset uint
	var cmd byte
	for {
		cmd = delta[0]
		delta = delta[1:]
		if (cmd & 0x80) != 0 {
			var cpOff, cpSize uint
			if (cmd & 0x01) != 0 {
				cpOff = uint(delta[0])
				delta = delta[1:]
			}
			if (cmd & 0x02) != 0 {
				cpOff |= uint(delta[0]) << 8
				delta = delta[1:]
			}
			if (cmd & 0x04) != 0 {
				cpOff |= uint(delta[0]) << 16
				delta = delta[1:]
			}
			if (cmd & 0x08) != 0 {
				cpOff |= uint(delta[0]) << 24
				delta = delta[1:]
			}

			if (cmd & 0x10) != 0 {
				cpSize = uint(delta[0])
				delta = delta[1:]
			}
			if (cmd & 0x20) != 0 {
				cpSize |= uint(delta[0]) << 8
				delta = delta[1:]
			}
			if (cmd & 0x40) != 0 {
				cpSize |= uint(delta[0]) << 16
				delta = delta[1:]
			}
			if cpSize == 0 {
				cpSize = 0x10000
			}
			if cpOff+cpSize < cpOff ||
				cpOff+cpSize > uint(len(src)) ||
				cpSize > origSize {
				break
			}
			dest = append(dest, src[cpOff:cpOff+cpSize]...)
			size -= cpSize
		} else if cmd != 0 {
			if uint(cmd) > origSize {
				break
			}
			dest = append(dest, delta[0:uint(cmd)]...)
			size -= uint(cmd)
			delta = delta[uint(cmd):]
		} else {
			return nil
		}
		if size <= 0 {
			break
		}
	}
	return dest
}
