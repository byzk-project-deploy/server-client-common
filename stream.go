package serverclientcommon

import (
	"bufio"
	"net"

	"github.com/gogo/protobuf/proto"
	"github.com/teamManagement/common/errors"
	"github.com/teamManagement/common/utils"
)

type Stream struct {
	rw   *bufio.ReadWriter
	conn net.Conn
}

func (s *Stream) WriteJsonMsg(msg any) error {
	return utils.WriteJsonMsgToWriter(s.rw.Writer, msg)
}

func (s *Stream) ReceiveJsonMsg(msg any) error {
	return utils.ReadJsonMsgByReader(s.rw.Reader, msg)
}

func (s *Stream) WriteProtoMsg(msg proto.Message) error {
	return utils.WriteProtoMsgToWriter(s.rw.Writer, msg)
}

func (s *Stream) ReceiveProtoMsg(msg proto.Message) error {
	return utils.ReadProtoMsgByReader(s.rw.Reader, msg)
}

func (s *Stream) WriteError(err *errors.Error) error {
	return utils.WriteErrToWriter(s.rw.Writer, err)
}

func (s *Stream) WriteMsg(data []byte) error {
	return utils.WriteBytesToWriter(s.rw.Writer, data, true)
}

func (s *Stream) ReceiveMsg() ([]byte, bool, error) {
	return utils.ReadBytesByReader(s.rw.Reader)
}

func NewStream(conn net.Conn) *Stream {
	return &Stream{
		rw:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		conn: conn,
	}
}
