package etcdService

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// 测试 New + Register + AddWatch（核心服务发现流程）
func TestRegistryAddWatchServiceDiscovery(t *testing.T) {
	// 1. 严格按照你指定的参数创建Registry实例
	rrr, err := New(
		[]string{"localhost:2379"}, // etcd 端点地址
		"", "",                     // 两个空字符串参数
		5*time.Second, // 超时时间1
		5*time.Second, // 超时时间2
	)

	assert.NoError(t, err, "New方法创建Registry实例失败")
	defer rrr.Close()

	// 2. 定义测试服务信息
	serviceName := "payment-service"
	serviceAddr := "127.0.0.1:8888"

	// 3. 先注册一个服务（为发现做准备）
	err = rrr.Register(serviceName, serviceAddr)
	assert.NoError(t, err, "服务注册失败")

	// 调用AddWatch，监听指定服务的变化并把结果写入通道
	err = rrr.AddWatch(serviceName, func(addrs map[string]string) {
		fmt.Println(addrs)
	})
	assert.NoError(t, err, "AddWatch方法调用失败")

	for i := 0; i < 10; i++ {
		rrr.Register(serviceName, serviceAddr)
		<-time.After(2 * time.Second)
		rrr.Unregister(serviceName)
		<-time.After(2 * time.Second)
	}
}
