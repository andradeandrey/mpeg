package video

type motionCompensationState struct {
	active  bool
	i       int
	channel []uint8
	stride  int
	h_half  bool
	v_half  bool
}

func (b *block) motion_compensation(motionVectors motionVectorData, i, mb_row, mb_addr int, fs frameStore) {

	bState := motionCompensationState{
		active: ((motionVectors.previous & motionVectorsFormed_FrameBackward) == motionVectorsFormed_FrameBackward) &&
			fs.future != nil,
	}
	fState := motionCompensationState{
		active: ((motionVectors.previous & motionVectorsFormed_FrameForward) == motionVectorsFormed_FrameForward) &&
			fs.past != nil,
	}

	// project _future_ temporal sample _backward_...
	if bState.active {
		horizontal, vertical := motionVectors.actual[0][1][0], motionVectors.actual[0][1][1]

		// Scale Cb/Cr vectors
		switch i {
		case 4, 5:
			horizontal >>= 1
			vertical >>= 1
		}

		// is half pel?
		bState.h_half, bState.v_half = (horizontal&1) == 1, (vertical&1) == 1
		// scale down by half
		horizontal, vertical = horizontal>>1, vertical>>1

		image := fs.future

		// channel switch
		switch i {
		case 0, 1, 2, 3:
			bState.channel = image.Y
		case 4:
			bState.channel = image.Cb
		case 5:
			bState.channel = image.Cr
		}

		// stride and index switch
		switch i {
		case 0, 1, 2, 3:
			bState.stride = image.YStride
			bState.i = (((mb_row * 16) + (i&2)<<2) * bState.stride) + (mb_addr * 16) + (i&1)<<3
		case 4, 5:
			bState.stride = image.CStride
			bState.i = (mb_row * 8 * bState.stride) + (mb_addr * 8)
		}

		bState.i += vertical * bState.stride
		bState.i += horizontal
	}

	// project _past_ temporal sample _forward_...
	if fState.active {
		horizontal, vertical := motionVectors.actual[0][0][0], motionVectors.actual[0][0][1]

		// Scale Cb/Cr vectors
		switch i {
		case 4, 5:
			horizontal >>= 1
			vertical >>= 1
		}

		// is half pel?
		fState.h_half, fState.v_half = (horizontal&1) == 1, (vertical&1) == 1
		// scale down by half
		horizontal, vertical = horizontal>>1, vertical>>1

		image := fs.past

		// channel switch
		switch i {
		case 0, 1, 2, 3:
			fState.channel = image.Y
		case 4:
			fState.channel = image.Cb
		case 5:
			fState.channel = image.Cr
		}

		switch i {
		case 0, 1, 2, 3:
			fState.stride = image.YStride
			fState.i = (((mb_row * 16) + (i&2)<<2) * fState.stride) + (mb_addr * 16) + (i&1)<<3
		case 4, 5:
			fState.stride = image.CStride
			fState.i = (mb_row * 8 * fState.stride) + (mb_addr * 8)
		}

		fState.i += vertical * fState.stride
		fState.i += horizontal
	}

	for v := 0; v < 8; v++ {
		for u := 0; u < 8; u++ {
			var pel int32 = 0
			var samples uint = 0
			if fState.active {
				i := fState.i + (v * fState.stride) + u
				samples++
				switch {
				case !fState.h_half && !fState.v_half:
					pel += int32(fState.channel[i])

				case fState.h_half && !fState.v_half:
					pel += (int32(fState.channel[i]) +
						int32(fState.channel[i+1])) / 2

				case !fState.h_half && fState.v_half:
					pel += (int32(fState.channel[i]) +
						int32(fState.channel[i+fState.stride])) / 2

				case fState.h_half && fState.v_half:
					pel += (int32(fState.channel[i]) +
						int32(fState.channel[i+1]) + int32(fState.channel[i+fState.stride]) + int32(fState.channel[i+fState.stride+1])) / 4
				}
			}
			if bState.active {
				i := bState.i + (v * bState.stride) + u
				samples++
				switch {
				case !bState.h_half && !bState.v_half:
					pel += int32(bState.channel[i])

				case bState.h_half && !bState.v_half:
					pel += (int32(bState.channel[i]) +
						int32(bState.channel[i+1])) / 2

				case !bState.h_half && bState.v_half:
					pel += (int32(bState.channel[i]) +
						int32(bState.channel[i+bState.stride])) / 2

				case bState.h_half && bState.v_half:
					pel += (int32(bState.channel[i]) +
						int32(bState.channel[i+1]) +
						int32(bState.channel[i+bState.stride]) +
						int32(bState.channel[i+bState.stride+1])) / 4
				}
			}
			b[v*8+u] += pel >> (samples - 1)
		}
	}
}
