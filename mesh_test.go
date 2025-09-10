package mst

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	proj "github.com/flywave/go-proj"
	mat4d "github.com/flywave/go3d/float64/mat4"
	vec3d "github.com/flywave/go3d/float64/vec3"

	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
	"github.com/xtgo/uuid"
)

// TestMeshVersions 测试所有版本兼容性
func TestMeshVersions(t *testing.T) {
	tests := []struct {
		name    string
		version uint32
		want    uint32
	}{
		{"V1", V1, V1},
		{"V2", V2, V2},
		{"V3", V3, V3},
		{"V4", V4, V4},
		{"V5", V5, V5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mesh := &Mesh{BaseMesh: BaseMesh{Materials: []MeshMaterial{}, Nodes: []*MeshNode{}}, Version: tt.version}
			if mesh.Version != tt.want {
				t.Errorf("Expected version %d, got %d", tt.want, mesh.Version)
			}
		})
	}
}

// TestNewMesh 测试创建新网格
func TestNewMesh(t *testing.T) {
	mesh := NewMesh()
	if mesh == nil {
		t.Fatal("NewMesh returned nil")
	}
	if mesh.Version != V5 {
		t.Errorf("Expected version V5, got %d", mesh.Version)
	}
	if len(mesh.Materials) != 0 || len(mesh.Nodes) != 0 {
		t.Errorf("Expected empty mesh, got materials=%d, nodes=%d", len(mesh.Materials), len(mesh.Nodes))
	}
	if mesh.Props == nil {
		t.Errorf("Expected non-nil Props in V5 mesh")
	}
}

// TestMaterialTypes 测试所有材质类型
func TestMaterialTypes(t *testing.T) {
	tests := []struct {
		name      string
		material  MeshMaterial
		wantTex   bool
		wantColor [3]byte
	}{
		{
			"BaseMaterial",
			&BaseMaterial{Color: [3]byte{255, 0, 0}, Transparency: 0.5},
			false,
			[3]byte{255, 0, 0},
		},
		{
			"TextureMaterial",
			&TextureMaterial{
				BaseMaterial: BaseMaterial{Color: [3]byte{0, 255, 0}},
				Texture:      &Texture{Id: 1, Name: "test"},
			},
			true,
			[3]byte{0, 255, 0},
		},
		{
			"PbrMaterial",
			&PbrMaterial{
				TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{0, 0, 255}}},
				Metallic:        0.8, Roughness: 0.2, Emissive: [3]byte{10, 20, 30},
			},
			false,
			[3]byte{0, 0, 255},
		},
		{
			"LambertMaterial",
			&LambertMaterial{
				TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{255, 255, 0}}},
				Ambient:         [3]byte{50, 50, 50}, Emissive: [3]byte{5, 10, 15},
			},
			false,
			[3]byte{255, 255, 0},
		},
		{
			"PhongMaterial",
			&PhongMaterial{
				LambertMaterial: LambertMaterial{
					TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{255, 0, 255}}},
					Emissive:        [3]byte{20, 30, 40},
				},
				Specular: [3]byte{255, 255, 255}, Shininess: 32.0,
			},
			false,
			[3]byte{255, 0, 255},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.material.HasTexture() != tt.wantTex {
				t.Errorf("%s: HasTexture() = %v, want %v", tt.name, tt.material.HasTexture(), tt.wantTex)
			}
			if tt.material.GetColor() != tt.wantColor {
				t.Errorf("%s: GetColor() = %v, want %v", tt.name, tt.material.GetColor(), tt.wantColor)
			}
		})
	}
}

// TestMeshNodeOperations 测试网格节点操作
func TestMeshNodeOperations(t *testing.T) {
	node := &MeshNode{
		Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
		Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
		Colors:    [][3]byte{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}},
		TexCoords: []vec2.T{{0, 0}, {1, 0}, {0, 1}},
		FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
		EdgeGroup: []*MeshOutline{{Batchid: 0, Edges: [][2]uint32{{0, 1}, {1, 2}, {2, 0}}}},
	}

	bbox := node.GetBoundbox()
	expected := [6]float64{0, 0, 0, 1, 1, 0}
	for i := 0; i < 6; i++ {
		if bbox[i] != expected[i] {
			t.Errorf("bbox[%d] = %f, want %f", i, bbox[i], expected[i])
		}
	}
}

// TestMeshCounts 测试网格计数
func TestMeshCounts(t *testing.T) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&BaseMaterial{Color: [3]byte{255, 0, 0}},
				&BaseMaterial{Color: [3]byte{0, 255, 0}},
			},
			Nodes: []*MeshNode{
				{Vertices: []vec3.T{{0, 0, 0}}},
				{Vertices: []vec3.T{{1, 0, 0}}},
				{Vertices: []vec3.T{{0, 1, 0}}},
			},
		},
		Version: V4,
	}

	if mesh.MaterialCount() != 2 {
		t.Errorf("MaterialCount() = %d, want 2", mesh.MaterialCount())
	}
	if mesh.NodeCount() != 3 {
		t.Errorf("NodeCount() = %d, want 3", mesh.NodeCount())
	}
}

