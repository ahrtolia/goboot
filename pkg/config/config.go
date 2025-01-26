package config

import (
	"bytes"
	"fmt"
	"log"
	"sync"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
)

var (
	mu         sync.Mutex
	once       sync.Once
	v          *viper.Viper // 全局唯一的 viper 实例
	nacosReady bool         // 标记 Nacos 配置是否已成功加载
)

// New 初始化 Viper 和加载 Nacos 配置
func New(path string) (*viper.Viper, error) {
	once.Do(func() {
		// 创建一个新的 viper 实例
		v = viper.New()
		v.AddConfigPath(".")
		v.SetConfigFile(path)

		// 读取配置文件
		err := v.ReadInConfig()
		if err != nil {
			fmt.Printf("无法读取配置文件: %s，使用默认配置\n", err.Error())
		} else {
			fmt.Printf("成功使用配置文件 -> %s\n", v.ConfigFileUsed())
		}

		// 初始化 Nacos 配置客户端
		configClient, nacosErr := initNacosClient()
		if nacosErr != nil {
			log.Printf("Nacos 客户端初始化失败: %v，使用配置文件中的配置\n", nacosErr)
			return
		}

		// 从 Nacos 加载配置
		nacosErr = loadNacosConfig(configClient)
		if nacosErr != nil {
			log.Printf("从 Nacos 中加载配置失败: %v，使用配置文件中的配置\n", nacosErr)
		} else {
			nacosReady = true
			log.Println("从 Nacos 中加载配置成功")
		}
	})

	return v, nil
}

// 初始化 Nacos 配置客户端
func initNacosClient() (config_client.IConfigClient, error) {
	// 从配置文件中读取 Nacos 连接信息
	serverIP := v.GetString("nacos.serverIP")
	if serverIP == "" {
		serverIP = "127.0.0.1"
	}
	serverPort := v.GetInt("nacos.serverPort")
	if serverPort == 0 {
		serverPort = 8848
	}
	namespaceID := v.GetString("nacos.namespace")
	username := v.GetString("nacos.username")
	password := v.GetString("nacos.password")

	// 检查必要的配置信息是否存在
	if namespaceID == "" {
		return nil, fmt.Errorf("nacos.namespace 配置缺失")
	}
	//if username == "" || password == "" {
	//	return nil, fmt.Errorf("nacos.username 或 nacos.password 配置缺失")
	//}

	// Nacos服务器配置
	serverConfigs := []constant.ServerConfig{
		{
			ContextPath: "/nacos",
			IpAddr:      serverIP,
			Port:        uint64(serverPort),
		},
	}

	// 客户端配置
	clientConfig := &constant.ClientConfig{
		NamespaceId: namespaceID,
		TimeoutMs:   5000,
		LogDir:      "./nacos/log",
		CacheDir:    "./nacos/cache",
		LogLevel:    "debug",
		Username:    username,
		Password:    password,
	}

	// 创建 Nacos 配置客户端
	configClient, err := clients.NewConfigClient(vo.NacosClientParam{
		ServerConfigs: serverConfigs,
		ClientConfig:  clientConfig,
	})
	if err != nil {
		log.Printf("创建 Nacos 配置客户端失败: %v", err)
		return nil, err
	}

	log.Println("Nacos 配置客户端初始化成功")
	return configClient, nil
}

// 从 Nacos 中加载配置
func loadNacosConfig(configClient config_client.IConfigClient) error {
	// 从配置文件中读取 AppName
	appName := v.GetString("app.name")
	if appName == "" {
		return fmt.Errorf("app.name 配置缺失")
	}

	// 从 Nacos 获取初始配置
	content, err := configClient.GetConfig(vo.ConfigParam{
		DataId: appName,
		Group:  "DEFAULT_GROUP", // 根据需要修改分组名称
	})
	if err != nil {
		return fmt.Errorf("从 Nacos 获取配置失败: %v", err)
	}

	// 将 Nacos 配置加载到 Viper
	mu.Lock()
	defer mu.Unlock()

	if err = v.MergeConfigMap(parseNacosConfig(content)); err != nil {
		return fmt.Errorf("合并 Nacos 配置失败: %v", err)
	}

	// 监听 Nacos 的配置变化
	err = configClient.ListenConfig(vo.ConfigParam{
		DataId: appName,
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			mu.Lock()
			defer mu.Unlock()

			log.Printf("Nacos 配置更新: %s = %s", dataId, data)
			if err = v.MergeConfigMap(parseNacosConfig(data)); err != nil {
				log.Printf("更新内存配置失败: %v", err)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("监听 Nacos 配置失败: %v", err)
	}

	return nil
}

// 解析 Nacos 的配置 (假设 Nacos 配置为 JSON 格式)
// parseNacosConfig 接收 Nacos 配置内容（作为字符串），并将本地 YAML 文件与 Nacos 配置整合，返回合并后的配置 map。
func parseNacosConfig(nacosContent string) map[string]interface{} {
	// 创建主配置存储
	finalConfig := make(map[string]interface{})

	// 使用 viper 处理本地配置文件
	localViper := viper.New()
	localViper.SetConfigName("config") // 假设本地配置文件名为 config.yaml
	localViper.SetConfigType("yaml")
	localViper.AddConfigPath(".") // 本地配置文件的路径（当前目录）

	// 读取本地配置文件
	err := localViper.ReadInConfig()
	if err != nil {
		fmt.Printf("读取本地配置文件失败: %v\n", err)
		return nil
	}

	// 将本地配置存入 finalConfig
	err = localViper.Unmarshal(&finalConfig)
	if err != nil {
		fmt.Printf("解析本地配置失败: %v\n", err)
		return nil
	}

	// 使用 viper 处理传入的 Nacos 配置内容
	nacosViper := viper.New()
	nacosViper.SetConfigType("yaml")

	// 读取 Nacos 配置内容
	err = nacosViper.ReadConfig(bytes.NewBuffer([]byte(nacosContent)))
	if err != nil {
		fmt.Printf("读取 Nacos 配置失败: %v\n", err)
		return nil
	}

	// 将 Nacos 配置存入一个临时 map
	nacosConfig := make(map[string]interface{})
	err = nacosViper.Unmarshal(&nacosConfig)
	if err != nil {
		fmt.Printf("解析 Nacos 配置失败: %v\n", err)
		return nil
	}

	// 合并本地配置和 Nacos 配置（优先使用 Nacos 配置覆盖本地配置）
	mergeMaps(finalConfig, nacosConfig)

	return finalConfig
}

// mergeMaps 合并两个 map，nacosConfig 中的值会覆盖本地 finalConfig 中的值
func mergeMaps(local map[string]interface{}, nacos map[string]interface{}) {
	for key, nacosValue := range nacos {
		localValue, exists := local[key]
		if !exists {
			// 本地配置不存在该 key，直接使用 Nacos 配置
			local[key] = nacosValue
			continue
		}

		// 如果本地和 Nacos 的值都是 map，则递归合并
		localMap, localIsMap := localValue.(map[string]interface{})
		nacosMap, nacosIsMap := nacosValue.(map[string]interface{})
		if localIsMap && nacosIsMap {
			mergeMaps(localMap, nacosMap)
		} else {
			// 否则直接覆盖本地值
			local[key] = nacosValue
		}
	}
}

// ProviderSet 是 Wire 的 provider 集合
//var ProviderSet = wire.NewSet(New)
