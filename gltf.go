package mst

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image/png"
	"io"

	mat4d "github.com/flywave/go3d/float64/mat4"

	"github.com/flywave/gltf"
	"github.com/flywave/gltf/ext/specular"
)

const (
	// GLTFVersion 定义GLTF规范版本
	GLTFVersion = "2.0"

	// PaddingChar 用于二进制填充的字符
	PaddingChar = 0x20
)

// MstToGltf 将MST网格转换为GLTF文档
func MstToGltf(meshes []*Mesh) (*gltf.Document, error) {
	doc := CreateDoc()
	for _, mesh := range meshes {
		if err := BuildGltf(doc, mesh, false); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// MstToGltfWithOutline 将MST网格转换为GLTF文档并包含轮廓线
func MstToGltfWithOutline(meshes []*Mesh) (*gltf.Document, error) {
	doc := CreateDoc()
	for _, mesh := range meshes {
		if err := BuildGltf(doc, mesh, true); err != nil {
			return nil, err
		}
	}
	return doc, nil
}

// CreateDoc 创建一个新的GLTF文档
func CreateDoc() *gltf.Document {
	doc := &gltf.Document{
		Asset: gltf.Asset{
			Version: GLTFVersion,
		},
		Scenes:  []*gltf.Scene{{}},
		Buffers: []*gltf.Buffer{{}},
	}

	sceneIndex := uint32(0)
	doc.Scene = &sceneIndex

	return doc
}

// bufferWriter 用于计算缓冲区大小的写入器
type bufferWriter struct {
	writer io.Writer
	size   int
}

func (w *bufferWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.writer.Write(p)
	w.size += n
	return n, nil
}

func (w *bufferWriter) Bytes() []byte {
	return w.writer.(*bytes.Buffer).Bytes()
}

func (w *bufferWriter) Size() int {
	return len(w.Bytes())
}

func newBufferWriter() *bufferWriter {
	return &bufferWriter{
		writer: bytes.NewBuffer(nil),
		size:   0,
	}
}

// calcPadding 计算需要的填充字节数
func calcPadding(offset, unit int) int {
	padding := offset % unit
	if padding != 0 {
		padding = unit - padding
	}
	return padding
}

// GetGltfBinary 将GLTF文档编码为二进制格式
func GetGltfBinary(doc *gltf.Document, paddingUnit int) ([]byte, error) {
	writer := newBufferWriter()

	encoder := gltf.NewEncoder(writer.writer)
	encoder.AsBinary = true

	if err := encoder.Encode(doc); err != nil {
		return nil, err
	}

	padding := calcPadding(writer.size, paddingUnit)
	if padding == 0 {
		return writer.Bytes(), nil
	}

	pad := bytes.Repeat([]byte{PaddingChar}, padding)
	writer.Write(pad)

	return writer.Bytes(), nil
}

// BuildGltf 构建GLTF文档
func BuildGltf(doc *gltf.Document, mesh *Mesh, exportOutline bool) error {
	// 处理主网格的属性
	if mesh.Props != nil && len(*mesh.Props) > 0 {
		if doc.Extensions == nil {
			doc.Extensions = make(map[string]interface{})
		}

		// 将属性添加到文档扩展中
		propsMap := propsToMap(mesh.Props)
		if propsMap != nil {
			if ext, exists := doc.Extensions["MST_mesh_properties"]; exists {
				if extMap, ok := ext.(map[string]interface{}); ok {
					// 合并属性
					for k, v := range propsMap {
						extMap[k] = v
					}
				} else {
					doc.Extensions["MST_mesh_properties"] = propsMap
				}
			} else {
				doc.Extensions["MST_mesh_properties"] = propsMap
			}
		}
	}

	if err := buildGltfFromBaseMesh(doc, &mesh.BaseMesh, nil, exportOutline); err != nil {
		return err
	}

	for i, instance := range mesh.Instances {
		// 处理实例网格的属性
		if len(instance.Props) > 0 {
			// 只使用第一个Props元素，或者合并所有Props元素
			for j, props := range instance.Props {
				if props != nil && len(*props) > 0 {
					if doc.Extensions == nil {
						doc.Extensions = make(map[string]interface{})
					}

					// 将实例属性添加到文档扩展中，使用实例索引和属性索引作为键
					propsMap := propsToMap(props)
					if propsMap != nil {
						instanceKey := "MST_instance_mesh_properties_" + fmt.Sprintf("%d_%d", i, j)
						doc.Extensions[instanceKey] = propsMap
					}
				}
			}
		}

		if err := buildGltfFromBaseMesh(doc, instance.Mesh, instance.Transfors, false); err != nil {
			return err
		}
	}

	return nil
}

// buildContext 构建上下文，存储构建过程中的状态
type buildContext struct {
	mtlSize uint32

	// 缓冲区视图索引
	bvIndex uint32
	bvPos   uint32
	bvTex   uint32
	bvNorm  uint32
}

// buildMeshBufferViews 构建网格的缓冲区视图
func buildMeshBufferViews(ctx *buildContext, buffer *gltf.Buffer, bufferViews []*gltf.BufferView, node *MeshNode) []*gltf.BufferView {
	buf := bytes.NewBuffer(nil)

	ctx.bvIndex = uint32(len(bufferViews))

	// 索引数据
	indicesView := &gltf.BufferView{
		ByteOffset: buffer.ByteLength,
		Buffer:     0,
	}

	for _, group := range node.FaceGroup {
		for _, face := range group.Faces {
			binary.Write(buf, binary.LittleEndian, face.Vertex)
		}
	}

	indicesView.ByteLength = uint32(buf.Len())
	bufferViews = append(bufferViews, indicesView)

	// 顶点位置数据
	positionsView := &gltf.BufferView{
		ByteOffset: uint32(buf.Len()) + buffer.ByteLength,
		Buffer:     0,
	}
	binary.Write(buf, binary.LittleEndian, node.Vertices)
	positionsView.ByteLength = uint32(buf.Len()) - positionsView.ByteOffset + buffer.ByteLength
	ctx.bvPos = uint32(len(bufferViews))
	bufferViews = append(bufferViews, positionsView)

	// 纹理坐标数据
	if len(node.TexCoords) > 0 {
		texCoordsView := &gltf.BufferView{
			ByteOffset: uint32(buf.Len()) + buffer.ByteLength,
			Buffer:     0,
		}
		binary.Write(buf, binary.LittleEndian, node.TexCoords)
		texCoordsView.ByteLength = uint32(buf.Len()) - texCoordsView.ByteOffset + buffer.ByteLength
		ctx.bvTex = uint32(len(bufferViews))
		bufferViews = append(bufferViews, texCoordsView)
	}

	// 法线数据
	if len(node.Normals) > 0 {
		normalsView := &gltf.BufferView{
			ByteOffset: uint32(buf.Len()) + buffer.ByteLength,
			Buffer:     0,
		}
		binary.Write(buf, binary.LittleEndian, node.Normals)
		normalsView.ByteLength = uint32(buf.Len()) - normalsView.ByteOffset + buffer.ByteLength
		ctx.bvNorm = uint32(len(bufferViews))
		bufferViews = append(bufferViews, normalsView)
	}

	buffer.ByteLength += uint32(buf.Len())
	buffer.Data = append(buffer.Data, buf.Bytes()...)

	return bufferViews
}

// buildOutlineBufferViews 构建轮廓线的缓冲区视图
func buildOutlineBufferViews(ctx *buildContext, buffer *gltf.Buffer, bufferViews []*gltf.BufferView, node *MeshNode) []*gltf.BufferView {
	buf := bytes.NewBuffer(nil)

	ctx.bvIndex = uint32(len(bufferViews))

	// 索引数据
	indicesView := &gltf.BufferView{
		ByteOffset: buffer.ByteLength,
		Buffer:     0,
	}

	for _, group := range node.EdgeGroup {
		for _, edge := range group.Edges {
			binary.Write(buf, binary.LittleEndian, edge)
		}
	}

	indicesView.ByteLength = uint32(buf.Len())
	bufferViews = append(bufferViews, indicesView)

	// 顶点位置数据
	positionsView := &gltf.BufferView{
		ByteOffset: uint32(buf.Len()) + buffer.ByteLength,
		Buffer:     0,
	}
	binary.Write(buf, binary.LittleEndian, node.Vertices)
	positionsView.ByteLength = uint32(buf.Len()) - positionsView.ByteOffset + buffer.ByteLength
	ctx.bvPos = uint32(len(bufferViews))
	bufferViews = append(bufferViews, positionsView)

	buffer.ByteLength += uint32(buf.Len())
	buffer.Data = append(buffer.Data, buf.Bytes()...)

	return bufferViews
}

// buildOutlineMesh 构建轮廓线网格
func buildOutlineMesh(ctx *buildContext, accessors []*gltf.Accessor, node *MeshNode) (*gltf.Mesh, []*gltf.Accessor) {
	mesh := &gltf.Mesh{}

	accessorOffset := uint32(len(accessors))
	positionAccessorIndex := uint32(len(node.EdgeGroup)) + accessorOffset

	var startOffset uint32 = 0

	for i, group := range node.EdgeGroup {
		batchID := group.Batchid
		if batchID < 0 {
			batchID = 0
		}

		materialID := uint32(batchID) + ctx.mtlSize

		primitive := &gltf.Primitive{
			Material: &materialID,
			Indices:  uint32Ptr(uint32(i) + accessorOffset),
			Mode:     gltf.PrimitiveLineStrip,
			Attributes: gltf.Attribute{
				"POSITION": positionAccessorIndex,
			},
		}

		mesh.Primitives = append(mesh.Primitives, primitive)

		// 索引访问器
		indexAccessor := &gltf.Accessor{
			ComponentType: gltf.ComponentUint,
			ByteOffset:    startOffset * 8,
			Count:         uint32(len(group.Edges)) * 2,
			BufferView:    uint32Ptr(ctx.bvIndex),
		}

		accessors = append(accessors, indexAccessor)
		startOffset += uint32(len(group.Edges))
	}

	// 位置访问器
	bounds := node.GetBoundbox()
	positionAccessor := &gltf.Accessor{
		ComponentType: gltf.ComponentFloat,
		Type:          gltf.AccessorVec3,
		Count:         uint32(len(node.Vertices)),
		BufferView:    uint32Ptr(ctx.bvPos),
		Min:           []float32{float32(bounds[0]), float32(bounds[1]), float32(bounds[2])},
		Max:           []float32{float32(bounds[3]), float32(bounds[4]), float32(bounds[5])},
	}
	accessors = append(accessors, positionAccessor)

	return mesh, accessors
}

// buildMeshPrimitives 构建网格图元
func buildMeshPrimitives(ctx *buildContext, accessors []*gltf.Accessor, node *MeshNode) (*gltf.Mesh, []*gltf.Accessor) {
	mesh := &gltf.Mesh{}

	accessorOffset := uint32(len(accessors))
	positionAccessorIndex := uint32(len(node.FaceGroup)) + accessorOffset

	var startOffset uint32 = 0

	for i, group := range node.FaceGroup {
		batchID := group.Batchid
		if batchID < 0 {
			batchID = 0
		}

		materialID := uint32(batchID) + ctx.mtlSize

		// 计算属性索引
		attributeIndex := positionAccessorIndex
		attributes := gltf.Attribute{"POSITION": attributeIndex}

		if len(node.TexCoords) > 0 {
			attributeIndex++
			attributes["TEXCOORD_0"] = attributeIndex
		}

		if len(node.Normals) > 0 {
			attributeIndex++
			attributes["NORMAL"] = attributeIndex
		}

		primitive := &gltf.Primitive{
			Material:   &materialID,
			Indices:    uint32Ptr(uint32(i) + accessorOffset),
			Mode:       gltf.PrimitiveTriangles,
			Attributes: attributes,
		}

		mesh.Primitives = append(mesh.Primitives, primitive)

		// 索引访问器
		indexAccessor := &gltf.Accessor{
			ComponentType: gltf.ComponentUint,
			ByteOffset:    startOffset * 12,
			Count:         uint32(len(group.Faces)) * 3,
			BufferView:    uint32Ptr(ctx.bvIndex),
		}

		accessors = append(accessors, indexAccessor)
		startOffset += uint32(len(group.Faces))
	}

	// 位置访问器
	bounds := node.GetBoundbox()
	positionAccessor := &gltf.Accessor{
		ComponentType: gltf.ComponentFloat,
		Type:          gltf.AccessorVec3,
		Count:         uint32(len(node.Vertices)),
		BufferView:    uint32Ptr(ctx.bvPos),
		Min:           []float32{float32(bounds[0]), float32(bounds[1]), float32(bounds[2])},
		Max:           []float32{float32(bounds[3]), float32(bounds[4]), float32(bounds[5])},
	}
	accessors = append(accessors, positionAccessor)

	// 纹理坐标访问器
	if len(node.TexCoords) > 0 {
		texCoordAccessor := &gltf.Accessor{
			ComponentType: gltf.ComponentFloat,
			Type:          gltf.AccessorVec2,
			Count:         uint32(len(node.TexCoords)),
			BufferView:    uint32Ptr(ctx.bvTex),
		}
		accessors = append(accessors, texCoordAccessor)
	}

	// 法线访问器
	if len(node.Normals) > 0 {
		normalAccessor := &gltf.Accessor{
			ComponentType: gltf.ComponentFloat,
			Type:          gltf.AccessorVec3,
			Count:         uint32(len(node.Normals)),
			BufferView:    uint32Ptr(ctx.bvNorm),
		}
		accessors = append(accessors, normalAccessor)
	}

	return mesh, accessors
}

// buildGltfFromBaseMesh 从基础网格构建GLTF
func buildGltfFromBaseMesh(doc *gltf.Document, mesh *BaseMesh, transforms []*mat4d.T, exportOutline bool) error {
	ctx := &buildContext{
		mtlSize: uint32(len(doc.Materials)),
	}

	for _, node := range mesh.Nodes {
		meshIndex := uint32(len(doc.Meshes))

		if exportOutline && len(node.EdgeGroup) > 0 {
			doc.BufferViews = buildOutlineBufferViews(ctx, doc.Buffers[0], doc.BufferViews, node)

			outlineMesh, accessors := buildOutlineMesh(ctx, doc.Accessors, node)
			doc.Meshes = append(doc.Meshes, outlineMesh)
			doc.Accessors = accessors
		} else {
			doc.BufferViews = buildMeshBufferViews(ctx, doc.Buffers[0], doc.BufferViews, node)

			mesh, accessors := buildMeshPrimitives(ctx, doc.Accessors, node)
			doc.Meshes = append(doc.Meshes, mesh)
			doc.Accessors = accessors
		}

		if transforms == nil {
			// 无变换矩阵，直接添加节点
			nodeIndex := uint32(len(doc.Nodes))
			gltfNode := &gltf.Node{Mesh: &meshIndex}

			if node.Mat != nil {
				position, rotation, scale := mat4d.Decompose(node.Mat)
				gltfNode.Translation = [3]float32{float32(position[0]), float32(position[1]), float32(position[2])}
				gltfNode.Rotation = [4]float32{float32(rotation[0]), float32(rotation[1]), float32(rotation[2]), float32(rotation[3])}
				gltfNode.Scale = [3]float32{float32(scale[0]), float32(scale[1]), float32(scale[2])}
			}

			doc.Nodes = append(doc.Nodes, gltfNode)
			doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, nodeIndex)
		} else {
			// 应用变换矩阵
			for _, transform := range transforms {
				position, rotation, scale := mat4d.Decompose(transform)
				gltfNode := &gltf.Node{
					Mesh:        &meshIndex,
					Translation: [3]float32{float32(position[0]), float32(position[1]), float32(position[2])},
					Rotation:    [4]float32{float32(rotation[0]), float32(rotation[1]), float32(rotation[2]), float32(rotation[3])},
					Scale:       [3]float32{float32(scale[0]), float32(scale[1]), float32(scale[2])},
				}

				doc.Nodes = append(doc.Nodes, gltfNode)
				doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, uint32(len(doc.Nodes)-1))
			}
		}
	}

	return fillMaterials(doc, mesh.Materials)
}