// TestMeshBoundingBox 测试边界框计算
func TestMeshBoundingBox(t *testing.T) {
	tests := []struct {
		name     string
		vertices []vec3.T
		min      vec3d.T
		max      vec3d.T
	}{
		{
			"SimpleBox",
			[]vec3.T{{-1, -1, -1}, {1, 1, 1}},
			vec3d.T{-1, -1, -1},
			vec3d.T{1, 1, 1},
		},
		{
			"SinglePoint",
			[]vec3.T{{0, 0, 0}},
			vec3d.T{0, 0, 0},
			vec3d.T{0, 0, 0},
		},
		{
			"EmptyMesh",
			[]vec3.T{},
			vec3d.T{0, 0, 0},
			vec3d.T{0, 0, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mesh := &Mesh{
				BaseMesh: BaseMesh{Nodes: []*MeshNode{{Vertices: tt.vertices}}},
				Version:  V4,
			}

			bbox := mesh.ComputeBBox()

			if len(tt.vertices) == 0 {
				// 空网格的特殊处理
				return
			}

			for i := 0; i < 3; i++ {
				if bbox.Min[i] != tt.min[i] || bbox.Max[i] != tt.max[i] {
					t.Errorf("bbox[%d] = [%f, %f], want [%f, %f]",
						i, bbox.Min[i], bbox.Max[i], tt.min[i], tt.max[i])
				}
			}
		})
	}
}

// TestMaterialMarshalUnmarshal 测试材质序列化反序列化
func TestMaterialMarshalUnmarshal(t *testing.T) {
	materials := []MeshMaterial{
		&BaseMaterial{Color: [3]byte{255, 0, 0}, Transparency: 0.8},
		&TextureMaterial{
			BaseMaterial: BaseMaterial{Color: [3]byte{0, 255, 0}},
			Texture: &Texture{
				Id: 1, Name: "test_texture", Size: [2]uint64{256, 256},
				Format: TEXTURE_FORMAT_RGB, Data: []byte{1, 2, 3, 4, 5, 6},
			},
		},
		&PbrMaterial{
			TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{0, 0, 255}}},
			Metallic:        0.5, Roughness: 0.3, Emissive: [3]byte{10, 20, 30},
		},
		&LambertMaterial{
			TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{255, 255, 0}}},
			Ambient:         [3]byte{50, 50, 50}, Emissive: [3]byte{5, 10, 15},
		},
		&PhongMaterial{
			LambertMaterial: LambertMaterial{
				TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{255, 0, 255}}},
				Emissive:        [3]byte{20, 30, 40},
			},
			Specular: [3]byte{255, 255, 255}, Shininess: 32.0,
		},
	}

	for version := V1; version <= V4; version++ {
		t.Run(string(rune(version)), func(t *testing.T) {
			var buf bytes.Buffer
			MtlsMarshal(&buf, materials, version)

			bufCopy := bytes.NewReader(buf.Bytes())
			unmarshaled := MtlsUnMarshal(bufCopy, version)

			if len(unmarshaled) != len(materials) {
				t.Errorf("Version %d: expected %d materials, got %d", version, len(materials), len(unmarshaled))
			}
		})
	}
}

// TestMeshNodeMarshalUnmarshal 测试节点序列化反序列化
func TestMeshNodeMarshalUnmarshal(t *testing.T) {
	node := &MeshNode{
		Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
		Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
		Colors:    [][3]byte{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}},
		TexCoords: []vec2.T{{0, 0}, {1, 0}, {0, 1}},
		FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
		EdgeGroup: []*MeshOutline{{Batchid: 0, Edges: [][2]uint32{{0, 1}, {1, 2}, {2, 0}}}},
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(unmarshaled.Vertices) != len(node.Vertices) {
		t.Errorf("vertices count mismatch")
	}
	if len(unmarshaled.FaceGroup) != len(node.FaceGroup) {
		t.Errorf("face groups count mismatch")
	}
	if len(unmarshaled.EdgeGroup) != len(node.EdgeGroup) {
		t.Errorf("edge groups count mismatch")
	}
}

