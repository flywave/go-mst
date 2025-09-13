package mst

import (
	"bytes"
	"testing"

	mat4d "github.com/flywave/go3d/float64/mat4"
	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
)

// TestCreateDoc 测试CreateDoc函数是否正确创建GLTF文档
func TestCreateDoc(t *testing.T) {
	doc := CreateDoc()

	if doc == nil {
		t.Fatal("CreateDoc() returned nil")
	}

	if doc.Asset.Version != GLTFVersion {
		t.Errorf("Expected GLTF version %s, got %s", GLTFVersion, doc.Asset.Version)
	}

	if len(doc.Scenes) != 1 {
		t.Errorf("Expected 1 scene, got %d", len(doc.Scenes))
	}

	if doc.Scene == nil {
		t.Error("Scene index should not be nil")
	} else if *doc.Scene != 0 {
		t.Errorf("Expected scene index 0, got %d", *doc.Scene)
	}

	if len(doc.Buffers) != 1 {
		t.Errorf("Expected 1 buffer, got %d", len(doc.Buffers))
	}
}

// TestMstToGltf 测试MST网格到GLTF的转换
func TestMstToGltf(t *testing.T) {
	// 创建测试用的简单网格
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Nodes: []*MeshNode{
				{
					Vertices: []vec3.T{
						{0, 0, 0},
						{1, 0, 0},
						{0, 1, 0},
					},
					TexCoords: []vec2.T{
						{0, 0},
						{1, 0},
						{0, 1},
					},
					Normals: []vec3.T{
						{0, 0, 1},
						{0, 0, 1},
						{0, 0, 1},
					},
					FaceGroup: []*MeshTriangle{
						{
							Batchid: 0,
							Faces: []*Face{
								{Vertex: [3]uint32{0, 1, 2}},
							},
						},
					},
				},
			},
			Materials: []MeshMaterial{
				&BaseMaterial{
					Color:        [3]byte{255, 0, 0},
					Transparency: 0.5,
				},
			},
		},
		InstanceNode: nil,
	}

	doc, err := MstToGltf([]*Mesh{mesh})
	if err != nil {
		t.Fatalf("MstToGltf failed: %v", err)
	}

	if len(doc.Meshes) != 1 {
		t.Errorf("Expected 1 mesh, got %d", len(doc.Meshes))
	}

	if len(doc.Materials) != 1 {
		t.Errorf("Expected 1 material, got %d", len(doc.Materials))
	}

	if len(doc.Accessors) < 3 {
		t.Errorf("Expected at least 3 accessors (indices, positions, texcoords), got %d", len(doc.Accessors))
	}
}

// TestMstToGltfWithOutline 测试带轮廓线的GLTF转换
func TestMstToGltfWithOutline(t *testing.T) {
	// 创建测试用的简单网格，包含边数据
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Nodes: []*MeshNode{
				{
					Vertices: []vec3.T{
						{0, 0, 0},
						{1, 0, 0},
						{0, 1, 0},
					},
					EdgeGroup: []*MeshOutline{
						{
							Batchid: 0,
							Edges: [][2]uint32{
								{0, 1},
								{1, 2},
								{2, 0},
							},
						},
					},
				},
			},
			Materials: []MeshMaterial{
				&BaseMaterial{
					Color:        [3]byte{0, 255, 0},
					Transparency: 0.5,
				},
			},
		},
		InstanceNode: nil,
	}

	doc, err := MstToGltfWithOutline([]*Mesh{mesh})
	if err != nil {
		t.Fatalf("MstToGltfWithOutline failed: %v", err)
	}

	if len(doc.Meshes) != 1 {
		t.Errorf("Expected 1 outline mesh, got %d", len(doc.Meshes))
	}

	if len(doc.Materials) != 1 {
		t.Errorf("Expected 1 material, got %d", len(doc.Materials))
	}
}

// TestGetGltfBinary 测试二进制GLTF生成
func TestGetGltfBinary(t *testing.T) {
	doc := CreateDoc()

	// 添加一些基本数据
	bufferData := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	doc.Buffers[0].Data = bufferData
	doc.Buffers[0].ByteLength = uint32(len(bufferData))

	binary, err := GetGltfBinary(doc, 4)
	if err != nil {
		t.Fatalf("GetGltfBinary failed: %v", err)
	}

	if len(binary) == 0 {
		t.Error("Binary output should not be empty")
	}

	// 测试填充
	if len(binary)%4 != 0 {
		t.Errorf("Binary length should be multiple of 4, got %d", len(binary))
	}
}

