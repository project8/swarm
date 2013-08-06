package record

type MonarchRecord struct {
	AcqId uint64 //types.AcqIDType
	RecId uint64 //types.RecIDType
	Clock int64 //types.ClockType
	Data []byte
}
