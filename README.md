# Consistent Hash

A high-performance, thread-safe consistent hashing implementation in Go, supporting virtual nodes and weighted distribution.

## Features

- 💪 Thread-safe implementation
- 🔄 Virtual nodes support for better distribution
- ⚖️ Configurable node weights
- 🎯 Customizable hash functions
- 📊 Detailed statistics for monitoring
- 🚀 High performance with minimal memory allocations

## Installation

```bash
go get github.com/game1991/consistent_hash
```

## Quick Start

```go
package main

import (
    "fmt"
    ch "github.com/game1991/consistent_hash"
)

func main() {
    // Create a new consistent hash ring with default configuration
    ring := ch.New(nil)

    // Add some nodes
    ring.Add("node1", "node2", "node3")

    // Add a node with custom weight
    ring.AddWithWeight("node4", 2)

    // Get the node for a key
    node := ring.Get("my-key")
    fmt.Printf("Key 'my-key' maps to node: %s\n", node)

    // Get multiple nodes for redundancy
    nodes := ring.GetN("my-key", 2)
    fmt.Printf("Key 'my-key' maps to nodes: %v\n", nodes)
}
```

## Configuration

```go
config := &ch.Config{
    Replicas: 100,    // Number of virtual nodes per physical node
    HashFunc: ch.NewCRC32(), // Hash function to use
}
ring := ch.New(config)
```

## API Reference

### Creating a New Hash Ring

- `New(config *Config) *ConsistentHash`: Creates a new consistent hash ring
- `DefaultConfig() *Config`: Returns default configuration

### Managing Nodes

- `Add(members ...string)`: Add nodes with default weight
- `AddWithWeight(member string, weight int) error`: Add a node with specific weight
- `Remove(members ...string)`: Remove nodes from the ring
- `Members() []string`: Get all current nodes

### Key Operations

- `Get(key string) string`: Get the node for a key
- `GetN(key string, n int) []string`: Get N nodes for a key (for redundancy)

### Statistics and Monitoring

- `GetStats() *Stats`: Get detailed statistics about the hash ring, including:
  - Total physical nodes
  - Total virtual nodes
  - Average weight
  - Weight distribution
  - Load distribution

## Implementation Details

This package implements consistent hashing with the following key features:

1. **Virtual Nodes**: Each physical node is represented by multiple virtual nodes on the hash ring to ensure better distribution.
2. **Weighted Distribution**: Nodes can have different weights, affecting their virtual node count and responsibility range.
3. **Thread Safety**: All operations are thread-safe through proper mutex usage.
4. **Customizable Hash Function**: Supports custom hash functions through the `Hasher` interface.

## Performance Considerations

- Uses fixed-size byte arrays for hash calculations to minimize memory allocations
- Efficient binary search for node lookup
- Pre-allocated capacity for the hash ring to reduce reallocations

## License

MIT License

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
