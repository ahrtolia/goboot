package config

import (
	"fmt"
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

	sub := v.Sub("config_center.nacos")
	if sub == nil {
		return fmt.Errorf("missing nacos config block in viper")
	}

	serverConfig := constant.ServerConfig{
		IpAddr:      sub.GetString("host"),
		Port:        uint64(sub.GetUint("port")),
		ContextPath: "/nacos",
	}

	fmt.Println("[Nacos] parsed server config:", serverConfig)

	clientConfig := constant.ClientConfig{
		NamespaceId:         sub.GetString("namespace"),
		TimeoutMs:           5000,
		NotLoadCacheAtStart: true,
		LogDir:              sub.GetString("log_dir"),
		CacheDir:            sub.GetString("cache_dir"),
	}

	client, err := clients.NewConfigClient(
		vo.NacosClientParam{
			ClientConfig:  &clientConfig,
			ServerConfigs: []constant.ServerConfig{serverConfig},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to create nacos client: %w", err)
	}

	n.client = client

	// 添加到 Init() 的最后
	dataID := sub.GetString("data_id")
	group := sub.GetString("group")

	fmt.Println("[Nacos] Initializing with dataID:", dataID, "group:", group, "namespace:", v.GetString("namespace"))

	content, err := n.client.GetConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
	})
	if err != nil {
		return fmt.Errorf("failed to get config from nacos: %w", err)
	}

	if err = mergeConfig(v, content); err != nil {
		return fmt.Errorf("failed to merge nacos config: %w", err)
	}

	return nil
}

func (n *nacosAdapter) Watch(v *viper.Viper, onChange func()) error {
	sub := v.Sub("config_center.nacos")
	if sub == nil {
		return fmt.Errorf("missing config_center.nacos block")
	}

	dataID := sub.GetString("data_id")
	group := sub.GetString("group")

	return n.client.ListenConfig(vo.ConfigParam{
		DataId: dataID,
		Group:  group,
		OnChange: func(_, _, _, data string) {
			fmt.Println("[Nacos] Config changed, reloading...")

			temp := viper.New()
			temp.SetConfigType("yaml")
			if err := temp.ReadConfig(strings.NewReader(data)); err != nil {
				fmt.Println("[Nacos] Failed to parse config:", err)
				return
			}

			// 如果你有 cm.ReloadConfig() 可调用它，否则：
			for k := range v.AllSettings() {
				v.Set(k, nil) // 清空旧配置
			}
			for k, val := range temp.AllSettings() {
				v.Set(k, val) // 设置新配置
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
