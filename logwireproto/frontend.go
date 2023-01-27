package logwireproto

import "net"

func protoParseString(in []byte, insz int) {

}

type LogicalFrontendProtocMessage interface {
	iLogicalFrontend()
}

type XLogRecPtr int64
type TimestampTz int64
type TransactionId int32
type Oid int32

type LogicalBegin struct {
	// 1 byte 'B'
	LsnFinal        XLogRecPtr
	CommitTimestamp TimestampTz
	XID             TransactionId
}

type LogicalMessage struct {
	// 1 byte 'M'
	XID    int32
	Flags  int8 /* not used, should be always 0*/
	Lsn    XLogRecPtr
	Prefix string
	Len    int32
	Data   []byte
}

type LogicalCommit struct {
	// 1 byte 'C'
	Flags     int8 /* should be always 0 */
	XID       XLogRecPtr
	CommitLSN int64
	EndLSN    int64
	CommitTS  TimestampTz
}

type LogicalOrigin struct {
	// single byte 'o'
	LSN  XLogRecPtr
	Name string
}

type LogicalRelation struct {
	// single byte 'R'
	XID          TransactionId
	OID          Oid
	Namespace    string
	Relname      string
	Relreplident int8
	ColNumber    int16
	ColFlags     int8

	Colname   string
	TypeOid   Oid
	atttypmod int32
}

type LogicalType struct {
	// single byte 'Y'
	XID       TransactionId
	TypeOid   Oid
	Namespace string
	Name      string
}

type LogicalInsert struct {
	// single byte 'I'
	XID TransactionId
}

type LogicalFrontend struct {
	conn net.Conn
}

func NewLocicalFrontend(conn net.Conn) *LogicalFrontend {
	return &LogicalFrontend{
		conn: conn,
	}
}

func (lf *LogicalFrontend) read(b []byte, n int) {

}

func (lf *LogicalFrontend) Recv() LogicalFrontend {

}
