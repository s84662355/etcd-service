package etcdService

import (
	"context"
	"fmt"
	"maps"
	"time"

	"go.etcd.io/etcd/client/v3"
)

func (r *registry) AddWatch(prefix string, fn watchfn) error {
	r.lock.Lock()
	if r.isClose {
		r.lock.Unlock()
		return fmt.Errorf("is close")
	}
	if _, ok := r.etcdWatchs[prefix]; ok {
		r.lock.Unlock()
		return fmt.Errorf("已经watch了")
	}
	w := newEtcdWatch(
		r.client, prefix, fn,
	)
	r.etcdWatchs[prefix] = w
	r.lock.Unlock()
	return nil
}

func (r *registry) RemoveWatch(prefix string) error {
	r.lock.Lock()
	if r.isClose {
		r.lock.Unlock()
		return fmt.Errorf("is close")
	}
	if w, ok := r.etcdWatchs[prefix]; !ok {
		r.lock.Unlock()
		return fmt.Errorf("没有watch")
	} else {
		delete(r.etcdWatchs, prefix)
		r.lock.Unlock()
		w.Close()
	}
	return nil
}

type etcdWatch struct {
	client *clientv3.Client
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}
}

func newEtcdWatch(
	client *clientv3.Client,
	prefix string, fn watchfn,
) *etcdWatch {
	w := &etcdWatch{
		client: client,
		done:   make(chan struct{}),
	}
	w.ctx, w.cancel = context.WithCancel(context.Background())

	go func() {
		defer close(w.done)
		w.init(prefix, fn)
	}()

	return w
}

func (w *etcdWatch) Close() {
	w.cancel()
	for range w.done {
	}
}

type watchfn func(map[string]string)

func (w *etcdWatch) init(prefix string, fn watchfn) {
	for {
		select {
		case <-w.ctx.Done():
			return
		default:
			func() {
				resp, err := w.client.Get(w.ctx, prefix, clientv3.WithPrefix())
				if err != nil {
					return
				}

				m := make(map[string]string, len(resp.Kvs))

				for _, kv := range resp.Kvs {
					m[string(kv.Key)] = string(kv.Value)
				}
				fn(maps.Clone(m))

				watchChan := w.client.Watch(w.ctx, prefix, clientv3.WithPrefix())
				ticker := time.NewTicker(5 * time.Second)
				defer ticker.Stop() // 确保释放资源
				for {
					select {
					case <-ticker.C:
						resp, err := w.client.Get(w.ctx, prefix, clientv3.WithPrefix())
						if err != nil {
							return
						}

						m := make(map[string]string, len(resp.Kvs))

						for _, kv := range resp.Kvs {
							m[string(kv.Key)] = string(kv.Value)
						}
						fn(maps.Clone(m))

					case <-w.ctx.Done():
						return
					case watchResp, ok := <-watchChan:
						if !ok || watchResp.Canceled {
							return
						}

						for _, event := range watchResp.Events {
							key := string(event.Kv.Key)
							switch event.Type {
							case clientv3.EventTypePut:
								if event.IsCreate() {
								} else {
								}

								m[key] = string(event.Kv.Value)

							case clientv3.EventTypeDelete:
								delete(m, key)

							}
						}
						fn(maps.Clone(m))

					}
				}
			}()
		}
		func() {
			ctx, cancel := context.WithTimeout(w.ctx, 1*time.Second)
			defer cancel()
			select {
			case <-ctx.Done():

			case <-w.ctx.Done():
				return
			}
		}()

	}
}
