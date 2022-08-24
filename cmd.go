package serverclientcommon

import (
	"encoding/json"
	"fmt"
	transportstream "github.com/go-base-lib/transport-stream"
	"github.com/gogo/protobuf/proto"
	"io"
	"net"
	"strings"
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
			if _, err := stream.ReceiveMsg(); err == transportstream.StreamIsEnd || err == io.EOF || strings.Contains(err.Error(), "connection reset by peer") {
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
		if err == transportstream.StreamIsEnd {
			return nil
		}
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
	// StreamHandle 流拦截器
	StreamHandle func(exchangeData ExchangeData, stream *transportstream.Stream) (ExchangeData, error)
	// StreamErrHandle 处理除了 transportstream.StreamIsEnd 与 io.EOF 之外的所有异常
	// breakStream 表示是否中断流, 如果返回true则向对断发送 transportstream.StreamIsEnd 指令并跳出流监听
	// targetErr 将把转换之后的异常信息发送至服务器端
	StreamErrHandle func(exchangeData ExchangeData, err error) (breakStream bool, targetErr *transportstream.ErrInfo)
	// Data 要发送的数据
	Data any
}

type CmdName string

func emptyStreamHandler(exchangeData ExchangeData, stream *transportstream.Stream) (ExchangeData, error) {
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
			if err != transportstream.StreamIsEnd && option.StreamErrHandle != nil {
				breakStream, e := option.StreamErrHandle(msg, err)
				if e != nil {
					_ = stream.WriteError(e)
				}

				if breakStream {
					return msg, e
				}
				continue
			}
			for {
				if _, e := stream.ReceiveMsg(); e == transportstream.StreamIsEnd || e == io.EOF || strings.Contains(err.Error(), "connection reset by peer") {
					break
				}
			}
			return msg, err
		}

		nextData, err := option.StreamHandle(msg, stream)
		if err == transportstream.StreamIsEnd {
			if err = stream.WriteEndMsgWithData(nextData); err != nil {
				return nil, fmt.Errorf("接收结束消息失败: %s", err.Error())
			}
			return nil, nil
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
		if nextData != nil {
			if err = stream.WriteMsg(nextData, transportstream.MsgFlagSuccess); err != nil {
				return nil, err
			}
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
	// CmdPluginEnabled 启用插件
	CmdPluginEnabled CmdName = "/plugin/enabled"
	// CmdPluginDisabled 禁用插件
	CmdPluginDisabled CmdName = "/plugin/disabled"
	// CmdPluginList 插件列表
	CmdPluginList CmdName = "/plugin/list"
	// CmdPluginInfo 插件详细信息
	CmdPluginInfo CmdName = "/plugin/info"
	// CmdPluginUninstall 插件卸载
	CmdPluginUninstall CmdName = "/plugin/uninstall"
	// CmdPluginInfoPromptList 插件info提示信息
	CmdPluginInfoPromptList CmdName = "/plugin/info/prompt"
	// CmdPluginEnable 开启插件
	CmdPluginEnable CmdName = "/plugin/enable"
	// CmdPluginDisable 禁用插件
	CmdPluginDisable CmdName = "/plugin/disable"
	// CmdKeyPair 密钥工具
	CmdKeyPair CmdName = "/inside/keypair"
	// CmdKeyPairRemoteClient 远程客户端密钥获取
	CmdKeyPairRemoteClient CmdName = "/inside/keypair/remote/client"
	// CmdRemoteServerList 远程服务列表
	CmdRemoteServerList CmdName = "/inside/remote/server/list"
	// CmdRemoteServerAdd 远程服务添加
	CmdRemoteServerAdd CmdName = "/inside/remote/server/add"
	// CmdRemoteServerDel 远程服务删除
	CmdRemoteServerDel CmdName = "/inside/remote/server/del"
	// CmdRemoteServerUpdate 远程服务器信息更新
	CmdRemoteServerUpdate CmdName = "/inside/remote/server/update"
	// CmdRemoteServerInfo 远程服务器信息查询
	CmdRemoteServerInfo CmdName = "/inside/remote/server/info"
	// CmdRemoteServerUpdateAlias 远程服务别名更新
	CmdRemoteServerUpdateAlias CmdName = "/inside/remote/server/alias"
	// CmdRemoteServerRepair 远程服务自动修复
	CmdRemoteServerRepair CmdName = "/inside/remote/server/repair"
	// CmdRemoteServerFileUpload 远程服务文件上传
	CmdRemoteServerFileUpload CmdName = "/inside/remote/server/file/upload"
	// CmdRemoteServerFileDownload 远程服务文件下载
	CmdRemoteServerFileDownload CmdName = "/inside/remote/server/file/download"
)
