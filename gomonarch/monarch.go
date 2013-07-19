package gomonarch

import (
	"os"
	"bytes"
	"syscall"
	"encoding/binary"
	"code.google.com/p/goprotobuf/proto"
	"github.com/project8/swarm/gomonarch/header"
	"github.com/project8/swarm/gomonarch/record"
)

type FileMode int
const (
	ReadMode FileMode = 1
	WriteMode FileMode = 2
)

type Monarch struct {
	f *os.File
	h header.MonarchHeader
}

func Open(fname string, mode FileMode) (*Monarch, error) {
	switch mode {
	case ReadMode:
		return open_readmode(fname)
	case WriteMode:
		return open_writemode(fname)
	}
	return nil, nil
}

func open_readmode(fname string) (*Monarch, error) {
	flags := syscall.O_RDONLY
	file, err := os.OpenFile(fname, flags, 0666)
	if err != nil {
		return nil, err
	}
	var magic int64
	magic_err := binary.Read(file, binary.LittleEndian, &magic)
	if magic_err != nil {
		return nil, magic_err
	}
	var hdr header.MonarchHeader
	pbuf_buf := make([]byte, magic, magic)
	_, buf_err := file.Read(pbuf_buf)
	if buf_err != nil {
		return nil, buf_err
	}
	decode_err := proto.Unmarshal(pbuf_buf, &hdr)
	if decode_err != nil {
		return nil, decode_err
	}
	return &Monarch{file, hdr}, nil
}

func open_writemode(fname string) (*Monarch, error) {
	return nil, nil
}

func Close(m *Monarch) error {
	return m.f.Close()
}

func NumChannels(m *Monarch) uint32 {
	return m.h.GetAcqMode()
}

func RecordLength(m *Monarch) uint32 {
	return m.h.GetRecSize()
}

func AcqRate(m *Monarch) float64 {
	return m.h.GetAcqRate()
}

func NextRecord(m *Monarch) (r *gomonarch.MonarchRecord, e error) {
	s := RecordLength(m)
	r = &gomonarch.MonarchRecord{Data: make([]byte, s, s)}
	return r,unmarshal_record(m.f,r)
}

func unmarshal_record(f *os.File, r *gomonarch.MonarchRecord) error {
	ar := make([]byte, 8,8)
	buf := bytes.NewBuffer(ar)
	_, acq_err := f.Read(ar)
	acq_id, _ := binary.ReadUvarint(buf)
	if acq_err != nil {
		return acq_err
	}
	_, rec_err := f.Read(ar)
	rec_id, _ := binary.ReadUvarint(buf)
	if rec_err != nil {
		return rec_err
	}
	_, clock_err := f.Read(ar)
	clock, _ := binary.ReadUvarint(buf)
	if clock_err != nil {
		return clock_err
	}
	_, data_err := f.Read(r.Data)
	if data_err == nil {
		r.AcqId = acq_id
		r.RecId = rec_id
		r.Clock = clock
	}
	return data_err
}