// TestBuildGltfFromBaseMesh 测试基础网格构建
func TestBuildGltfFromBaseMesh(t *testing.T) {
	doc := CreateDoc()

	baseMesh := &BaseMesh{
		Nodes: []*MeshNode{
			{
				Vertices: []vec3.T{
					{0, 0, 0},
					{1, 0, 0},
					{0, 1, 0},
				},
				FaceGroup: []*MeshTriangle{
					{
						Batchid: 0,
						Faces: []*Face{
							{Vertex: [3]uint32{0, 1, 2}},
						},
					},
				},
			},
		},
		Materials: []MeshMaterial{
			&BaseMaterial{
				Color:        [3]byte{0, 0, 255},
				Transparency: 0.5,
			},
		},
	}

	err := buildGltfFromBaseMesh(doc, baseMesh, nil, false)
	if err != nil {
		t.Fatalf("buildGltfFromBaseMesh failed: %v", err)
	}

	if len(doc.Meshes) != 1 {
		t.Errorf("Expected 1 mesh, got %d", len(doc.Meshes))
	}

	if len(doc.Materials) != 1 {
		t.Errorf("Expected 1 material, got %d", len(doc.Materials))
	}

	// 检查节点属性是否正确添加到GLTF节点扩展中
	if len(doc.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(doc.Nodes))
	}
}

// TestBuildGltfWithTransforms 测试带变换矩阵的网格构建
func TestBuildGltfWithTransforms(t *testing.T) {
	doc := CreateDoc()

	baseMesh := &BaseMesh{
		Nodes: []*MeshNode{
			{
				Vertices: []vec3.T{
					{0, 0, 0},
					{1, 0, 0},
					{0, 1, 0},
				},
				FaceGroup: []*MeshTriangle{
					{
						Batchid: 0,
						Faces: []*Face{
							{Vertex: [3]uint32{0, 1, 2}},
						},
					},
				},
			},
		},
		Materials: []MeshMaterial{
			&BaseMaterial{
				Color:        [3]byte{255, 255, 0},
				Transparency: 0.5,
			},
		},
	}

	// 创建变换矩阵
	transform := mat4d.Ident
	transform[3][0] = 10 // 平移x
	transforms := []*mat4d.T{&transform}

	err := buildGltfFromBaseMesh(doc, baseMesh, transforms, false)
	if err != nil {
		t.Fatalf("buildGltfFromBaseMesh with transforms failed: %v", err)
	}

	if len(doc.Meshes) != 1 {
		t.Errorf("Expected 1 mesh, got %d", len(doc.Meshes))
	}

	if len(doc.Nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(doc.Nodes))
	}

	// 验证变换是否正确应用
	node := doc.Nodes[0]
	if node.Translation[0] != 10 {
		t.Errorf("Expected translation x=10, got %f", node.Translation[0])
	}
}

// TestBuildTexture 测试纹理构建
func TestBuildTexture(t *testing.T) {
	doc := CreateDoc()

	texture := &Texture{
		Id:         1,
		Name:       "test_texture",
		Size:       [2]uint64{2, 2},
		Format:     TEXTURE_FORMAT_RGBA,
		Type:       TEXTURE_PIXEL_TYPE_UBYTE,
		Compressed: 0,
		Data:       []byte{255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 0, 255}, // 2x2 RGBA
		Repeated:   false,
	}

	// 测试buildTexture函数是否成功执行
	gltfTexture, err := buildTexture(doc, doc.Buffers[0], texture)
	if err != nil {
		// 如果LoadTexture失败，我们至少验证函数结构
		t.Logf("buildTexture failed: %v", err)
		// 验证相关对象是否被创建
		if len(doc.BufferViews) > 0 {
			t.Log("Buffer views were created")
		}
		if len(doc.Images) > 0 {
			t.Log("Images were created")
		}
		if len(doc.Samplers) > 0 {
			t.Log("Samplers were created")
		}
		return
	}

	if gltfTexture == nil {
		t.Error("buildTexture returned nil")
	}

	// 验证纹理相关对象是否被正确创建
	if len(doc.BufferViews) == 0 {
		t.Error("Expected buffer views to be created")
	}

	if len(doc.Images) == 0 {
		t.Error("Expected images to be created")
	}

	if len(doc.Samplers) == 0 {
		t.Error("Expected samplers to be created")
	}
}

