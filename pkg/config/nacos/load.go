package nacos

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"reflect"
	"sync"

	"github.com/google/wire"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"github.com/spf13/viper"
)

const (
	defaultCachePath = "./nacos-merged.yaml"
)

var (
	confMu sync.Mutex
	mu     sync.Mutex
)

// NewNacosConfigClient 创建Nacos配置客户端
func NewNacosConfigClient(configPath string) (config_client.IConfigClient, error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取本地配置文件失败: %w", err)
	}

	serverIP := v.GetString("nacos.serverIP")
	if serverIP == "" {
		serverIP = "127.0.0.1"
	}

	serverPort := v.GetInt("nacos.serverPort")
	if serverPort == 0 {
		serverPort = 8848
	}

	clientConfig := &constant.ClientConfig{
		NamespaceId:         v.GetString("nacos.namespace"),
		TimeoutMs:           5000,
		NotLoadCacheAtStart: false,
		LogDir:              "./nacos/log",
		CacheDir:            "./nacos/cache",
		LogLevel:            "debug",
		Username:            v.GetString("nacos.username"),
		Password:            v.GetString("nacos.password"),
	}

	serverConfigs := []constant.ServerConfig{
		{
			IpAddr:      serverIP,
			Port:        uint64(serverPort),
			ContextPath: "/nacos",
		},
	}

	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  clientConfig,
		ServerConfigs: serverConfigs,
	})

	if err != nil {
		return nil, fmt.Errorf("创建Nacos客户端失败: %w", err)
	}

	return client, nil
}

// MergeNacosConfig 合并配置并返回最终Viper实例
func MergeNacosConfig(configPath string, client config_client.IConfigClient) (*viper.Viper, error) {
	// 1. 读取本地配置
	vLocal, err := readLocalConfig(configPath)
	if err != nil {
		return nil, err
	}

	// 2. 合并配置
	merged, err := mergeAllConfigs(vLocal, client)
	if err != nil {
		return nil, err
	}

	// 3. 监听远程配置变更
	if err := watchRemoteConfig(client, vLocal.GetString("app.name"), defaultCachePath, merged); err != nil {
		log.Printf("[WARN] 配置监听初始化失败: %v", err)
	}

	// 4. 创建最终Viper实例
	return createFinalViper(merged)
}

func readLocalConfig(path string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("本地配置读取失败: %w", err)
	}
	return v, nil
}

func mergeAllConfigs(vLocal *viper.Viper, client config_client.IConfigClient) (*viper.Viper, error) {
	// 1. 读取缓存配置
	vCache := readCacheConfig()

	// 2. 合并缓存 + 本地配置
	merged := viper.New()
	mergeConfig(merged, vCache)
	mergeConfig(merged, vLocal)

	// 3. 合并远程配置
	if remoteConfig, err := getRemoteConfig(client, vLocal.GetString("app.name")); err == nil {
		if vRemote, err := parseConfigString(remoteConfig); err == nil {
			mergeConfig(merged, vRemote)
			log.Println("[INFO] 远程配置合并成功")
		}
	}

	// 4. 持久化合并结果
	if err := writeMergedConfig(merged); err != nil {
		return nil, err
	}

	return merged, nil
}

func readCacheConfig() *viper.Viper {
	v := viper.New()
	if !fileExists(defaultCachePath) {
		return v
	}

	v.SetConfigFile(defaultCachePath)
	if err := v.ReadInConfig(); err != nil {
		log.Printf("[WARN] 缓存文件读取失败: %v", err)
	}
	return v
}

func getRemoteConfig(client config_client.IConfigClient, appName string) (string, error) {
	content, err := client.GetConfig(vo.ConfigParam{
		DataId: appName,
		Group:  "DEFAULT_GROUP",
	})
	if err != nil {
		log.Printf("[WARN] 远程配置获取失败: %v", err)
		return "", err
	}
	return content, nil
}