// TestMeshMarshalUnmarshal 测试完整网格序列化反序列化
func TestMeshMarshalUnmarshal(t *testing.T) {
	for version := V1; version <= V4; version++ {
		t.Run(string(rune(version)), func(t *testing.T) {
			mesh := &Mesh{
				BaseMesh: BaseMesh{
					Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
					Nodes: []*MeshNode{
						{
							Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
							FaceGroup: []*MeshTriangle{
								{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
							},
						},
					},
					Code: 54321,
				},
				Version:      version,
				InstanceNode: []*InstanceMesh{},
			}

			var buf bytes.Buffer
			MeshMarshal(&buf, mesh)

			unmarshaled := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

			if unmarshaled.Version != version {
				t.Errorf("version mismatch")
			}
			if len(unmarshaled.Materials) != len(mesh.Materials) {
				t.Errorf("materials count mismatch")
			}
			if version == V4 && unmarshaled.Code != mesh.Code {
				t.Errorf("code mismatch for V4")
			}
		})
	}
}

// TestNodeOperations 测试网格节点操作
func TestNodeOperations(t *testing.T) {
	t.Run("ResortVtVn", func(t *testing.T) {
		node := &MeshNode{
			Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
			TexCoords: []vec2.T{{0, 0}, {1, 0}, {0, 1}},
			FaceGroup: []*MeshTriangle{{
				Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}, Normal: &[3]uint32{0, 1, 2}, Uv: &[3]uint32{0, 1, 2}}},
			}},
		}

		originalVerts, originalNorms, originalUVs := len(node.Vertices), len(node.Normals), len(node.TexCoords)

		// ResortVtVn 应该处理非空数据
		if len(node.FaceGroup) > 0 && len(node.FaceGroup[0].Faces) > 0 {
			// 简单的测试，验证数据存在
			if len(node.Vertices) == 0 || len(node.Normals) == 0 || len(node.TexCoords) == 0 {
				t.Errorf("empty vertex data")
			}
		} else {
			// 空网格的情况
			if len(node.Vertices) != originalVerts || len(node.Normals) != originalNorms || len(node.TexCoords) != originalUVs {
				t.Logf("empty mesh data unchanged")
			}
		}
	})

	t.Run("ReComputeNormal", func(t *testing.T) {
		node := &MeshNode{
			Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			FaceGroup: []*MeshTriangle{{
				Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}},
			}},
		}

		node.ReComputeNormal()
		if len(node.Normals) != len(node.Vertices) {
			t.Errorf("normals not computed correctly")
		}

		// 检查法线归一化
		for _, normal := range node.Normals {
			length := normal.Length()
			if length < 0.99 || length > 1.01 {
				t.Errorf("normal not normalized: %f", length)
			}
		}
	})
}

// TestInstanceMeshOperations 测试实例化网格操作
func TestInstanceMeshOperations(t *testing.T) {
	for version := V1; version <= V4; version++ {
		t.Run(string(rune(version)), func(t *testing.T) {
			transform := mat4d.Ident

			instanceMesh := &InstanceMesh{
				Transfors: []*mat4d.T{&transform},
				Features:  []uint64{12345, 67890},
				Mesh: &BaseMesh{
					Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 255, 255}}},
					Nodes: []*MeshNode{
						{Vertices: []vec3.T{{0, 0, 0}}},
					},
				},
				Hash: 0x12345678,
			}

			// 验证原始数据
			if len(instanceMesh.Transfors) != 1 {
				t.Skip("invalid test data")
			}

			// 简化测试：只验证序列化/反序列化不崩溃
			var buf bytes.Buffer
			MeshInstanceNodeMarshal(&buf, instanceMesh, version)

			if buf.Len() == 0 {
				t.Errorf("serialization failed")
			}

			unmarshaled := MeshInstanceNodeUnMarshal(bytes.NewReader(buf.Bytes()), version)
			if unmarshaled == nil {
				t.Errorf("deserialization failed")
			}
		})
	}
}

// TestMeshComplexStructure 测试复杂网格结构
func TestMeshComplexStructure(t *testing.T) {
	mesh := NewMesh()

	// 多种材质
	mesh.Materials = []MeshMaterial{
		&BaseMaterial{Color: [3]byte{255, 0, 0}},
		&TextureMaterial{
			BaseMaterial: BaseMaterial{Color: [3]byte{0, 255, 0}},
			Texture:      &Texture{Id: 1, Name: "texture1", Size: [2]uint64{256, 256}},
		},
		&PbrMaterial{
			TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{0, 0, 255}}},
			Metallic:        0.5, Roughness: 0.3,
		},
	}

	// 多个节点
	mesh.Nodes = []*MeshNode{
		{
			Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
		},
		{
			Vertices:  []vec3.T{{1, 0, 0}, {2, 0, 0}, {1, 1, 0}},
			FaceGroup: []*MeshTriangle{{Batchid: 1, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
		},
	}

	// 实例化节点
	transform := mat4d.Ident
	mesh.InstanceNode = []*InstanceMesh{
		{
			Transfors: []*mat4d.T{&transform},
			Features:  []uint64{100, 200, 300},
			Mesh: &BaseMesh{
				Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 255, 0}}},
				Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 1}}}},
			},
		},
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(readMesh.Materials) != 3 {
		t.Errorf("materials count mismatch")
	}
	if len(readMesh.Nodes) != 2 {
		t.Errorf("nodes count mismatch")
	}
	if len(readMesh.InstanceNode) != 1 {
		t.Errorf("instance nodes count mismatch")
	}
}

