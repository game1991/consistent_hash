# 一致性哈希算法

一个用 Go 语言实现的高性能、线程安全的一致性哈希库，支持虚拟节点和权重分配。

## 特性

- 💪 线程安全实现
- 🔄 支持虚拟节点，实现更均匀的分布
- ⚖️ 可配置节点权重
- 🎯 可自定义哈希函数
- 📊 详细的统计信息
- 🚀 高性能，最小化内存分配

## 安装

```bash
go get github.com/game1991/consistent_hash
```

## 快速开始

```go
package main

import (
    "fmt"
    ch "github.com/game1991/consistent_hash"
)

func main() {
    // 使用默认配置创建一个新的一致性哈希环
    ring := ch.New(nil)

    // 添加一些节点
    ring.Add("node1", "node2", "node3")

    // 添加一个自定义权重的节点
    ring.AddWithWeight("node4", 2)

    // 获取某个键对应的节点
    node := ring.Get("my-key")
    fmt.Printf("键 'my-key' 映射到节点: %s\n", node)

    // 获取多个节点用于冗余
    nodes := ring.GetN("my-key", 2)
    fmt.Printf("键 'my-key' 映射到节点列表: %v\n", nodes)
}
```

## 配置选项

```go
config := &ch.Config{
    Replicas: 100,           // 每个物理节点的虚拟节点数
    HashFunc: ch.NewCRC32(), // 使用的哈希函数
}
ring := ch.New(config)
```

## API 参考

### 创建哈希环

- `New(config *Config) *ConsistentHash`: 创建新的一致性哈希环
- `DefaultConfig() *Config`: 返回默认配置

### 节点管理

- `Add(members ...string)`: 使用默认权重添加节点
- `AddWithWeight(member string, weight int) error`: 添加指定权重的节点
- `Remove(members ...string)`: 从环中移除节点
- `Members() []string`: 获取所有当前节点

### 键操作

- `Get(key string) string`: 获取键对应的节点
- `GetN(key string, n int) []string`: 获取键对应的 N 个节点（用于冗余）

### 统计和监控

- `GetStats() *Stats`: 获取哈希环的详细统计信息，包括：
  - 物理节点总数
  - 虚拟节点总数
  - 平均权重
  - 权重分布
  - 负载分布

## 实现细节

本包实现了一致性哈希算法，具有以下关键特性：

1. **虚拟节点**: 每个物理节点在哈希环上由多个虚拟节点表示，确保更好的分布
2. **权重分配**: 节点可以有不同的权重，影响其虚拟节点数量和负责范围
3. **线程安全**: 通过适当的互斥锁使用确保所有操作都是线程安全的
4. **自定义哈希函数**: 通过 `Hasher` 接口支持自定义哈希函数

## 性能考虑

- 使用固定大小的字节数组进行哈希计算，最小化内存分配
- 使用高效的二分查找进行节点查找
- 预分配哈希环容量以减少重新分配

## 使用场景

1. **分布式缓存**: 在多个缓存节点之间分配数据
2. **负载均衡**: 在多个服务器之间分配请求
3. **分布式存储**: 在分布式存储系统中确定数据存储位置
4. **服务发现**: 在微服务架构中进行服务节点选择

## 许可证

MIT License

## 贡献

欢迎提交 Pull Request 来改进这个项目！
