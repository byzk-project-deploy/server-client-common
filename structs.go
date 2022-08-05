package serverclientcommon

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	rpcinterfaces "github.com/byzk-project-deploy/base-interface"
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

// SystemCallCommandRunOptionUnmarshal 系统调用选项反序列化
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
	// Network 监听模式
	Network string
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

// DbPluginInfo 插件信息
type DbPluginInfo struct {
	// Id 主键, 使用sha512
	Id string `json:"id,omitempty" gorm:"primary_key"`
	// Author 作者名称
	Author string
	// Name 名称
	Name string `json:"name,omitempty"`
	// ShortDesc 短描述
	ShortDesc string `json:"longDesc,omitempty"`
	// Desc 描述
	Desc string `json:"desc,omitempty"`
	// Icon 图标
	Icon string `json:"icon,omitempty"`
	// CreateTime 创建时间
	CreateTime time.Time
	// Type 插件类别
	Type rpcinterfaces.PluginType
	// Path 路径
	Path string `json:"-"`
	// InstallTime 安装时间
	InstallTime time.Time
	// Enable 是否启用
	Enable bool
}

// PluginStatus 插件状态
type PluginStatus uint8

const (
	// PluginStatusNoRunning 没有启动
	PluginStatusNoRunning PluginStatus = iota
	// PluginStatusOk 启动成功
	PluginStatusOk
	// PluginStatusRebooting 重启中
	PluginStatusRebooting
	// PluginStatusErr 启动失败
	PluginStatusErr
)

// PluginStatusInfo 插件状态信息
type PluginStatusInfo struct {
	// DbPluginInfo 插件信息
	*DbPluginInfo
	// Status 状态
	Status PluginStatus
	// Msg 消息
	Msg string
	// StartTime 启动时间
	StartTime time.Time
	// StopTime 停止时间
	StopTime time.Time
}

// KeypairGeneratorInfo 密钥对生成信息
type KeypairGeneratorInfo struct {
	// Type 要生成的类型: plugin、client
	Type string
	// Author 作者名称，当 Type 为plugin时生效
	Author string
	// Name 插件名称, 当前 Type 为plugin时生效
	Name string
}

// ServerInfo 服务器信息
type ServerInfo struct {
	// Id id
	Id string
	// IP ip地址
	IP net.IP
	// Port 端口
	Port int
	// Alias 别名
	Alias []string
	// CertPem 证书PEM
	CertPem string
	// CertPrivateKeyPem 私钥PEM
	CertPrivateKeyPem string
	// JoinTime 加入时间
	JoinTime time.Time
}