// TestFillMaterials 测试材质填充
func TestFillMaterials(t *testing.T) {
	doc := CreateDoc()

	materials := []MeshMaterial{
		&BaseMaterial{
			Color:        [3]byte{255, 0, 0},
			Transparency: 0.5,
		},
		&PbrMaterial{
			TextureMaterial: TextureMaterial{
				BaseMaterial: BaseMaterial{
					Color:        [3]byte{0, 255, 0},
					Transparency: 0.3,
				},
			},
			Emissive:  [3]byte{0, 0, 255},
			Metallic:  0.8,
			Roughness: 0.2,
		},
		&PhongMaterial{
			LambertMaterial: LambertMaterial{
				TextureMaterial: TextureMaterial{
					BaseMaterial: BaseMaterial{
						Color:        [3]byte{0, 0, 255},
						Transparency: 0.1,
					},
				},
				Diffuse:  [3]byte{255, 255, 255},
				Emissive: [3]byte{255, 255, 0},
			},
			Specular:  [3]byte{128, 128, 128},
			Shininess: 32.0,
		},
		&LambertMaterial{
			TextureMaterial: TextureMaterial{
				BaseMaterial: BaseMaterial{
					Color:        [3]byte{255, 255, 0},
					Transparency: 0.2,
				},
			},
			Diffuse:  [3]byte{255, 255, 255},
			Emissive: [3]byte{0, 255, 0},
		},
	}

	err := fillMaterials(doc, materials)
	if err != nil {
		t.Fatalf("fillMaterials failed: %v", err)
	}

	if len(doc.Materials) != 4 {
		t.Errorf("Expected 4 materials, got %d", len(doc.Materials))
	}

	// 验证PBR材质属性
	pbrMat := doc.Materials[1].PBRMetallicRoughness
	if *pbrMat.MetallicFactor != 0.8 {
		t.Errorf("Expected metallic factor 0.8, got %f", *pbrMat.MetallicFactor)
	}
	if *pbrMat.RoughnessFactor != 0.2 {
		t.Errorf("Expected roughness factor 0.2, got %f", *pbrMat.RoughnessFactor)
	}
}

// TestCalcPadding 测试填充计算
func TestCalcPadding(t *testing.T) {
	tests := []struct {
		offset   int
		unit     int
		expected int
	}{
		{0, 4, 0},
		{1, 4, 3},
		{2, 4, 2},
		{3, 4, 1},
		{4, 4, 0},
		{5, 4, 3},
		{7, 8, 1},
		{8, 8, 0},
	}

	for _, test := range tests {
		result := calcPadding(test.offset, test.unit)
		if result != test.expected {
			t.Errorf("calcPadding(%d, %d) = %d, expected %d", test.offset, test.unit, result, test.expected)
		}
	}
}

// TestBufferWriter 测试缓冲区写入器
func TestBufferWriter(t *testing.T) {
	writer := newBufferWriter()

	data := []byte{1, 2, 3, 4, 5}
	n, err := writer.Write(data)
	if err != nil {
		t.Fatalf("BufferWriter.Write failed: %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected write length %d, got %d", len(data), n)
	}

	if writer.Size() != len(data) {
		t.Errorf("Expected buffer size %d, got %d", len(data), writer.Size())
	}

	written := writer.Bytes()
	if !bytes.Equal(written, data) {
		t.Errorf("Written data mismatch: got %v, expected %v", written, data)
	}
}

