package serverclientcommon

import (
	"encoding/json"
	"fmt"
	transportstream "github.com/go-base-lib/transport-stream"
	"github.com/gogo/protobuf/proto"
	"io"
	"net"
)

type ExchangeData []byte

func (e ExchangeData) UnmarshalJson(i any) error {
	if e == nil {
		return nil
	}

	if err := json.Unmarshal(e, i); err != nil {
		return fmt.Errorf("数据尝试从JSON反序列化到结构体失败: %s", err.Error())
	}

	return nil
}

func (e ExchangeData) UnmarshalProto(msg proto.Message) error {
	if e == nil {
		return nil
	}

	if err := proto.Unmarshal(e, msg); err != nil {
		return fmt.Errorf("数据尝试从proto反序列化到结构体失败: %s", err.Error())
	}
	return nil
}

func NewExchangeDataByStr(str string) ExchangeData {
	return []byte(str)
}

func NewExchangeDataByJson(data any) (ExchangeData, error) {
	marshal, err := json.Marshal(data)
	return marshal, err
}

func NewExchangeDataByJsonMust(data any) ExchangeData {
	marshal, _ := json.Marshal(data)
	return marshal
}

func NewExchangeDataByProto(data proto.Message) (ExchangeData, error) {
	marshal, err := proto.Marshal(data)
	return marshal, err
}

func NewExchangeDataByProtoMust(data proto.Message) ExchangeData {
	marshal, _ := proto.Marshal(data)
	return marshal
}

type CmdHandler func(stream *transportstream.Stream, conn net.Conn) (ExchangeData, error)

var cmdMap = map[CmdName]CmdHandler{}

func CmdRoute(stream *transportstream.Stream, conn net.Conn) error {
	sendEndOk := false
	defer func() {
		if sendEndOk {
			return
		}
		_ = stream.WriteEndMsg()
		for {
			if _, err := stream.ReceiveMsg(); err == transportstream.StreamIsEnd {
				return
			}
		}
	}()
	defer func() {
		e := recover()
		if e != nil {
			errMsg := ""
			switch r := e.(type) {
			case string:
				errMsg = r
			case error:
				errMsg = r.Error()
			}
			_ = stream.WriteError(ErrCodeUnknown.Newf("未知的指令处理异常: %s", errMsg))
		}
	}()

	cmdBytes, err := stream.ReceiveMsg()
	if err != nil {
		_ = stream.WriteError(ErrCodeReadCommand.New("读取命令码失败: " + err.Error()))
		return err
	}

	cmdName := CmdName(cmdBytes)
	cmdHandle, ok := cmdMap[cmdName]
	if !ok {
		_ = stream.WriteError(ErrCodeCommandUndefined.Newf("命令[%s]未被识别", cmdName))
		return nil
	}

	if err = stream.WriteMsg(nil, transportstream.MsgFlagSuccess); err != nil {
		return err
	}

	if nextData, err := cmdHandle(stream, conn); err != nil {
		switch e := err.(type) {
		case *transportstream.ErrInfo:
			_ = stream.WriteError(e)
		default:
			_ = stream.WriteError(ErrCodeUnknown.New(err.Error()))
		}
		return nil
	} else {
		if err = stream.WriteEndMsgWithData(nextData); err != nil {
			return nil
		}
		sendEndOk = true

		for {
			if _, err = stream.ReceiveMsg(); err == transportstream.StreamIsEnd {
				return nil
			}
		}

	}

}

type RWStreamInterface interface {
	WriteStreamInterface
	ReadLine() ([]byte, bool, error)
}

type WriteStreamInterface interface {
	Write([]byte) (int, error)

	Flush() error
}

type ExchangeOption struct {
	// StreamHandle 流拦截器, 返回bool来确定是否准备跳出流读取
	StreamHandle func(exchangeData ExchangeData) (ExchangeData, error)
	// Data 要发送的数据
	Data any
}

type CmdName string

func emptyStreamHandler(exchangeData ExchangeData) (ExchangeData, error) {
	return nil, transportstream.StreamIsEnd
}

// SendCommand 发送一条命令到对端
func (c CmdName) SendCommand(stream *transportstream.Stream) error {
	if err := stream.WriteMsg([]byte(c), transportstream.MsgFlagSuccess); err != nil {
		return err
	}

	if _, err := stream.ReceiveMsg(); err != nil {
		return err
	}
	return nil
}

// ExchangeWithOption 交换数据到对端，数据为一来一回
func (c CmdName) ExchangeWithOption(stream *transportstream.Stream, option *ExchangeOption) (ExchangeData, error) {
	defer stream.WriteEndMsg()

	if option.StreamHandle == nil {
		option.StreamHandle = emptyStreamHandler
	}

	if err := c.SendCommand(stream); err != nil {
		return nil, err
	}

	if option.Data != nil {
		if err := stream.WriteJsonMsg(option.Data); err != nil {
			return nil, err
		}
	} else {
		if err := stream.WriteMsg(nil, transportstream.MsgFlagSuccess); err != nil {
			return nil, err
		}
	}

	for {
		msg, err := stream.ReceiveMsg()
		if err == transportstream.StreamIsEnd {
			return msg, nil
		}

		if err != nil {
			for {
				if _, e := stream.ReceiveMsg(); e == transportstream.StreamIsEnd || e == io.EOF {
					break
				}
			}
			return msg, err
		}

		nextData, err := option.StreamHandle(msg)
		if err == transportstream.StreamIsEnd {
			if err = stream.WriteEndMsgWithData(nextData); err != nil {
				return nil, fmt.Errorf("接收结束消息失败: %s", err.Error())
			}
			continue
		}

		if err != nil {
			switch e := err.(type) {
			case *transportstream.ErrInfo:
				if err = stream.WriteError(e); err != nil {
					return nil, err
				}
			default:
				_err := ErrCodeUnknown.New(err.Error())
				_err.RawData = nextData
				if err = stream.WriteError(_err); err != nil {
					return nil, err
				}
			}
			continue
		}
		if err = stream.WriteMsg(nextData, transportstream.MsgFlagSuccess); err != nil {
			return nil, err
		}
	}

}

func (c CmdName) ExchangeWithData(data any, stream *transportstream.Stream) (ExchangeData, error) {
	return c.ExchangeWithOption(stream, &ExchangeOption{
		Data: data,
	})
}

func (c CmdName) Exchange(stream *transportstream.Stream) (ExchangeData, error) {
	return c.ExchangeWithData(nil, stream)
}

func (c CmdName) Registry(handle CmdHandler) {
	cmdMap[c] = handle
}

var (
	// CmdHello hello测试
	CmdHello CmdName = "/hello"
	// CmdSystemCall 系统调用
	CmdSystemCall CmdName = "/system/call"
	// CmdSystemShellList 系统调用获取可用的shell列表
	CmdSystemShellList CmdName = "/system/shell/list"
	// CmdSystemShellCurrent 获取当前shell的内容
	CmdSystemShellCurrent CmdName = "/system/shell/current"
	// CmdSystemShellCurrentSetting 设置当前使用中的shell
	CmdSystemShellCurrentSetting CmdName = "/system/shell/current/setting"
	// CmdSystemDirPath 系统内目录路径验证
	CmdSystemDirPath CmdName = "/system/dir/path/verify"
	// CmdPluginInstall 插件安装
	CmdPluginInstall CmdName = "/plugin/install"
	// CmdPluginList 插件列表
	CmdPluginList CmdName = "/plugin/list"
)
