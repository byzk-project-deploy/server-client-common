package serverclientcommon

import (
	"bufio"
	"bytes"
	"fmt"
)

type CmdHandler func(result *Result, conn RWStreamInterface) *Result

var cmdMap = map[string]CmdHandler{}

func CmdRoute(cmdName string, data []byte, rw RWStreamInterface) {
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
			ErrResult(ErrServerInside, "未知的指令处理异常: "+errMsg).WriteToEnd(rw)
		}
	}()
	dataR := &Result{}
	if err := dataR.Parse(data); err != nil {
		ErrResult(ErrDataParse, "数据包解析失败: "+err.Error()).WriteToEnd(rw)
		return
	}

	cmdHandle, ok := cmdMap[cmdName]
	if !ok {
		ErrResult(ErrCodeNotFount, "未识别的指令").WriteToEnd(rw)
		return
	}
	r := cmdHandle(dataR, rw)
	if r == nil {
		SuccessResultEmpty().WriteToEnd(rw)
		return
	}
	r.WriteToEnd(rw)
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
	StreamHandle func(r *Result) (bool, *Result, error)
	// Data 要发送的数据
	Data any
}

type CmdName string

func (c CmdName) ExchangeWithOption(rw RWStreamInterface, option *ExchangeOption) (*Result, error) {
	var isContinue bool
	data := option.Data
	rw.Write(append([]byte(c), '\n'))
	rw.Flush()

	line, _, err := rw.ReadLine()
	if err != nil {
		return nil, fmt.Errorf("数据包读取失败: %s", err.Error())
	}

	r := &Result{}
	if err = r.Parse(line); err != nil {
		return nil, fmt.Errorf("数据包内容转换失败: %s", err.Error())
	}

	if r.Error {
		return nil, fmt.Errorf("code: %s, msg: %s", r.Code, r.Msg)
	}

	SuccessResult(data).WriteTo(rw)
	rw.Flush()

	tempBuf := &bytes.Buffer{}
	for {
		l, isPrefix, err := rw.ReadLine()
		if err != nil {
			return nil, fmt.Errorf("指令响应包读取失败: %s", err.Error())
		}

		if isPrefix {
			tempBuf.Write(l)
			continue
		}

		if tempBuf.Len() > 0 {
			l = append(tempBuf.Bytes(), l...)
			tempBuf.Reset()
		}

		r = &Result{}
		if err = r.Parse(l); err != nil {
			return nil, fmt.Errorf("解析指令响应包失败: %s", err.Error())
		}

		if option.StreamHandle != nil {
			isContinue, r, err = option.StreamHandle(r)
			if err != nil {
				return nil, err
			}

			if isContinue {
				continue
			}
			if r == nil {
				r = SuccessResultEmpty()
			}
		}

		return r, nil
	}
}

func (c CmdName) ExchangeWithData(data any, rw RWStreamInterface) (*Result, error) {
	return c.ExchangeWithOption(rw, &ExchangeOption{
		Data: data,
	})
}

func (c CmdName) Exchange(rw *bufio.ReadWriter) (*Result, error) {
	return c.ExchangeWithData(nil, rw)
}

func (c CmdName) Registry(handle CmdHandler) {
	cmdMap[string(c)] = handle
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
)
