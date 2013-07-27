package frame

import(
	"bytes"
	"github.com/project8/swarm/gomonarch"
	"github.com/project8/swarm/gomonarch/record"
)

const (
	BadMonarchPtr int = -1
	NoMoreData int = -2
)

type Position struct {
	RecIdx, SampIdx uint64
}

type FrameError struct {
	ErrCode int
	What string
}

func (f FrameError) Error() string {
	return f.What
}

type Framer struct {
	m *gomonarch.Monarch
	r, next_r *record.MonarchRecord

	// size of the frame in samples
	Size uint64

	// are we currently overlapped?
	overlapped bool

	// Current frame offset
	Offset, End Position
}

type Frame struct {
	source *Framer
	Data []byte
}

// Creating a framer essentially opens a view of the data which
// begins at the current position of the open file pointed to by
// the argument of the function.  If the argument does not point to
// an open file, or there is no more data to be had, e will be non-nil
// and f will be nil.
func NewFramer(m *gomonarch.Monarch, width uint64) (f *Framer, e error) {
	if m == nil {
		e = FrameError{
			ErrCode: BadMonarchPtr,
			What: "pointer is not a valid monarch instance.",
		}
		f = nil
		return
	} 
	r, rec_err := m.NextRecord()
	if rec_err != nil {
		e = FrameError{
			ErrCode: NoMoreData,
			What: "no data available from file",
		}
		f = nil
		return
	}
	f = &Framer{
		m: m,
		r: r,
		next_r: nil,
		Size: width,
		overlapped: false,
		Offset: Position{
			RecIdx: r.RecId, 
			SampIdx: 0,
		},
		End: Position{
			RecIdx: r.RecId,
			SampIdx: width,
		},
	}
	return
}

// Advancing a frame means jumping ahead exactly N samples where N is
// the length of the frame.  
// The logic is fairly simple: check the current position plus N, and
// see if it overwraps the available data in the record.  If it does,
// grab the next record, set next_r equal to it, and create a slice
// that straddles the data in the two records.  If it doesn't, simply
// return the next frame's worth of data.
func (s *Framer) Advance() (f *Frame, e error) {	
	// If we are currently overlapped...
	if s.overlapped {
		s.r = s.next_r
		s.Offset = s.End
		s.overlapped = false
	}

	s.Offset.SampIdx += s.Size
	tgt_end := s.Offset.SampIdx + s.Size
	// The simplest case - we are still inside a single record.
	if tgt_end <= uint64(s.m.RecordLength()) {
		f = &Frame{
			Data: s.r.Data[s.Offset.SampIdx:tgt_end],
		}
		e = nil
		s.End.SampIdx += s.Size
		return
	} else { // It might just be time to get a new record.
		if tgt_end > uint64(s.m.RecordLength()) {
			r, r_er := s.m.NextRecord()
			if r_er != nil {
				e = FrameError{
					ErrCode: NoMoreData,
					What: "data stream exhausted.",
				}
				f = nil
				return
			}
			s.r = r
			s.next_r = nil

			s.Offset = Position{
				SampIdx: 0,
				RecIdx: s.r.RecId,
			}

			s.End = Position{
				SampIdx: s.Size,
				RecIdx: s.r.RecId,
			}

			f = &Frame{
				Data: s.r.Data[s.Offset.SampIdx:s.End.SampIdx],
			}
			e = nil
			return
		} else { // Otherwise, we are overlapped.
			s.overlapped = true
			r, r_er := s.m.NextRecord()
			if r_er != nil {
				e = FrameError{
					ErrCode: NoMoreData,
					What: "data stream exhausted.",
				}
				f = nil
				return
			}
			s.next_r = r

			var nothing []byte 

			s_r := s.r.Data[s.Offset.SampIdx:len(s.r.Data)]
			overlap := s.Size - uint64(len(s_r))
			s_next := s.next_r.Data[0:overlap]
			s.End = Position{
				RecIdx: s.next_r.RecId,
				SampIdx: overlap,
			}
			f = &Frame{
				Data: bytes.Join([][]byte{s_r,s_next}, nothing),
			}
			e = nil
			return
		}
	}

	return
}
