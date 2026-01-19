# etcd-service 项目说明

## 项目简介

etcd-service 是一个基于 Go 语言开发的 etcd 服务注册与发现库，提供了简单易用的 API 来管理服务的注册、注销和监听等操作。

### 主要功能

- **服务注册**：将服务信息注册到 etcd 集群
- **服务注销**：从 etcd 集群中移除服务信息
- **服务监听**：监听指定前缀的服务变化
- **自动心跳**：通过 etcd 租约机制实现服务心跳，确保服务可用性
- **优雅关闭**：支持资源的正确释放

## 项目结构

```
etcd-service/
├── directory.go   # 服务发现和监听实现
├── go.mod         # 项目依赖管理
├── go.sum         # 依赖版本锁定
├── register.go    # 服务注册和注销实现
├── registry.go    # 核心接口和实现
└── registry_test.go # 测试文件
```

## 安装

### 前提条件

- Go 1.25.5 或更高版本
- etcd 集群（v3 版本）

### 安装方法

```bash
go get github.com/s84662355/etcdService
```

## 快速开始

### 基本使用

#### 1. 创建 Registry 实例

```go
import (
    "time"
    "github.com/s84662355/etcdService"
)

// 创建 Registry 实例
registry, err := etcdService.New(
    []string{"localhost:2379"}, // etcd 端点地址
    "", "", // 用户名和密码（如果不需要认证则为空）
    5*time.Second, // 默认超时时间
    5*time.Second, // 服务租约时间
)
if err != nil {
    panic(err)
}
defer registry.Close()
```

#### 2. 注册服务

```go
// 注册服务
err = registry.Register("/services/user-service/instance1", "192.168.1.100:8080")
if err != nil {
    panic(err)
}
```

#### 3. 监听服务变化

```go
// 监听服务变化
err = registry.AddWatch("/services/", func(services map[string]string) {
    // 处理服务变化
    for key, value := range services {
        fmt.Printf("服务: %s, 地址: %s\n", key, value)
    }
})
if err != nil {
    panic(err)
}
```

#### 4. 注销服务

```go
// 注销服务
err = registry.Unregister("/services/user-service/instance1")
if err != nil {
    panic(err)
}
```

## API 文档

### Registry 接口

```go
type Registry interface {
    AddWatch(prefix string, fn watchfn) error    // 添加服务监听
    RemoveWatch(prefix string) error             // 移除服务监听
    Register(path, value string) error           // 注册服务
    Unregister(path string) error                // 注销服务
    Close()                                      // 关闭 Registry 实例
}
```

### 方法说明

#### New

**功能**：创建一个新的 Registry 实例

**参数**：
- `Endpoints []string`：etcd 集群的端点地址列表
- `Username string`：etcd 认证用户名（可选）
- `Password string`：etcd 认证密码（可选）
- `DefaultTimeout time.Duration`：默认超时时间
- `ttl time.Duration`：服务租约时间

**返回值**：
- `Registry`：Registry 接口实例
- `error`：错误信息

#### AddWatch

**功能**：添加服务监听

**参数**：
- `prefix string`：要监听的键前缀
- `fn watchfn`：服务变化回调函数，参数为服务键值对映射

**返回值**：
- `error`：错误信息

#### RemoveWatch

**功能**：移除服务监听

**参数**：
- `prefix string`：要移除监听的键前缀

**返回值**：
- `error`：错误信息

#### Register

**功能**：注册服务

**参数**：
- `path string`：服务键路径
- `value string`：服务值（通常为服务地址）

**返回值**：
- `error`：错误信息

#### Unregister

**功能**：注销服务

**参数**：
- `path string`：服务键路径

**返回值**：
- `error`：错误信息

#### Close

**功能**：关闭 Registry 实例，释放资源

**参数**：无

**返回值**：无

## 技术实现

### 服务注册与心跳机制