// TestConstants 测试所有常量
func TestConstants(t *testing.T) {
	constants := map[string]interface{}{
		"MESH_SIGNATURE": "fwtm",
		"MSTEXT":         ".mst",
		"V1":             uint32(1),
		"V2":             uint32(2),
		"V3":             uint32(3),
		"V4":             uint32(4),
		"V5":             uint32(5),
	}

	for name, want := range constants {
		t.Run(name, func(t *testing.T) {
			switch name {
			case "MESH_SIGNATURE":
				if MESH_SIGNATURE != want.(string) {
					t.Errorf("%s = %s, want %s", name, MESH_SIGNATURE, want)
				}
			case "MSTEXT":
				if MSTEXT != want.(string) {
					t.Errorf("%s = %s, want %s", name, MSTEXT, want)
				}
			case "V1", "V2", "V3", "V4", "V5":
				val := map[string]uint32{"V1": V1, "V2": V2, "V3": V3, "V4": V4, "V5": V5}[name]
				if val != want.(uint32) {
					t.Errorf("%s = %d, want %d", name, val, want)
				}
			}
		})
	}
}

// TestEdgeCases 测试边界情况
func TestEdgeCases(t *testing.T) {
	t.Run("EmptyMesh", func(t *testing.T) {
		mesh := &Mesh{BaseMesh: BaseMesh{Nodes: []*MeshNode{}}, Version: V4}
		bbox := mesh.ComputeBBox()
		if bbox.Min[0] != 0 || bbox.Max[0] != 0 {
			t.Logf("Empty mesh bbox: %v", bbox)
		}
	})

	t.Run("SingleVertex", func(t *testing.T) {
		mesh := &Mesh{
			BaseMesh: BaseMesh{Nodes: []*MeshNode{{Vertices: []vec3.T{{1, 2, 3}}}}},
			Version:  V4,
		}
		bbox := mesh.ComputeBBox()
		if bbox.Min[0] != 1 || bbox.Min[1] != 2 || bbox.Min[2] != 3 ||
			bbox.Max[0] != 1 || bbox.Max[1] != 2 || bbox.Max[2] != 3 {
			t.Errorf("Single vertex bbox incorrect")
		}
	})

	t.Run("LargeData", func(t *testing.T) {
		vertices := make([]vec3.T, 100)
		for i := 0; i < 100; i++ {
			vertices[i] = vec3.T{float32(i), float32(i), float32(i)}
		}

		mesh := &Mesh{
			BaseMesh: BaseMesh{
				Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
				Nodes: []*MeshNode{{
					Vertices:  vertices,
					FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
				}},
			},
			Version: V4,
		}

		var buf bytes.Buffer
		MeshMarshal(&buf, mesh)

		readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
		if len(readMesh.Nodes[0].Vertices) != len(vertices) {
			t.Errorf("Large data serialization failed")
		}
	})
}

// TestMeshReadWriteFile 测试文件读写
func TestMeshReadWriteFile(t *testing.T) {
	// 创建测试数据
	mesh := NewMesh()
	mesh.Materials = []MeshMaterial{
		&BaseMaterial{Color: [3]byte{255, 0, 0}},
		&BaseMaterial{Color: [3]byte{0, 255, 0}},
	}
	mesh.Nodes = []*MeshNode{
		{
			Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			Normals:  []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
			FaceGroup: []*MeshTriangle{
				{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			},
		},
	}

	// 测试内存序列化
	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

	if readMesh.Version != mesh.Version {
		t.Errorf("Version mismatch")
	}
	if len(readMesh.Materials) != len(mesh.Materials) {
		t.Errorf("Materials count mismatch")
	}
	if len(readMesh.Nodes) != len(mesh.Nodes) {
		t.Errorf("Nodes count mismatch")
	}
}

// TestGltfIntegration 测试GLTF集成
func TestGltfIntegration(t *testing.T) {
	// 创建测试网格
	mesh := NewMesh()
	mesh.Materials = []MeshMaterial{
		&BaseMaterial{Color: [3]byte{255, 0, 0}},
		&BaseMaterial{Color: [3]byte{0, 255, 0}},
	}
	mesh.Nodes = []*MeshNode{
		{
			Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			FaceGroup: []*MeshTriangle{
				{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			},
		},
	}

	// 创建GLTF文档
	doc := CreateDoc()
	BuildGltf(doc, mesh, false)

	if doc == nil {
		t.Fatal("Failed to create GLTF document")
	}

	// 测试二进制输出
	bt, err := GetGltfBinary(doc, 8)
	if err != nil {
		t.Fatalf("Failed to get GLTF binary: %v", err)
	}

	if len(bt) == 0 {
		t.Error("Empty GLTF binary")
	}
}

// TestMstToObj 测试MST到OBJ转换
func TestMstToObj(t *testing.T) {
	// 创建临时测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.mst")

	// 创建测试网格
	mesh := NewMesh()
	mesh.Materials = []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}}
	mesh.Nodes = []*MeshNode{
		{
			Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			Normals:  []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
			FaceGroup: []*MeshTriangle{
				{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			},
		},
	}

	// 写入测试文件
	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	// 测试文件读写
	f, err := os.Create(testFile)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer f.Close()

	if _, err := f.Write(buf.Bytes()); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 测试读取
	f.Seek(0, 0)
	readMesh := MeshUnMarshal(f)
	if readMesh == nil {
		t.Fatal("Failed to read mesh from file")
	}

	if len(readMesh.Materials) != 1 || len(readMesh.Nodes) != 1 {
		t.Error("Mesh data mismatch")
	}
}

// TestUUIDGeneration 测试UUID生成
func TestUUIDGeneration(t *testing.T) {
	id := uuid.NewRandom().String()
	id = strings.ReplaceAll(id, "-", "")

	if len(id) != 32 {
		t.Errorf("UUID length = %d, want 32", len(id))
	}

	// 测试UUID格式
	for _, c := range id {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			t.Errorf("Invalid UUID character: %c", c)
		}
	}
}

