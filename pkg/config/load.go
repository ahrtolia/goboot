package config

import (
	"log"
	"sync"
)

// ChangeCallback 回调函数定义，用于在配置变更时通知各模块更新
type ChangeCallback func()

// Updater 用于管理所有配置变更的回调
type Updater struct {
	callbacks []ChangeCallback
	mu        sync.Mutex
}

// NewConfigUpdater 创建一个新的 Updater 实例
func NewConfigUpdater() *Updater {
	return &Updater{
		callbacks: make([]ChangeCallback, 0),
	}
}

// Register 注册一个配置变更回调
func (cu *Updater) Register(callback ChangeCallback) {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	cu.callbacks = append(cu.callbacks, callback)
}

// Notify 通知所有注册的回调执行
func (cu *Updater) Notify() {
	cu.mu.Lock()
	defer cu.mu.Unlock()

	for _, callback := range cu.callbacks {
		go func(cb ChangeCallback) {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("配置变更回调失败: %v", err)
				}
			}()
			cb()
		}(callback)
	}
}
