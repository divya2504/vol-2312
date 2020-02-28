package commands

import (
	"github.com/opencord/voltha-lib-go/v3/pkg/config"
	"github.com/opencord/voltha-lib-go/v3/pkg/db/kvstore"
	"strconv"
)

const (
	kvStoreType           = "etcd"
	defaultKVStoreType    = kvStoreType
	defaultKVStoreTimeout = 1 //in seconds
	defaultKVStoreHost    = "127.0.0.1"
	defaultKVStorePort    = 2379 // Consul = 8500; Etcd = 2379
)

func NewDefaultKVStore() *config.KvStore {
	return &config.KvStore{ // Default values
		KVStoreType:    defaultKVStoreType,
		KVStoreTimeout: defaultKVStoreTimeout,
		KVStoreHost:    defaultKVStoreHost,
		KVStorePort:    defaultKVStorePort,
	}
}

func setConfigManager(kv *config.KvStore) (*config.ConfigManager, error) {
	addr := kv.KVStoreHost + ":" + strconv.Itoa(kv.KVStorePort)
	if kv.KVStoreType == "etcd" {
		client, err := kvstore.NewEtcdClient(addr, kv.KVStoreTimeout)
		if err != nil {
			return nil, err
		}
		kv.KvClient = client
	}
	cm := config.NewConfigManager(kv.KvClient, kv.KVStoreType, kv.KVStoreHost, kv.KVStorePort, kv.KVStoreTimeout)
	return cm, nil
}