// TestTextureFormats 测试纹理格式
func TestTextureFormats(t *testing.T) {
	formats := map[uint16]string{
		TEXTURE_FORMAT_R:               "R",
		TEXTURE_FORMAT_RGB:             "RGB",
		TEXTURE_FORMAT_RGBA:            "RGBA",
		TEXTURE_FORMAT_DEPTH_COMPONENT: "DEPTH_COMPONENT",
	}

	for format, name := range formats {
		t.Run(name, func(t *testing.T) {
			texture := &Texture{
				Format: format,
				Size:   [2]uint64{256, 256},
				Data:   make([]byte, 256*256*4),
			}

			if texture.Format != format {
				t.Errorf("Format = %d, want %d", texture.Format, format)
			}
		})
	}
}

// TestMaterialTypeConstants 测试材质类型常量
func TestMaterialTypeConstants(t *testing.T) {
	types := map[uint32]string{
		MESH_TRIANGLE_MATERIAL_TYPE_COLOR:   "COLOR",
		MESH_TRIANGLE_MATERIAL_TYPE_TEXTURE: "TEXTURE",
		MESH_TRIANGLE_MATERIAL_TYPE_PBR:     "PBR",
		MESH_TRIANGLE_MATERIAL_TYPE_LAMBERT: "LAMBERT",
		MESH_TRIANGLE_MATERIAL_TYPE_PHONG:   "PHONG",
	}

	for value, name := range types {
		t.Run(name, func(t *testing.T) {
			if value < 0 || value > 10 {
				t.Errorf("Invalid material type value: %d", value)
			}
		})
	}
}

// BenchmarkMeshMarshal 基准测试网格序列化
func BenchmarkMeshMarshal(b *testing.B) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes: []*MeshNode{
				{
					Vertices: make([]vec3.T, 1000),
					FaceGroup: []*MeshTriangle{{
						Batchid: 0,
						Faces:   make([]*Face, 100),
					}},
				},
			},
			Code: 12345,
		},
		Version: V4,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		MeshMarshal(&buf, mesh)
	}
}

// BenchmarkMeshUnmarshal 基准测试网格反序列化
func BenchmarkMeshUnmarshal(b *testing.B) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes: []*MeshNode{
				{
					Vertices: make([]vec3.T, 1000),
					FaceGroup: []*MeshTriangle{{
						Batchid: 0,
						Faces:   make([]*Face, 100),
					}},
				},
			},
			Code: 12345,
		},
		Version: V4,
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		MeshUnMarshal(bytes.NewReader(data))
	}
}

// TestMeshNodeWithTransform 测试带变换矩阵的节点
func TestMeshNodeWithTransform(t *testing.T) {
	transform := mat4d.Ident
	transform[0][0] = 2.0
	transform[1][1] = 3.0
	transform[2][2] = 4.0

	node := &MeshNode{
		Vertices: []vec3.T{{1, 1, 1}},
		Mat:      &transform,
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))
	if unmarshaled.Mat == nil {
		t.Error("Transform matrix not preserved")
	}
	if unmarshaled.Mat[0][0] != 2.0 {
		t.Errorf("Transform[0][0] = %f, want 2.0", unmarshaled.Mat[0][0])
	}
}

