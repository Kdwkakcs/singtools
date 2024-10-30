package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sagernet/sing/common/json"
	bolt "go.etcd.io/bbolt"
)

// StoreProxy 存储单个代理数据
func (manager *CacheManager) StoreProxy(key string, data ProxyData) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(data); err != nil {
		return fmt.Errorf("encode proxy data: %w", err)
	}

	return manager.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		return b.Put([]byte(key), buf.Bytes())
	})
}

// GetProxy 获取单个代理数据
func (manager *CacheManager) GetProxy(key string) (ProxyData, error) {
	var data ProxyData
	err := manager.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketProxies)
		}
		v := b.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("proxy not found")
		}
		return gob.NewDecoder(bytes.NewReader(v)).Decode(&data)
	})
	if err != nil {
		return ProxyData{}, err
	}
	return data, nil
}

// GetAllProxies 获取所有代理数据
func (manager *CacheManager) GetAllProxies() (map[string]ProxyData, error) {
	proxies := make(map[string]ProxyData)
	err := manager.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketProxies)
		}
		return b.ForEach(func(k, v []byte) error {
			var data ProxyData
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&data); err != nil {
				return err
			}
			proxies[string(k)] = data
			return nil
		})
	})
	return proxies, err
}

// mergeNewProxies 合并新的代理数据
func (manager *CacheManager) mergeNewProxies(newProxies []interface{}) error {
	existingProxies, err := manager.GetAllProxies()
	if err != nil && !errors.Is(err, bolt.ErrBucketNotFound) {
		return fmt.Errorf("failed to get existing proxies: %w", err)
	}

	return manager.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketProxies)
		}

		for _, proxy := range newProxies {
			proxyMap, ok := proxy.(map[string]interface{})
			if !ok {
				continue
			}

			proxyBytes, err := json.Marshal(proxyMap)
			if err != nil {
				log.Printf("Failed to marshal proxy: %v", err)
				continue
			}
			key := fmt.Sprintf("proxy_%x", sha256.Sum256(proxyBytes))[:16]

			if existingData, exists := existingProxies[key]; exists {
				// 更新现有代理
				existingData.ProxyConfig = proxyMap
				existingData.UpdatedAt = time.Now()
				var buf bytes.Buffer
				if err := gob.NewEncoder(&buf).Encode(existingData); err != nil {
					return err
				}
				if err := b.Put([]byte(key), buf.Bytes()); err != nil {
					return err
				}
			} else {
				// 创建新代理
				newData := ProxyData{
					ProxyConfig:  proxyMap,
					UpdatedAt:    time.Now(),
					LastChecked:  time.Time{},
					IsAccessible: false,
					Latency:      0,
				}
				var buf bytes.Buffer
				if err := gob.NewEncoder(&buf).Encode(newData); err != nil {
					return err
				}
				if err := b.Put([]byte(key), buf.Bytes()); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// CleanupOldProxies 清理过期的代理数据
func (manager *CacheManager) CleanupOldProxies(maxAge time.Duration) error {
	now := time.Now()
	return manager.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketProxies)
		}

		var keysToDelete [][]byte
		err := b.ForEach(func(k, v []byte) error {
			var data ProxyData
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&data); err != nil {
				return err
			}
			if now.Sub(data.UpdatedAt) > maxAge {
				keysToDelete = append(keysToDelete, k)
			}
			return nil
		})
		if err != nil {
			return err
		}

		for _, k := range keysToDelete {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAccessibleProxies 获取所有可访问的代理
func (manager *CacheManager) GetAccessibleProxies() ([]ProxyData, error) {
	var accessibleProxies []ProxyData
	err := manager.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketProxies)
		}
		return b.ForEach(func(k, v []byte) error {
			var data ProxyData
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&data); err != nil {
				return err
			}
			if data.IsAccessible {
				accessibleProxies = append(accessibleProxies, data)
			}
			return nil
		})
	})
	return accessibleProxies, err
}

// GetRecentProxies 获取最近更新的代理
func (manager *CacheManager) GetRecentProxies(since time.Duration) ([]ProxyData, error) {
	threshold := time.Now().Add(-since)
	var recentProxies []ProxyData

	err := manager.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketProxies))
		if b == nil {
			return fmt.Errorf("bucket %s not found", bucketProxies)
		}
		return b.ForEach(func(k, v []byte) error {
			var data ProxyData
			if err := gob.NewDecoder(bytes.NewReader(v)).Decode(&data); err != nil {
				return err
			}
			if data.UpdatedAt.After(threshold) {
				recentProxies = append(recentProxies, data)
			}
			return nil
		})
	})
	return recentProxies, err
}