// buildTexture 构建纹理
func buildTexture(doc *gltf.Document, buffer *gltf.Buffer, texture *Texture) (*gltf.Texture, error) {
	samplerIndex := uint32(len(doc.Samplers))
	imageIndex := uint32(len(doc.Images))

	gltfTexture := &gltf.Texture{
		Sampler: &samplerIndex,
		Source:  &imageIndex,
	}

	// 加载图像
	img, err := LoadTexture(texture, true)
	if err != nil {
		return nil, err
	}

	// 编码PNG
	buf := bytes.NewBuffer(nil)
	if err := png.Encode(buf, img); err != nil {
		return nil, err
	}

	// 创建缓冲区视图
	bufferViewIndex := uint32(len(doc.BufferViews))
	bufferView := &gltf.BufferView{
		ByteOffset: buffer.ByteLength,
		ByteLength: uint32(buf.Len()),
		Buffer:     0,
	}

	buffer.ByteLength += uint32(buf.Len())
	buffer.Data = append(buffer.Data, buf.Bytes()...)
	doc.BufferViews = append(doc.BufferViews, bufferView)

	// 创建图像
	gltfImage := &gltf.Image{
		MimeType:   "image/png",
		BufferView: &bufferViewIndex,
	}
	doc.Images = append(doc.Images, gltfImage)

	// 创建采样器
	var sampler *gltf.Sampler
	if texture.Repeated {
		sampler = &gltf.Sampler{
			WrapS: gltf.WrapRepeat,
			WrapT: gltf.WrapRepeat,
		}
	} else {
		sampler = &gltf.Sampler{
			WrapS: gltf.WrapClampToEdge,
			WrapT: gltf.WrapClampToEdge,
		}
	}
	doc.Samplers = append(doc.Samplers, sampler)

	return gltfTexture, nil
}