1. **服务注册**：通过 etcd 的 `Put` 操作将服务信息存储到 etcd，同时关联一个租约
2. **心跳机制**：使用 etcd 的 `KeepAlive` 机制保持租约活跃，确保服务信息不会过期
3. **服务监听**：通过 etcd 的 `Watch` 机制监听服务键的变化，确保服务状态的一致性

### 服务发现机制

1. **初始获取**：通过 etcd 的 `Get` 操作（带前缀）获取初始服务列表
2. **实时监听**：通过 etcd 的 `Watch` 操作监听服务变化
3. **定时刷新**：每 5 秒主动获取一次服务列表，确保服务状态的准确性

### 并发安全

- 使用 `sync.RWMutex` 保护共享资源
- 使用 `sync.Once` 确保某些操作只执行一次
- 使用 `context` 和 `sync.WaitGroup` 实现优雅关闭

## 最佳实践

### 服务路径设计

建议使用以下路径格式：
```
/services/{service-name}/{instance-id}
```

例如：
```
/services/user-service/instance-1
/services/order-service/instance-2
```

### 错误处理

- 服务注册失败时，应进行重试
- 服务监听失败时，应重新建立监听
- 连接 etcd 失败时，应进行重连

### 性能优化

- 合理设置租约时间（ttl），避免频繁的心跳操作
- 避免监听过多的服务前缀，减少网络流量
- 使用连接池管理 etcd 连接

## 示例代码

### 完整示例

```go
package main

import (
    "fmt"
    "time"
    "github.com/s84662355/etcdService"
)

func main() {
    // 创建 Registry 实例
    registry, err := etcdService.New(
        []string{"localhost:2379"}, // etcd 端点地址
        "", "", // 用户名和密码
        5*time.Second, // 默认超时时间
        5*time.Second, // 服务租约时间
    )
    if err != nil {
        panic(err)
    }
    defer registry.Close()

    // 注册服务
    servicePath := "/services/user-service/instance-1"
    serviceAddress := "192.168.1.100:8080"
    
    err = registry.Register(servicePath, serviceAddress)
    if err != nil {
        panic(err)
    }
    fmt.Printf("服务注册成功: %s -> %s\n", servicePath, serviceAddress)

    // 监听服务变化
    err = registry.AddWatch("/services/", func(services map[string]string) {
        fmt.Println("服务列表变化:")
        for key, value := range services {
            fmt.Printf("  %s: %s\n", key, value)
        }
    })
    if err != nil {
        panic(err)
    }

    // 等待一段时间
    time.Sleep(30 * time.Second)

    // 注销服务
    err = registry.Unregister(servicePath)
    if err != nil {
        panic(err)
    }
    fmt.Printf("服务注销成功: %s\n", servicePath)

    // 等待一段时间
    time.Sleep(10 * time.Second)
}
```

## 故障排查

### 常见问题

1. **服务注册失败**
   - 检查 etcd 集群是否可用
   - 检查网络连接是否正常
   - 检查服务路径是否已存在

2. **服务监听无响应**
   - 检查 etcd 集群是否可用
   - 检查监听前缀是否正确
   - 检查回调函数是否阻塞

3. **服务心跳失败**
   - 检查网络连接是否稳定
   - 检查 etcd 集群负载是否过高
   - 考虑增加租约时间

### 日志与监控

- 建议在生产环境中添加详细的日志记录
- 建议监控 etcd 集群的健康状态
- 建议监控服务注册和发现的性能指标

## 版本历史

### v1.0.0

- 初始版本
- 支持服务注册、注销和监听
- 支持自动心跳和优雅关闭

## 贡献

欢迎提交 Issue 和 Pull Request 来改进这个项目。

### 开发流程

1. Fork 项目
2. 创建功能分支
3. 提交代码
4. 运行测试
5. 提交 Pull Request

## 许可证

本项目采用 MIT 许可证。详情请参阅 LICENSE 文件。

## 联系方式

- 作者：s84662355
- GitHub：[https://github.com/s84662355/etcdService](https://github.com/s84662355/etcdService)