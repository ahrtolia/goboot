// pkg/config/nacos.go
package config

import (
	"fmt"
	"github.com/google/wire"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
)

type nacosAdapter struct {
	client config_client.IConfigClient
}

func NewNacosAdapter() ConfigCenter {
	return &nacosAdapter{}
}

func (n *nacosAdapter) Name() string {
	return "nacos"
}

func (n *nacosAdapter) Init(v *viper.Viper) error {
	serverConfig := constant.ServerConfig{
		IpAddr:      v.GetString("host"),
		Port:        uint64(v.GetUint("port")),
		ContextPath: "/nacos",
	}

	clientConfig := constant.ClientConfig{
		NamespaceId:         v.GetString("namespace"),
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              v.GetString("log_dir"),
		CacheDir:            v.GetString("cache_dir"),
	}

	client, err := clients.CreateConfigClient(map[string]interface{}{
		"serverConfigs": []constant.ServerConfig{serverConfig},
		"clientConfig":  clientConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create nacos client: %w", err)
	}

	n.client = client
	return nil
}

func (n *nacosAdapter) Watch(v *viper.Viper, onChange func()) error {
	dataID := v.GetString("data_id")
	group := v.GetString("group")

	content, err := n.client.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
	})
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}

	if err := mergeConfig(v, content); err != nil {
		return err
	}

	return n.client.ListenConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
		OnChange: func(_, _, _, data string) {
			if err := mergeConfig(v, data); err != nil {
				return
			}
			onChange()
		},
	})
}

func (n *nacosAdapter) Close() {
	// Nacos client doesn't need explicit close
}

func mergeConfig(v *viper.Viper, content string) error {
	temp := viper.New()
	temp.SetConfigType("yaml")
	if err := temp.ReadConfig(strings.NewReader(content)); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	return v.MergeConfigMap(temp.AllSettings())
}

var NacosProvider = wire.NewSet(
	wire.Value(ConfigCenter(NewNacosAdapter())),
)
