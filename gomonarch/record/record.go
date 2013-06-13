package record

import (
	"github.com/project8/swarm/core/types"
)

type MonarchRecord struct {
	_AcqId types.AcqIDType
	_RecId types.RecIDType
	_Clock types.ClockType
	_Data []byte
}

func AcqeId(m *MonarchRecord) uint64 {
	
}
