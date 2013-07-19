package gomonarch

import (
//	"github.com/project8/swarm/core/types"
)

type MonarchRecord struct {
	AcqId uint64 //types.AcqIDType
	RecId uint64 //types.RecIDType
	Clock uint64 //types.ClockType
	Data []byte
}
