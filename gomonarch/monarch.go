package gomonarch

/*
#cgo LDFLAGS: -L/Users/kofron/Code/C/monarch_c -lmonarch
#cgo CFLAGS: -I/Users/kofron/Code/C/monarch_c/include

#include <stdlib.h>
#include "monarch.h"
*/
import "C"

import (
	"unsafe"
	"errors"
)

type Monarch struct {
	fd *(C.monarch_fd)
}

type MonarchRecord struct {
	rec *(C.monarch_record)
}

func NumChannels(m *Monarch) (n int) {
	n = int(C.monarch_n_channels(m.fd))
	return
}

func RecordLength(m *Monarch) (n int) {
	n = int(C.monarch_record_len(m.fd))
	return
}

func Open(name string) (*Monarch, error) {
	// This only opens for reading at the moment
	var monarch *(C.monarch_fd)
	c_fname := C.CString(name)
	defer C.free(unsafe.Pointer(c_fname))
	monarch = C.monarch_open(c_fname, C.read_mode)
	if monarch == nil {
		return nil, errors.New("couldn't open file for reading!")
	}
	
	return &Monarch{monarch}, nil
}

func Close(monarch *Monarch) {
	C.monarch_close(monarch.fd)
}

func NextRecord(monarch *Monarch) ([]byte, error) {
	rec := C.monarch_new_record_alloc(4194304)
	res := C.monarch_read_next_record(monarch.fd,rec)
	if int(res) != 0 {
		return nil, errors.New("oh no")
	}
	c_ar := C.monarch_record_data(rec)
	return C.GoBytes(unsafe.Pointer(c_ar), 4194304), nil
}