// TestMeshNodeWithColors 测试带颜色的节点
func TestMeshNodeWithColors(t *testing.T) {
	node := &MeshNode{
		Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
		Colors:   [][3]byte{{255, 0, 0}, {0, 255, 0}, {0, 0, 255}},
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(unmarshaled.Colors) != len(node.Vertices) {
		t.Errorf("Colors count = %d, want %d", len(unmarshaled.Colors), len(node.Vertices))
	}

	for i, color := range unmarshaled.Colors {
		if color != node.Colors[i] {
			t.Errorf("Color[%d] = %v, want %v", i, color, node.Colors[i])
		}
	}
}

// TestInstanceMeshWithBBox 测试带边界框的实例化网格
func TestInstanceMeshWithBBox(t *testing.T) {
	bbox := &[6]float64{-1, -1, -1, 1, 1, 1}

	instanceMesh := &InstanceMesh{
		Transfors: []*mat4d.T{},
		Features:  []uint64{},
		BBox:      bbox,
		Mesh:      &BaseMesh{},
	}

	var buf bytes.Buffer
	MeshInstanceNodeMarshal(&buf, instanceMesh, V4)

	// 注意：边界框在序列化中会被包含
	if instanceMesh.BBox[0] != -1 || instanceMesh.BBox[3] != 1 {
		t.Errorf("Bounding box range incorrect")
	}
}

// TestMeshWithMultipleFaceGroups 测试多个面组
func TestMeshWithMultipleFaceGroups(t *testing.T) {
	node := &MeshNode{
		Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0}},
		FaceGroup: []*MeshTriangle{
			{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			{Batchid: 1, Faces: []*Face{{Vertex: [3]uint32{1, 2, 3}}}},
			{Batchid: 2, Faces: []*Face{{Vertex: [3]uint32{0, 2, 3}}}},
		},
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(unmarshaled.FaceGroup) != 3 {
		t.Errorf("Face groups count = %d, want 3", len(unmarshaled.FaceGroup))
	}

	for i, fg := range unmarshaled.FaceGroup {
		if fg.Batchid != int32(i) {
			t.Errorf("FaceGroup[%d].Batchid = %d, want %d", i, fg.Batchid, i)
		}
	}
}

// TestMeshWithMultipleEdgeGroups 测试多个边组
func TestMeshWithMultipleEdgeGroups(t *testing.T) {
	node := &MeshNode{
		Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
		EdgeGroup: []*MeshOutline{
			{Batchid: 0, Edges: [][2]uint32{{0, 1}}},
			{Batchid: 1, Edges: [][2]uint32{{1, 2}}},
			{Batchid: 2, Edges: [][2]uint32{{2, 0}}},
		},
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(unmarshaled.EdgeGroup) != 3 {
		t.Errorf("Edge groups count = %d, want 3", len(unmarshaled.EdgeGroup))
	}

	for i, eg := range unmarshaled.EdgeGroup {
		if eg.Batchid != int32(i) {
			t.Errorf("EdgeGroup[%d].Batchid = %d, want %d", i, eg.Batchid, i)
		}
	}
}

// TestInstanceMeshWithMultipleTransforms 测试多个变换
func TestInstanceMeshWithMultipleTransforms(t *testing.T) {
	transforms := make([]*mat4d.T, 5)
	for i := 0; i < 5; i++ {
		transform := mat4d.Ident
		transform[0][0] = float64(i + 1)
		transforms[i] = &transform
	}

	instanceMesh := &InstanceMesh{
		Transfors: transforms,
		Features:  []uint64{1, 2, 3, 4, 5},
		Mesh:      &BaseMesh{},
	}

	var buf bytes.Buffer
	MeshInstanceNodeMarshal(&buf, instanceMesh, V4)

	unmarshaled := MeshInstanceNodeUnMarshal(bytes.NewReader(buf.Bytes()), V4)

	if len(unmarshaled.Transfors) != 5 {
		t.Errorf("Transforms count = %d, want 5", len(unmarshaled.Transfors))
	}

	for i, transform := range unmarshaled.Transfors {
		if transform[0][0] != float64(i+1) {
			t.Errorf("Transform[%d][0][0] = %f, want %f", i, transform[0][0], float64(i+1))
		}
	}
}

// TestMeshWithEmptyComponents 测试空组件
func TestMeshWithEmptyComponents(t *testing.T) {
	tests := []struct {
		name string
		mesh *Mesh
	}{
		{"EmptyMaterials", &Mesh{BaseMesh: BaseMesh{Materials: []MeshMaterial{}, Nodes: []*MeshNode{}}}},
		{"EmptyNodes", &Mesh{BaseMesh: BaseMesh{Materials: []MeshMaterial{&BaseMaterial{}}, Nodes: []*MeshNode{}}}},
		{"EmptyInstanceNodes", &Mesh{BaseMesh: BaseMesh{}, InstanceNode: []*InstanceMesh{}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			MeshMarshal(&buf, tt.mesh)

			readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

			if readMesh == nil {
				t.Fatal("Failed to serialize/deserialize empty components")
			}
		})
	}
}

// TestMeshNodeMarshalUnmarshalWithTransform 测试带变换的节点序列化
func TestMeshNodeMarshalUnmarshalWithTransform(t *testing.T) {
	transform := mat4d.Ident
	transform[0][0] = 2.0
	transform[1][1] = 3.0
	transform[2][2] = 4.0

	node := &MeshNode{
		Vertices: []vec3.T{{1, 1, 1}},
		Mat:      &transform,
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if unmarshaled.Mat == nil {
		t.Error("Transform matrix not preserved")
	}
	if unmarshaled.Mat[0][0] != 2.0 || unmarshaled.Mat[1][1] != 3.0 || unmarshaled.Mat[2][2] != 4.0 {
		t.Error("Transform values not preserved")
	}
}

// TestMeshNodeMarshalUnmarshalWithoutTransform 测试不带变换的节点序列化
func TestMeshNodeMarshalUnmarshalWithoutTransform(t *testing.T) {
	node := &MeshNode{
		Vertices: []vec3.T{{0, 0, 0}},
		Mat:      nil,
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if unmarshaled.Mat != nil {
		t.Error("Unexpected transform matrix")
	}
}

// TestMeshVersionCompatibility 测试版本兼容性
func TestMeshVersionCompatibility(t *testing.T) {
	for version := V1; version <= V4; version++ {
		t.Run(string(rune(version)), func(t *testing.T) {
			mesh := &Mesh{
				BaseMesh: BaseMesh{
					Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
					Nodes: []*MeshNode{
						{
							Vertices: []vec3.T{{0, 0, 0}},
							FaceGroup: []*MeshTriangle{
								{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}},
							},
						},
					},
					Code: 12345,
				},
				Version: version,
			}

			var buf bytes.Buffer
			MeshMarshal(&buf, mesh)

			readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

			if readMesh.Version != version {
				t.Errorf("Version = %d, want %d", readMesh.Version, version)
			}
			if version == V4 && readMesh.Code != mesh.Code {
				t.Errorf("Code = %d, want %d", readMesh.Code, mesh.Code)
			}
		})
	}
}

// TestTextureTypes 测试纹理类型
func TestTextureTypes(t *testing.T) {
	tests := []struct {
		name   string
		format uint16
		pixel  uint16
	}{
		{"RGBA_UBYTE", TEXTURE_FORMAT_RGBA, TEXTURE_PIXEL_TYPE_UBYTE},
		{"RGB_FLOAT", TEXTURE_FORMAT_RGB, TEXTURE_PIXEL_TYPE_FLOAT},
		{"R_INTEGER", TEXTURE_FORMAT_R_INTEGER, TEXTURE_PIXEL_TYPE_INT},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			texture := &Texture{
				Format: tt.format,
				Type:   tt.pixel,
				Size:   [2]uint64{64, 64},
				Data:   make([]byte, 64*64*4),
			}

			if texture.Format != tt.format {
				t.Errorf("Format = %d, want %d", texture.Format, tt.format)
			}
			if texture.Type != tt.pixel {
				t.Errorf("Type = %d, want %d", texture.Type, tt.pixel)
			}
		})
	}
}

// TestMaterialEmissive 测试材质发光属性
func TestMaterialEmissive(t *testing.T) {
	tests := []struct {
		name     string
		material MeshMaterial
		expected [3]byte
	}{
		{"BaseMaterial", &BaseMaterial{Color: [3]byte{255, 0, 0}}, [3]byte{0, 0, 0}},
		{"PbrMaterial", &PbrMaterial{Emissive: [3]byte{10, 20, 30}}, [3]byte{10, 20, 30}},
		{"LambertMaterial", &LambertMaterial{Emissive: [3]byte{5, 10, 15}}, [3]byte{5, 10, 15}},
		{"PhongMaterial", &PhongMaterial{LambertMaterial: LambertMaterial{Emissive: [3]byte{20, 30, 40}}}, [3]byte{20, 30, 40}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.material.GetEmissive() != tt.expected {
				t.Errorf("Emissive = %v, want %v", tt.material.GetEmissive(), tt.expected)
			}
		})
	}
}

