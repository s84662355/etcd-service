package etcdService

import (
	"log"
	"testing"
	"time"
)

// 测试 New + Register + AddWatch（核心服务发现流程）
func TestRegistryAddWatchServiceDiscovery(t *testing.T) {
	// 1. 初始化 Registry 实例（严格匹配测试文件的参数）
	rrr, err := New(
		[]string{"localhost:2379"}, // etcd 端点地址
		"", "",                     // 测试文件中留空的两个字符串参数
		5*time.Second, // 第一个超时时间（如连接超时）
		5*time.Second, // 第二个超时时间（如操作超时）
	)
	if err != nil {
		log.Fatalf("创建 Registry 实例失败: %v", err)
	}
	defer rrr.Close() // 确保退出时关闭资源

	// 2. 定义测试服务信息
	serviceName := "payment-service"
	serviceAddr := "127.0.0.1:8888"

	// 3. 注册服务（匹配测试文件的调用方式）
	err = rrr.Register(serviceName, serviceAddr)
	if err != nil {
		log.Fatalf("首次注册服务失败: %v", err)
	}
	log.Printf("服务 %s(%s) 注册成功", serviceName, serviceAddr)

	// 4. 添加服务监听（通过回调函数接收实例变化）
	err = rrr.AddWatch(serviceName, func(addrs map[string]string) {
		log.Printf("服务实例变化，最新地址列表: %v", addrs)
	})
	if err != nil {
		log.Fatalf("添加服务监听失败: %v", err)
	}
	log.Println("开始监听服务实例变化...")

	// 5. 模拟服务注册/注销的循环（参考测试文件逻辑）
	go func() {
		for i := 0; i < 3; i++ { // 简化循环次数，便于观察
			// 重新注册（模拟服务重启）
			err = rrr.Register(serviceName, serviceAddr)
			if err != nil {
				log.Printf("第%d次注册失败: %v", i+1, err)
			} else {
				log.Printf("第%d次注册服务成功", i+1)
			}
			time.Sleep(2 * time.Second)

			// 注销服务
			err = rrr.Unregister(serviceName)
			if err != nil {
				log.Printf("第%d次注销失败: %v", i+1, err)
			} else {
				log.Printf("第%d次注销服务成功", i+1)
			}
			time.Sleep(2 * time.Second)
		}
	}()

	// 阻塞程序运行（观察监听结果）
	select {}
}
