package commands

import (
	"errors"
	"fmt"
	"github.com/opencord/voltha-lib-go/v3/pkg/config"
	"github.com/opencord/voltha-lib-go/v3/pkg/db/kvstore"
	"github.com/opencord/voltha-lib-go/v3/pkg/log"
	"strconv"
)

const (
	EtcdStoreName              = "etcd"
	defaultKVStoreType         = EtcdStoreName
	defaultKVStoreTimeout      = 5 //in seconds
	defaultKVStoreHost         = "127.0.0.1"
	defaultKVStorePort         = 2379 // Consul = 8500; Etcd = 2379
	kvStoreDataPathPrefix      = "/service/voltha"
	defaultKVStoreConfigPrefix = "/config/"
)

type kvStore struct {
	kvClient            kvstore.Client
	KVStoreType         string
	KVStoreHost         string
	KVStorePort         int
	KVStoreTimeout      int
	KVStoreDataPrefix   string
	KVStoreConfigPrefix string
}

func NewKVStore() *kvStore {
	var kvStore = kvStore{ // Default values
		KVStoreType:         defaultKVStoreType,
		KVStoreTimeout:      defaultKVStoreTimeout,
		KVStoreHost:         defaultKVStoreHost,
		KVStorePort:         defaultKVStorePort,
		KVStoreDataPrefix:   kvStoreDataPathPrefix,
		KVStoreConfigPrefix: defaultKVStoreConfigPrefix,
	}
	return &kvStore
}

func newKVClient(storeType string, address string, timeout int) (kvstore.Client, error) {

	switch storeType {
	case "etcd":
		return kvstore.NewEtcdClient(address, timeout)
	}
	return nil, errors.New("unsupported-kv-store")
}

func (kv *kvStore) setKVClient() (*config.ConfigManager, error) {
	addr := kv.KVStoreHost + ":" + strconv.Itoa(kv.KVStorePort)
	client, err := newKVClient(kv.KVStoreType, addr, kv.KVStoreTimeout)
	if err != nil {
		return nil, err
	}
	kv.kvClient = client
	cm := config.NewConfigManager(client, kv.KVStoreType, kv.KVStoreHost, kv.KVStorePort, kv.KVStoreTimeout)
	return cm, nil
}
