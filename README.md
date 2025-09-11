# go-mst

[![Go Reference](https://pkg.go.dev/badge/github.com/flywave/go-mst.svg)](https://pkg.go.dev/github.com/flywave/go-mst)
[![Go Report Card](https://goreportcard.com/badge/github.com/flywave/go-mst)](https://goreportcard.com/report/github.com/flywave/go-mst)
[![License](https://img.shields.io/github/license/flywave/go-mst)](https://github.com/flywave/go-mst/blob/main/LICENSE)

Go-MST 是一个用于处理 MST (Mesh Scene Tree) 格式的 Go 语言库，支持将 MST 网格数据转换为 glTF 格式，便于在 WebGL 和其他 3D 应用中使用。

## 功能特性

- **MST 网格处理**: 支持读取、写入和操作 MST 网格数据
- **glTF 转换**: 将 MST 网格转换为标准 glTF 2.0 格式
- **多种材质支持**: 支持基础材质、PBR 材质、Lambert 材质和 Phong 材质
- **纹理处理**: 支持纹理映射和法线贴图
- **属性系统**: 支持网格、节点和实例的自定义属性
- **轮廓线导出**: 支持将网格轮廓线导出为 glTF 线条
- **二进制格式**: 支持生成二进制 glTF (.glb) 文件

## 安装

```bash
go get github.com/flywave/go-mst
```

## 快速开始

### 基本用法

```go
package main

import (
    "github.com/flywave/go-mst"
)

func main() {
    // 创建一个新的 MST 网格
    mesh := mst.NewMesh()
    
    // 添加网格数据...
    
    // 转换为 glTF
    doc, err := mst.MstToGltf([]*mst.Mesh{mesh})
    if err != nil {
        panic(err)
    }
    
    // 导出为二进制格式
    binary, err := mst.GetGltfBinary(doc, 4)
    if err != nil {
        panic(err)
    }
    
    // 使用 binary 数据...
}
```

### 使用属性系统

```go
// 为网格添加自定义属性
(*mesh.Props)["name"] = mst.PropsValue{
    Type:  mst.PROP_TYPE_STRING, 
    Value: "MyMesh",
}

(*mesh.Props)["version"] = mst.PropsValue{
    Type:  mst.PROP_TYPE_INT, 
    Value: int64(1),
}
```

## API 文档

详细的 API 文档请参考 [GoDoc](https://pkg.go.dev/github.com/flywave/go-mst)。

## 支持的材质类型

- `BaseMaterial`: 基础材质，支持颜色和透明度
- `PbrMaterial`: PBR (Physically Based Rendering) 材质
- `LambertMaterial`: Lambert 材质
- `PhongMaterial`: Phong 材质
- `TextureMaterial`: 纹理材质，支持基础纹理和法线贴图

## 支持的属性类型

- `PROP_TYPE_STRING`: 字符串类型
- `PROP_TYPE_INT`: 整数类型
- `PROP_TYPE_FLOAT`: 浮点数类型
- `PROP_TYPE_BOOL`: 布尔类型
- `PROP_TYPE_ARRAY`: 数组类型
- `PROP_TYPE_MAP`: 映射类型

## 贡献

欢迎提交 Issue 和 Pull Request。在提交代码前，请确保：

1. 代码符合 Go 语言规范
2. 添加了相应的单元测试
3. 通过了所有测试

```bash
go test ./...
```

## 许可证

本项目采用 [MIT](LICENSE) 许可证。