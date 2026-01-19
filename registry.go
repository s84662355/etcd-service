package etcdService

import (
	"context"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
)

type Registry interface {
	AddWatch(prefix string, fn watchfn) error
	RemoveWatch(prefix string) error
	Register(path, value string) error
	Unregister(path string) error
	Close()
}

type registry struct {
	client        *clientv3.Client
	lock          sync.RWMutex
	registrations map[string]*nodeRegister
	etcdWatchs    map[string]*etcdWatch
	ctx           context.Context
	cancel        context.CancelFunc
	onceDo        sync.Once
	ttl           time.Duration
	isClose       bool
}

func New(
	Endpoints []string, // etcd 端点地址,
	Username, Password string,
	DefaultTimeout time.Duration, // 默认超时时间
	ttl time.Duration,
) (Registry, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:            Endpoints,
		Username:             Username,
		Password:             Password,
		DialTimeout:          DefaultTimeout,
		DialKeepAliveTime:    10 * time.Second,
		DialKeepAliveTimeout: 30 * time.Second,
		AutoSyncInterval:     30 * time.Second,
		RejectOldCluster:     true,
	})
	if err != nil {
		return nil, err
	}
	r := &registry{
		client:        client,
		registrations: make(map[string]*nodeRegister),
		etcdWatchs:    make(map[string]*etcdWatch),
		ttl:           ttl,
	}
	r.ctx, r.cancel = context.WithCancel(context.Background())
	return r, nil
}

func (r *registry) Close() {
	r.onceDo.Do(func() {
		r.lock.Lock()
		r.isClose = true
		registrations := r.registrations
		etcdWatchs := r.etcdWatchs
		r.etcdWatchs = nil
		r.registrations = nil
		r.lock.Unlock()
		r.cancel()
		waitGroup := sync.WaitGroup{}
		waitGroup.Add(2)
		go func() {
			defer waitGroup.Done()
			for _, v := range registrations {
				v.Close()
			}
		}()
		go func() {
			defer waitGroup.Done()
			for _, v := range etcdWatchs {
				v.Close()
			}
		}()
		waitGroup.Wait()
		r.client.Close()
	})
}

// func main() {
// 	rrr, err := NewRegistry(
// 		[]string{"localhost:2379"}, // etcd 端点地址,
// 		"", "",
// 		5*time.Second, // 默认超时时间
// 		5*time.Second,
// 	)
// 	if err != nil {
// 		panic(err)
// 	}

// 	rrr.Register("/aasd/asd/asdas/dasdas/dsfds", "54353453453453")
// 	rrr.Register("/aasd/asd/asdas/dasdas/dsfds23332", "54fd53453")
// 	// // 1. 连接 etcd
// 	// cli, err := clientv3.New(clientv3.Config{
// 	// 	Endpoints:   []string{"localhost:2379"},
// 	// 	DialTimeout: 5 * time.Second,
// 	// })
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// defer cli.Close()

// 	// nnn, err := newNodeRegister(cli, "/aasd/asd/asdas/dasdas/dsfds", "54353453453453", 3*time.Second)
// 	// if err != nil {
// 	// 	panic(err)
// 	// }
// 	// defer nnn.Close()

// 	time.Sleep(30 * time.Second)
// 	rrr.Close()
// 	///nnn.Close()

// 	fmt.Println(111111111111111)
// 	time.Sleep(10 * time.Second)
// }