// fillMaterials 填充材质数据
func fillMaterials(doc *gltf.Document, materials []MeshMaterial) error {
	textureMap := make(map[int32]uint32)
	useExtension := false

	for _, material := range materials {
		gltfMaterial := &gltf.Material{
			DoubleSided: true,
			AlphaMode:   gltf.AlphaMask,
			PBRMetallicRoughness: &gltf.PBRMetallicRoughness{
				BaseColorFactor: &[4]float32{1, 1, 1, 1},
			},
			Extensions: make(map[string]interface{}),
		}

		var textureMaterial *TextureMaterial
		var baseColor *[4]float32

		switch mtl := material.(type) {
		case *BaseMaterial:
			baseColor = &[4]float32{
				float32(mtl.Color[0]) / 255,
				float32(mtl.Color[1]) / 255,
				float32(mtl.Color[2]) / 255,
				1 - float32(mtl.Transparency),
			}

		case *PbrMaterial:
			baseColor = &[4]float32{
				float32(mtl.Color[0]) / 255,
				float32(mtl.Color[1]) / 255,
				float32(mtl.Color[2]) / 255,
				1 - float32(mtl.Transparency),
			}

			metallic := float32(mtl.Metallic)
			roughness := float32(mtl.Roughness)
			gltfMaterial.PBRMetallicRoughness.MetallicFactor = &metallic
			gltfMaterial.PBRMetallicRoughness.RoughnessFactor = &roughness

			gltfMaterial.EmissiveFactor = [3]float32{
				float32(mtl.Emissive[0]) / 255,
				float32(mtl.Emissive[1]) / 255,
				float32(mtl.Emissive[2]) / 255,
			}

			textureMaterial = &mtl.TextureMaterial

		case *LambertMaterial:
			baseColor = &[4]float32{
				float32(mtl.Color[0]) / 255,
				float32(mtl.Color[1]) / 255,
				float32(mtl.Color[2]) / 255,
				1 - float32(mtl.Transparency),
			}

			specularGlossiness := &specular.PBRSpecularGlossiness{
				DiffuseFactor: &[4]float32{
					float32(mtl.Diffuse[0]) / 255,
					float32(mtl.Diffuse[1]) / 255,
					float32(mtl.Diffuse[2]) / 255,
					1,
				},
			}

			gltfMaterial.EmissiveFactor = [3]float32{
				float32(mtl.Emissive[0]) / 255,
				float32(mtl.Emissive[1]) / 255,
				float32(mtl.Emissive[2]) / 255,
			}

			gltfMaterial.Extensions[specular.ExtensionName] = specularGlossiness
			useExtension = true
			textureMaterial = &mtl.TextureMaterial

		case *PhongMaterial:
			baseColor = &[4]float32{
				float32(mtl.Color[0]) / 255,
				float32(mtl.Color[1]) / 255,
				float32(mtl.Color[2]) / 255,
				1 - float32(mtl.Transparency),
			}

			specularGlossiness := &specular.PBRSpecularGlossiness{
				DiffuseFactor: &[4]float32{
					float32(mtl.Diffuse[0]) / 255,
					float32(mtl.Diffuse[1]) / 255,
					float32(mtl.Diffuse[2]) / 255,
					1,
				},
				SpecularFactor: &[3]float32{
					float32(mtl.Specular[0]) / 255,
					float32(mtl.Specular[1]) / 255,
					float32(mtl.Specular[2]) / 255,
				},
				GlossinessFactor: &mtl.Shininess,
			}

			gltfMaterial.EmissiveFactor = [3]float32{
				float32(mtl.Emissive[0]) / 255,
				float32(mtl.Emissive[1]) / 255,
				float32(mtl.Emissive[2]) / 255,
			}

			gltfMaterial.Extensions[specular.ExtensionName] = specularGlossiness
			useExtension = true
			textureMaterial = &mtl.TextureMaterial

		case *TextureMaterial:
			textureMaterial = mtl
			baseColor = &[4]float32{
				float32(mtl.Color[0]) / 255,
				float32(mtl.Color[1]) / 255,
				float32(mtl.Color[2]) / 255,
				1 - float32(mtl.Transparency),
			}
		}

		// 处理基础颜色纹理
		if textureMaterial != nil && textureMaterial.HasTexture() {
			if index, exists := textureMap[textureMaterial.Texture.Id]; exists {
				gltfMaterial.PBRMetallicRoughness.BaseColorTexture = &gltf.TextureInfo{Index: index}
			} else {
				textureIndex := uint32(len(doc.Textures))
				textureMap[textureMaterial.Texture.Id] = textureIndex

				tex, err := buildTexture(doc, doc.Buffers[0], textureMaterial.Texture)
				if err != nil {
					return err
				}

				gltfMaterial.PBRMetallicRoughness.BaseColorTexture = &gltf.TextureInfo{Index: textureIndex}
				doc.Textures = append(doc.Textures, tex)
			}
		}

		// 处理法线纹理
		if textureMaterial != nil && textureMaterial.HasNormalTexture() {
			if index, exists := textureMap[textureMaterial.Normal.Id]; exists {
				gltfMaterial.NormalTexture = &gltf.NormalTexture{Index: &index}
			} else {
				normalTextureIndex := uint32(len(doc.Textures))
				textureMap[textureMaterial.Normal.Id] = normalTextureIndex

				tex, err := buildTexture(doc, doc.Buffers[0], textureMaterial.Normal)
				if err != nil {
					return err
				}

				gltfMaterial.NormalTexture = &gltf.NormalTexture{Index: &normalTextureIndex}
				doc.Textures = append(doc.Textures, tex)
			}
		}

		gltfMaterial.PBRMetallicRoughness.BaseColorFactor = baseColor

		// 设置默认值
		if gltfMaterial.PBRMetallicRoughness.MetallicFactor == nil {
			defaultMetallic := float32(0)
			gltfMaterial.PBRMetallicRoughness.MetallicFactor = &defaultMetallic
		}

		if gltfMaterial.PBRMetallicRoughness.RoughnessFactor == nil {
			defaultRoughness := float32(1)
			gltfMaterial.PBRMetallicRoughness.RoughnessFactor = &defaultRoughness
		}

		doc.Materials = append(doc.Materials, gltfMaterial)
	}

	// 添加扩展
	if useExtension {
		for _, ext := range doc.ExtensionsUsed {
			if ext == specular.ExtensionName {
				return nil
			}
		}
		doc.ExtensionsUsed = append(doc.ExtensionsUsed, specular.ExtensionName)
	}

	return nil
}

// uint32Ptr 返回uint32指针的辅助函数
func uint32Ptr(v uint32) *uint32 {
	return &v
}

// propsToMap 将Properties转换为map[string]interface{}格式，以便序列化到GLTF扩展中
func propsToMap(props *Properties) map[string]interface{} {
	if props == nil {
		return nil
	}

	result := make(map[string]interface{})
	for key, value := range *props {
		result[key] = propsValueToInterface(value)
	}
	return result
}

// propsValueToInterface 将PropsValue转换为interface{}格式
func propsValueToInterface(value PropsValue) interface{} {
	switch value.Type {
	case PROP_TYPE_STRING:
		return value.Value.(string)
	case PROP_TYPE_INT:
		return value.Value.(int64)
	case PROP_TYPE_FLOAT:
		return value.Value.(float64)
	case PROP_TYPE_BOOL:
		return value.Value.(bool)
	case PROP_TYPE_ARRAY:
		arr := value.Value.([]PropsValue)
		result := make([]interface{}, len(arr))
		for i, item := range arr {
			result[i] = propsValueToInterface(item)
		}
		return result
	case PROP_TYPE_MAP:
		subProps := value.Value.(Properties)
		return propsToMap(&subProps)
	default:
		return nil
	}
}