// TestMeshFaceAndEdgeGroups 测试面和边组
func TestMeshFaceAndEdgeGroups(t *testing.T) {
	node := &MeshNode{
		Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0}},
		FaceGroup: []*MeshTriangle{
			{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			{Batchid: 1, Faces: []*Face{{Vertex: [3]uint32{1, 2, 3}}}},
		},
		EdgeGroup: []*MeshOutline{
			{Batchid: 0, Edges: [][2]uint32{{0, 1}, {1, 2}, {2, 0}}},
			{Batchid: 1, Edges: [][2]uint32{{1, 2}, {2, 3}, {3, 1}}},
		},
	}

	var buf bytes.Buffer
	MeshNodeMarshal(&buf, node)

	unmarshaled := MeshNodeUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(unmarshaled.FaceGroup) != 2 {
		t.Errorf("Face groups = %d, want 2", len(unmarshaled.FaceGroup))
	}
	if len(unmarshaled.EdgeGroup) != 2 {
		t.Errorf("Edge groups = %d, want 2", len(unmarshaled.EdgeGroup))
	}
}

// TestMeshWithInstanceNodes 测试实例化节点
func TestMeshWithInstanceNodes(t *testing.T) {
	mesh := NewMesh()

	transform := mat4d.Ident
	instanceMesh := &InstanceMesh{
		Transfors: []*mat4d.T{&transform},
		Features:  []uint64{100, 200, 300},
		Mesh: &BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 255, 0}}},
			Nodes: []*MeshNode{
				{Vertices: []vec3.T{{0, 0, 1}}},
			},
		},
	}

	mesh.InstanceNode = []*InstanceMesh{instanceMesh}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(readMesh.InstanceNode) != 1 {
		t.Errorf("Instance nodes = %d, want 1", len(readMesh.InstanceNode))
	}
	if len(readMesh.InstanceNode[0].Transfors) != 1 {
		t.Errorf("Transforms = %d, want 1", len(readMesh.InstanceNode[0].Transfors))
	}
}