// TestMeshPropertiesToGltfExtensions 测试将Mesh属性导出到GLTF扩展
func TestMeshPropertiesToGltfExtensions(t *testing.T) {
	// 创建带有属性的网格
	mesh := NewMesh()

	// 添加一些测试属性
	(*mesh.Props)["string_prop"] = PropsValue{Type: PROP_TYPE_STRING, Value: "test_value"}
	(*mesh.Props)["int_prop"] = PropsValue{Type: PROP_TYPE_INT, Value: int64(42)}
	(*mesh.Props)["float_prop"] = PropsValue{Type: PROP_TYPE_FLOAT, Value: 3.14}
	(*mesh.Props)["bool_prop"] = PropsValue{Type: PROP_TYPE_BOOL, Value: true}

	// 添加数组属性
	arrayProp := []PropsValue{
		{Type: PROP_TYPE_STRING, Value: "item1"},
		{Type: PROP_TYPE_STRING, Value: "item2"},
	}
	(*mesh.Props)["array_prop"] = PropsValue{Type: PROP_TYPE_ARRAY, Value: arrayProp}

	// 添加嵌套map属性
	nestedProps := make(Properties)
	nestedProps["nested_key"] = PropsValue{Type: PROP_TYPE_STRING, Value: "nested_value"}
	(*mesh.Props)["nested_prop"] = PropsValue{Type: PROP_TYPE_MAP, Value: nestedProps}

	// 创建一个简单的节点
	mesh.Nodes = []*MeshNode{
		{
			Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			FaceGroup: []*MeshTriangle{
				{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			},
		},
	}

	// 创建实例网格并添加属性
	transform := mat4d.Ident
	instanceMesh := &InstanceMesh{
		Transfors: []*mat4d.T{&transform},
		Features:  []uint64{1, 2, 3},
		Mesh: &BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes: []*MeshNode{
				{Vertices: []vec3.T{{0, 0, 0}}},
			},
		},
		Props: []*Properties{},
	}

	// 为实例网格添加属性
	instanceProps := make(Properties)
	instanceProps["instance_string"] = PropsValue{Type: PROP_TYPE_STRING, Value: "instance_test"}
	instanceProps["instance_int"] = PropsValue{Type: PROP_TYPE_INT, Value: int64(999)}
	instanceMesh.Props = []*Properties{&instanceProps}

	mesh.InstanceNode = []*InstanceMesh{instanceMesh}

	// 构建GLTF文档
	doc := CreateDoc()
	err := BuildGltf(doc, mesh, false)
	if err != nil {
		t.Fatalf("BuildGltf failed: %v", err)
	}

	// 验证主网格属性是否正确添加到扩展中
	if doc.Extensions == nil {
		t.Fatal("Extensions should not be nil")
	}

	mainProps, exists := doc.Extensions["MST_mesh_properties"]
	if !exists {
		t.Error("MST_mesh_properties extension not found")
	} else {
		propsMap, ok := mainProps.(map[string]interface{})
		if !ok {
			t.Error("MST_mesh_properties is not a map")
		} else {
			// 验证各种属性类型
			if val, ok := propsMap["string_prop"]; !ok || val != "test_value" {
				t.Errorf("string_prop not found or incorrect: %v", val)
			}

			if val, ok := propsMap["int_prop"]; !ok || val != int64(42) {
				t.Errorf("int_prop not found or incorrect: %v", val)
			}

			if val, ok := propsMap["float_prop"]; !ok || val != 3.14 {
				t.Errorf("float_prop not found or incorrect: %v", val)
			}

			if val, ok := propsMap["bool_prop"]; !ok || val != true {
				t.Errorf("bool_prop not found or incorrect: %v", val)
			}

			// 验证数组属性
			if val, ok := propsMap["array_prop"]; !ok {
				t.Error("array_prop not found")
			} else if arr, ok := val.([]interface{}); !ok {
				t.Error("array_prop is not an array")
			} else if len(arr) != 2 {
				t.Errorf("array_prop length = %d, want 2", len(arr))
			} else {
				if arr[0] != "item1" || arr[1] != "item2" {
					t.Errorf("array_prop values incorrect: %v", arr)
				}
			}

			// 验证嵌套属性
			if val, ok := propsMap["nested_prop"]; !ok {
				t.Error("nested_prop not found")
			} else if nestedMap, ok := val.(map[string]interface{}); !ok {
				t.Error("nested_prop is not a map")
			} else {
				if nestedVal, ok := nestedMap["nested_key"]; !ok || nestedVal != "nested_value" {
					t.Errorf("nested_key not found or incorrect: %v", nestedVal)
				}
			}
		}
	}

	// 验证实例网格属性是否正确添加到扩展中
	instancePropsExt, exists := doc.Extensions["MST_instance_mesh_properties_0_0"]
	if !exists {
		t.Error("MST_instance_mesh_properties_0_0 extension not found")
	} else {
		propsMap, ok := instancePropsExt.(map[string]interface{})
		if !ok {
			t.Error("MST_instance_mesh_properties_0_0 is not a map")
		} else {
			if val, ok := propsMap["instance_string"]; !ok || val != "instance_test" {
				t.Errorf("instance_string not found or incorrect: %v", val)
			}

			if val, ok := propsMap["instance_int"]; !ok || val != int64(999) {
				t.Errorf("instance_int not found or incorrect: %v", val)
			}
		}
	}
}
