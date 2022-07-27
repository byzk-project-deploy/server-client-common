package serverclientcommon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	sshServer "github.com/gliderlabs/ssh"
	"golang.org/x/crypto/ssh"
)

type ShellSettingOption struct {
	Name string
	Args []string
}

// CommandRunOption 命令运行选项
type CommandRunOption struct {
	WorkDir string
	Env     []string
}

// SystemCallOptionMarshal 系统调用选项序列化
func (s *CommandRunOption) SystemCallOptionMarshal(session *ssh.Session) error {
	if s == nil {
		return nil
	}
	data, err := json.Marshal(s)
	if err != nil {
		return fmt.Errorf("序列化数据格式失败: %s", err.Error())
	}

	session.Setenv(systemCallOptionEnvKey, base64.StdEncoding.EncodeToString(data))
	return nil
}

const systemCallOptionEnvKey = "SYSTEM_COMMAND_CALL_OPTION"

// SystemCallOptionName 系统调用选项反序列化
func SystemCallCommandRunOptionUnmarshal(s sshServer.Session) (*CommandRunOption, error) {

	sessionEnv := s.Environ()
	for i := range sessionEnv {
		envStr := sessionEnv[i]
		index := strings.Index(envStr, "=")
		if index == -1 {
			continue
		}
		key := envStr[:index]
		if key != systemCallOptionEnvKey {
			continue
		}
		val := envStr[index+1:]
		rawData, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return nil, fmt.Errorf("数据包格式转换失败: %s", err.Error())
		}

		var r *CommandRunOption
		if err = json.Unmarshal(rawData, &r); err != nil {
			return nil, err
		}
		return r, nil
	}
	return nil, io.EOF
}

// SystemCallOption 系统调用命令参数
type SystemCallOption struct {
	// Name 请求的服务名称
	Name string
	// Rand 随机数
	Rand string
	// Addr 服务器返回地址
	Addr string
}

// SystemCmdOptions 系统命令选项
type SystemCmdOptions struct {
	// WorkDir 工作目录
	WorkDir string
	// Env 环境变量
	Env []string
}
