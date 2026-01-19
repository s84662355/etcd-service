package etcdService

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.etcd.io/etcd/client/v3"
)

func (r *registry) Register(path, value string) error {
	r.lock.Lock()
	if r.isClose {
		r.lock.Unlock()
		return fmt.Errorf("is close")
	}

	if _, ok := r.registrations[path]; ok {
		r.lock.Unlock()
		return fmt.Errorf("已经注册了")
	}
	n, err := newNodeRegister(
		r.client,
		path,
		value,
		r.ttl,
	)
	if err != nil {
		return err
	}
	r.registrations[path] = n
	r.lock.Unlock()

	return nil
}

func (r *registry) Unregister(path string) error {
	r.lock.Lock()
	if r.isClose {
		r.lock.Unlock()
		return fmt.Errorf("is close")
	}
	if n, ok := r.registrations[path]; !ok {
		r.lock.Unlock()
		return fmt.Errorf("没有注册了")
	} else {
		delete(r.registrations, path)
		r.lock.Unlock()
		n.Close()
	}

	return nil
}

type nodeRegister struct {
	client *clientv3.Client
	value  string
	ctx    context.Context
	cancel context.CancelFunc
	ttl    time.Duration
	path   string
	done   chan struct{}
	onceDo sync.Once
}

func newNodeRegister(
	client *clientv3.Client,
	path string,
	value string,
	ttl time.Duration,
) (*nodeRegister, error) {
	n := &nodeRegister{
		client: client,
		value:  value,
		path:   path,
		ttl:    ttl,
	}
	n.ctx, n.cancel = context.WithCancel(context.Background())
	n.done = make(chan struct{})

	go func() {
		defer close(n.done)
		for {
			select {
			case <-n.ctx.Done():
				return
			default:
				n.register()
				func() {
					ticker := time.NewTicker(1 * time.Second)
					defer ticker.Stop() // 确保释放资源

					select {
					case <-n.ctx.Done():
						return
					case <-ticker.C:
					}
				}()

			}
		}
	}()
	return n, nil
}

func (n *nodeRegister) register() {
	leaseResp, err := n.client.Grant(n.ctx, int64(n.ttl.Seconds()))
	if err != nil {
		return
	}
	leaseID := leaseResp.ID
	defer n.client.Revoke(context.Background(), leaseID)

	keepAliveCh, err := n.client.KeepAlive(n.ctx, leaseID)
	if err != nil {
		return
	}

	putResp, err := n.client.Put(n.ctx, n.path, n.value, clientv3.WithLease(leaseID))
	if err != nil {
		n.client.Revoke(context.Background(), leaseID)
		return
	}

	originalCreateRev := putResp.Header.Revision

	defer func() {
		n.client.Txn(context.Background()).
			If(clientv3.Compare(clientv3.CreateRevision(n.path), "=", originalCreateRev)).
			Then(clientv3.OpDelete(n.path)).
			Commit()
	}()

	watchCh := n.client.Watch(n.ctx, n.path)

	for {
		select {
		case re, ok := <-keepAliveCh:

			if re == nil || !ok {
				return
			}
		case watchResp, ok := <-watchCh:
			if !ok || watchResp.Canceled {
				return
			}

			for _, event := range watchResp.Events {
				if event.Kv.CreateRevision != originalCreateRev {
					return
				}
				switch event.Type {
				case clientv3.EventTypePut:
				case clientv3.EventTypeDelete:
					return
				}
			}

		case <-n.ctx.Done():

			return
		}
	}
}

func (n *nodeRegister) Done() <-chan struct{} {
	return n.done
}

func (n *nodeRegister) Close() {
	n.onceDo.Do(func() {
		n.cancel()

		for range n.done {
		}
	})
}
