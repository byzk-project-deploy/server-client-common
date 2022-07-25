package serverclientcommon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

type ResultData string

func (r *ResultData) Unmarshal(i any) error {
	if r == nil {
		return io.EOF
	}
	s, err := base64.StdEncoding.DecodeString(string(*r))
	if err != nil {
		return fmt.Errorf("反序列化结果数据失败: %s", err.Error())
	}

	if err = json.Unmarshal(s, i); err != nil {
		return fmt.Errorf("解析数据结果内容失败: %s", err.Error())
	}
	return nil
}

func NewResultData(data any) (*ResultData, error) {
	d, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("序列化数据失败: %s", err.Error())
	}
	r := ResultData(base64.StdEncoding.EncodeToString(d))
	return &r, nil
}

// Result 结果
type Result struct {
	Msg       string
	Data      *ResultData
	StreamEnd bool
}

func (r *Result) Parse(parseData []byte) error {
	dest := make([]byte, base64.StdEncoding.DecodedLen(len(parseData)))
	n, err := base64.StdEncoding.Decode(dest, parseData)
	if err != nil {
		return fmt.Errorf("解析数据包失败, 错误信息: %s", err.Error())
	}

	if err = json.Unmarshal(dest[:n], r); err != nil {
		return fmt.Errorf("反序列化数据包失败: %s", err.Error())
	}
	return nil
}

func (r *Result) Marshal() ([]byte, error) {

	marshalData, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("序列化数据包失败: %s", err.Error())
	}
	return []byte(base64.StdEncoding.EncodeToString(marshalData)), nil
}

func (r *Result) WriteToEnd(w WriteStreamInterface) error {
	r.StreamEnd = true
	return r.WriteTo(w)
}

func (r *Result) WriteTo(w WriteStreamInterface) error {
	writeData, err := r.Marshal()
	if err != nil {
		return err
	}

	writeData = append(writeData, '\n')

	if _, err = w.Write(writeData); err != nil {
		return fmt.Errorf("写出数据包失败: %s", err.Error())
	}
	w.Flush()
	return nil
}

func ErrResult(code ErrCode, msg string) *Result {
	return &Result{
		Msg: msg,
	}
}

func ErrResultWithData(code ErrCode, msg string, data any) *Result {
	rData, _ := NewResultData(data)
	return &Result{
		Msg:  msg,
		Data: rData,
	}
}

func SuccessResult(data any) *Result {
	rd, _ := NewResultData(data)
	return &Result{
		Data: rd,
	}
}

func SuccessResultEmpty() *Result {
	return &Result{}
}