func parseConfigString(content string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(bytes.NewBufferString(content)); err != nil {
		return nil, fmt.Errorf("配置解析失败: %w", err)
	}
	return v, nil
}

// watchRemoteConfig 中的OnChange回调函数需要改进
// watchRemoteConfig 修改后的配置变更处理
func watchRemoteConfig(client config_client.IConfigClient, appName string, configPath string, merged *viper.Viper) error {
	return client.ListenConfig(vo.ConfigParam{
		DataId: appName,
		Group:  "DEFAULT_GROUP",
		OnChange: func(namespace, group, dataId, data string) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[PANIC] 配置变更处理异常: %v", r)
				}
			}()

			confMu.Lock()
			defer confMu.Unlock()

			log.Printf("[INFO] 检测到配置变更: %s", dataId)

			// 1. 读取最新本地配置
			vLocal, err := readLocalConfig(configPath)
			if err != nil {
				log.Printf("[ERROR] 本地配置重载失败: %v", err)
				return
			}

			// 2. 创建全新合并实例
			newMerged := viper.New()

			// 3. 按照正确优先级合并：
			//    a. 先合并缓存（最低优先级）
			mergeConfig(newMerged, readCacheConfig())
			//    b. 合并本地配置（覆盖缓存）
			mergeConfig(newMerged, vLocal)
			//    c. 合并最新远程配置（最高优先级）
			if vRemote, err := parseConfigString(data); err == nil {
				mergeConfig(newMerged, vRemote)
			}

			// 4. 获取当前合并结果和最新合并结果
			currentConfig := merged.AllSettings()
			newConfig := newMerged.AllSettings()

			// 5. 深度比较配置差异
			if reflect.DeepEqual(currentConfig, newConfig) {
				log.Println("[INFO] 配置内容无实质性变化")
				return
			}

			// 6. 更新主配置并持久化
			merged.MergeConfigMap(newConfig)
			if err := writeMergedConfig(merged); err != nil {
				log.Printf("[ERROR] 配置持久化失败: %v", err)
				return
			}

			log.Printf("[INFO] 检测到有效配置变更，已更新 %d 处配置", countChanges(currentConfig, newConfig))
		},
	})
}

// countChanges 辅助函数：计算配置差异数量
func countChanges(a, b map[string]interface{}) int {
	changes := 0
	for k, v := range b {
		if !reflect.DeepEqual(a[k], v) {
			changes++
		}
	}
	return changes
}

// mergeConfig 优化后的合并方法
func mergeConfig(target *viper.Viper, source *viper.Viper) {
	if source == nil {
		return
	}

	// 使用底层合并逻辑确保正确的优先级
	if err := target.MergeConfigMap(source.AllSettings()); err != nil {
		log.Printf("[WARN] 配置合并失败: %v", err)
	}
}

// writeMergedConfig 增加文件同步操作
func writeMergedConfig(v *viper.Viper) error {

	// recover panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[PANIC] 配置持久化处理异常: %v", r)
		}
	}()

	mu.Lock()
	defer mu.Unlock()

	v.SetConfigType("yaml")
	file, err := os.OpenFile(defaultCachePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("文件打开失败: %w", err)
	}
	defer file.Close()

	if err := v.WriteConfigAs(defaultCachePath); err != nil {
		return fmt.Errorf("配置写入失败: %w", err)
	}

	// 确保文件内容同步到磁盘
	if err := file.Sync(); err != nil {
		return fmt.Errorf("文件同步失败: %w", err)
	}

	return nil
}

// copyMap 深度拷贝配置map
func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		if vm, ok := v.(map[string]interface{}); ok {
			result[k] = copyMap(vm)
		} else {
			result[k] = v
		}
	}
	return result
}

func createFinalViper(merged *viper.Viper) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigFile(defaultCachePath)
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("最终配置读取失败: %w", err)
	}
	v.WatchConfig()
	return v, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

var ProviderSet = wire.NewSet(NewNacosConfigClient, MergeNacosConfig)
