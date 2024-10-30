package cache

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

// ProxyData 存储代理数据及其状态
type ProxyData struct {
	ProxyConfig  map[string]interface{} // 原始代理配置
	LastChecked  time.Time              // 最后检查时间
	IsAccessible bool                   // 是否可访问
	Latency      time.Duration          // 延迟时间
	UpdatedAt    time.Time              // 更新时间
}

// UpdateProxyStatus 更新代理状态
func (manager *CacheManager) UpdateProxyStatus(key string, isAccessible bool, latency time.Duration) error {
	return manager.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		v := b.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("proxy not found")
		}

		var data ProxyData
		if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&data); err != nil {
			return err
		}

		data.IsAccessible = isAccessible
		data.Latency = latency
		data.LastChecked = time.Now()

		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(data); err != nil {
			return err
		}

		return b.Put([]byte(key), buf.Bytes())
	})
}

// StoreProxies 存储所有代理数据
func (manager *CacheManager) StoreProxies(proxies []interface{}) error {
	return manager.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		for i, proxy := range proxies {
			key := fmt.Sprintf("proxy_%d", i)
			data := ProxyData{
				ProxyConfig:  proxy.(map[string]interface{}),
				UpdatedAt:    time.Now(),
				LastChecked:  time.Now(),
				IsAccessible: false, // 初始状态设为 false，需要通过检查更新
			}

			var buf bytes.Buffer
			if err := gob.NewEncoder(&buf).Encode(data); err != nil {
				return err
			}

			if err := b.Put([]byte(key), buf.Bytes()); err != nil {
				return err
			}
		}
		return nil
	})
}