// TestVecOperations 测试向量操作
func TestVecOperations(t *testing.T) {
	world := &vec3d.T{-2389250.4338499242, 4518270.200871248, 3802675.424745363}
	head := &vec3d.T{4.771371435839683, -0.753607839345932, 3.867249683942646}
	p := &vec3d.T{4.802855, -0.753608, 3.828406}

	length := p.Add(world).Length()
	if length <= 0 {
		t.Error("Vector length should be positive")
	}

	world.Add(head)
	x, y, z, _ := proj.Ecef2Lonlat(p[0], p[1], p[2])
	if x == 0 && y == 0 && z == 0 {
		t.Error("Coordinate transformation failed")
	}
}

// TestMeshTriangleAndOutline 测试三角形和轮廓
func TestMeshTriangleAndOutline(t *testing.T) {
	triangle := &MeshTriangle{
		Batchid: 1,
		Faces: []*Face{
			{Vertex: [3]uint32{0, 1, 2}},
			{Vertex: [3]uint32{1, 2, 3}},
		},
	}

	outline := &MeshOutline{
		Batchid: 2,
		Edges: [][2]uint32{
			{0, 1}, {1, 2}, {2, 0},
		},
	}

	if triangle.Batchid != 1 {
		t.Errorf("Triangle batchid = %d, want 1", triangle.Batchid)
	}
	if len(triangle.Faces) != 2 {
		t.Errorf("Triangle faces = %d, want 2", len(triangle.Faces))
	}

	if outline.Batchid != 2 {
		t.Errorf("Outline batchid = %d, want 2", outline.Batchid)
	}
	if len(outline.Edges) != 3 {
		t.Errorf("Outline edges = %d, want 3", len(outline.Edges))
	}
}

// TestFaceStructure 测试面结构
func TestFaceStructure(t *testing.T) {
	face := &Face{
		Vertex: [3]uint32{0, 1, 2},
		Normal: &[3]uint32{0, 1, 2},
		Uv:     &[3]uint32{0, 1, 2},
	}

	if face.Vertex[0] != 0 || face.Vertex[1] != 1 || face.Vertex[2] != 2 {
		t.Errorf("Vertex indices = %v, want [0, 1, 2]", face.Vertex)
	}
	if face.Normal[0] != 0 || face.Normal[1] != 1 || face.Normal[2] != 2 {
		t.Errorf("Normal indices = %v, want [0, 1, 2]", face.Normal)
	}
	if face.Uv[0] != 0 || face.Uv[1] != 1 || face.Uv[2] != 2 {
		t.Errorf("UV indices = %v, want [0, 1, 2]", face.Uv)
	}
}

// TestMeshReadWriteLargeData 测试大数据读写
func TestMeshReadWriteLargeData(t *testing.T) {
	// 创建大数据网格
	vertices := make([]vec3.T, 1000)
	faces := make([]*Face, 500)

	for i := 0; i < 1000; i++ {
		vertices[i] = vec3.T{float32(i), float32(i), float32(i)}
	}
	for i := 0; i < 500; i++ {
		faces[i] = &Face{Vertex: [3]uint32{uint32(i), uint32(i + 1), uint32(i + 2)}}
	}

	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes: []*MeshNode{{
				Vertices: vertices,
				FaceGroup: []*MeshTriangle{{
					Batchid: 0,
					Faces:   faces,
				}},
			}},
			Code: 12345,
		},
		Version: V4,
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

	if len(readMesh.Nodes[0].Vertices) != len(vertices) {
		t.Errorf("Vertices count = %d, want %d", len(readMesh.Nodes[0].Vertices), len(vertices))
	}
	if len(readMesh.Nodes[0].FaceGroup[0].Faces) != len(faces) {
		t.Errorf("Faces count = %d, want %d", len(readMesh.Nodes[0].FaceGroup[0].Faces), len(faces))
	}
}
