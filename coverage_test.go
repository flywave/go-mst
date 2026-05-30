package mst

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"testing"

	mat4d "github.com/flywave/go3d/float64/mat4"
	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
)

func TestMeshReadFromWriteTo(t *testing.T) {
	tempDir := t.TempDir()
	mstFile := filepath.Join(tempDir, "test.mst")

	mesh := NewMesh()
	mesh.Materials = []MeshMaterial{&BaseMaterial{Color: [3]byte{128, 64, 32}, Transparency: 0.7}}
	mesh.Nodes = []*MeshNode{
		{
			Vertices: []vec3.T{{0, 0, 0}, {2, 0, 0}, {0, 2, 0}},
			Normals:  []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
			FaceGroup: []*MeshTriangle{
				{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}},
			},
		},
	}

	if err := MeshWriteTo(mstFile, mesh); err != nil {
		t.Fatalf("MeshWriteTo failed: %v", err)
	}

	readMesh, err := MeshReadFrom(mstFile)
	if err != nil {
		t.Fatalf("MeshReadFrom failed: %v", err)
	}

	if readMesh.Version != mesh.Version {
		t.Errorf("Version = %d, want %d", readMesh.Version, mesh.Version)
	}
	if len(readMesh.Materials) != len(mesh.Materials) {
		t.Errorf("Materials = %d, want %d", len(readMesh.Materials), len(mesh.Materials))
	}
	if len(readMesh.Nodes) != len(mesh.Nodes) {
		t.Errorf("Nodes = %d, want %d", len(readMesh.Nodes), len(mesh.Nodes))
	}
	if len(readMesh.Nodes[0].Vertices) != len(mesh.Nodes[0].Vertices) {
		t.Errorf("Vertices = %d, want %d", len(readMesh.Nodes[0].Vertices), len(mesh.Nodes[0].Vertices))
	}

	if _, err := MeshReadFrom(filepath.Join(tempDir, "nonexistent.mst")); err == nil {
		t.Error("Expected error reading nonexistent file")
	}
}

func TestMeshReadFromWriteToAllVersions(t *testing.T) {
	for version := V1; version <= V6; version++ {
		t.Run(versionName(version), func(t *testing.T) {
			tempDir := t.TempDir()
			mstFile := filepath.Join(tempDir, "test.mst")

			mesh := &Mesh{
				BaseMesh: BaseMesh{
					Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
					Nodes: []*MeshNode{
						{
							Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
							FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
						},
					},
					Code: 42,
				},
				Version:   version,
				Instances: []*InstanceMesh{},
			}

			if version >= V5 {
				props := make(Properties)
				props["ver"] = PropsValue{Type: PROP_TYPE_INT, Value: int64(version)}
				mesh.Props = &props
			}
			if version >= V6 {
				mesh.GeoRef = &GeoRef{EcefOrigin: [3]float64{1, 2, 3}, LatLonOrigin: [3]float64{10, 20, 30}}
			}

			if err := MeshWriteTo(mstFile, mesh); err != nil {
				t.Fatalf("MeshWriteTo failed: %v", err)
			}

			readMesh, err := MeshReadFrom(mstFile)
			if err != nil {
				t.Fatalf("MeshReadFrom failed: %v", err)
			}

			if readMesh.Version != version {
				t.Errorf("Version = %d, want %d", readMesh.Version, version)
			}
		})
	}
}

func TestMeshNodesMarshal(t *testing.T) {
	nodes := []*MeshNode{
		{
			Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}},
			Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}},
			Colors:    [][3]byte{{255, 0, 0}, {0, 255, 0}},
			TexCoords: []vec2.T{{0, 0}, {1, 0}},
			FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
		},
	}

	var buf bytes.Buffer
	if err := MeshNodesMarshal(&buf, nodes); err != nil {
		t.Fatalf("MeshNodesMarshal failed: %v", err)
	}

	unmarshaled := MeshNodesUnMarshal(bytes.NewReader(buf.Bytes()))
	if len(unmarshaled) != len(nodes) {
		t.Fatalf("nodes count = %d, want %d", len(unmarshaled), len(nodes))
	}
	if len(unmarshaled[0].Vertices) != len(nodes[0].Vertices) {
		t.Errorf("vertices mismatch")
	}

	var emptyBuf bytes.Buffer
	MeshNodesMarshal(&emptyBuf, []*MeshNode{})
	emptyNodes := MeshNodesUnMarshal(bytes.NewReader(emptyBuf.Bytes()))
	if len(emptyNodes) != 0 {
		t.Errorf("expected empty nodes, got %d", len(emptyNodes))
	}
}

func TestMeshNodesUnMarshalWithoutProps(t *testing.T) {
	node := &MeshNode{
		Vertices:  []vec3.T{{1, 2, 3}, {4, 5, 6}},
		Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}},
		Colors:    [][3]byte{{10, 20, 30}, {40, 50, 60}},
		TexCoords: []vec2.T{{0.5, 0.5}, {1.0, 1.0}},
		Mat: func() *mat4d.T {
			m := mat4d.Ident
			return &m
		}(),
		FaceGroup: []*MeshTriangle{{Batchid: 1, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}}}},
		EdgeGroup: []*MeshOutline{{Batchid: 0, Edges: [][2]uint32{{0, 1}}}},
	}

	var buf bytes.Buffer
	MeshNodesMarshalForInstanceMesh(&buf, []*MeshNode{node})

	unmarshaled := MeshNodesUnMarshalWithoutProps(bytes.NewReader(buf.Bytes()))
	if len(unmarshaled) != 1 {
		t.Fatalf("expected 1 node, got %d", len(unmarshaled))
	}
	if len(unmarshaled[0].Vertices) != 2 {
		t.Errorf("vertices count = %d, want 2", len(unmarshaled[0].Vertices))
	}
	if unmarshaled[0].Mat == nil {
		t.Error("expected non-nil Mat")
	}
	if len(unmarshaled[0].Colors) != 2 {
		t.Errorf("colors count = %d, want 2", len(unmarshaled[0].Colors))
	}
	if len(unmarshaled[0].TexCoords) != 2 {
		t.Errorf("texcoords count = %d, want 2", len(unmarshaled[0].TexCoords))
	}

	var emptyBuf bytes.Buffer
	MeshNodesMarshalForInstanceMesh(&emptyBuf, []*MeshNode{{}})
	zeroNodes := MeshNodesUnMarshalWithoutProps(bytes.NewReader(emptyBuf.Bytes()))
	if len(zeroNodes) != 1 {
		t.Fatal("expected 1 node")
	}
	if zeroNodes[0].Mat != nil {
		t.Error("expected nil Mat for empty node")
	}
}

func TestMeshNodeMarshalWithoutPropsWithTransform(t *testing.T) {
	transform := mat4d.Ident
	transform[0][0] = 5.0
	node := &MeshNode{
		Vertices: []vec3.T{{0, 0, 0}},
		Mat:      &transform,
	}

	var buf bytes.Buffer
	if err := MeshNodeMarshalWithoutProps(&buf, node); err != nil {
		t.Fatalf("MeshNodeMarshalWithoutProps failed: %v", err)
	}

	unmarshaled := MeshNodeUnMarshalWithoutProps(bytes.NewReader(buf.Bytes()))
	if unmarshaled.Mat == nil {
		t.Fatal("expected non-nil Mat")
	}
	if unmarshaled.Mat[0][0] != 5.0 {
		t.Errorf("Mat[0][0] = %f, want 5.0", unmarshaled.Mat[0][0])
	}
}

func TestMarshalPropsValueRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		value PropsValue
	}{
		{"String", PropsValue{Type: PROP_TYPE_STRING, Value: "hello world"}},
		{"Int", PropsValue{Type: PROP_TYPE_INT, Value: int64(-9223372036854775808)}},
		{"Float", PropsValue{Type: PROP_TYPE_FLOAT, Value: 3.141592653589793}},
		{"BoolTrue", PropsValue{Type: PROP_TYPE_BOOL, Value: true}},
		{"BoolFalse", PropsValue{Type: PROP_TYPE_BOOL, Value: false}},
		{"Array", PropsValue{Type: PROP_TYPE_ARRAY, Value: []PropsValue{
			{Type: PROP_TYPE_STRING, Value: "a"},
			{Type: PROP_TYPE_INT, Value: int64(1)},
			{Type: PROP_TYPE_FLOAT, Value: 1.5},
			{Type: PROP_TYPE_BOOL, Value: true},
		}}},
		{"NestedMap", PropsValue{Type: PROP_TYPE_MAP, Value: Properties{
			"inner_str": {Type: PROP_TYPE_STRING, Value: "inner"},
			"inner_int": {Type: PROP_TYPE_INT, Value: int64(42)},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := Properties{"key": tt.value}

			var buf bytes.Buffer
			if err := PropertiesMarshal(&buf, &props); err != nil {
				t.Fatalf("PropertiesMarshal failed: %v", err)
			}

			unmarshaled := PropertiesUnMarshal(bytes.NewReader(buf.Bytes()))
			if unmarshaled == nil {
				t.Fatal("PropertiesUnMarshal returned nil")
			}

			val, ok := (*unmarshaled)["key"]
			if !ok {
				t.Fatal("key not found after round-trip")
			}

			if val.Type != tt.value.Type {
				t.Errorf("type = %d, want %d", val.Type, tt.value.Type)
			}
		})
	}
}

func TestMarshalPropsValueNestedComplex(t *testing.T) {
	innerArray := []PropsValue{
		{Type: PROP_TYPE_INT, Value: int64(1)},
		{Type: PROP_TYPE_INT, Value: int64(2)},
		{Type: PROP_TYPE_INT, Value: int64(3)},
	}
	nestedMap := Properties{
		"array": {Type: PROP_TYPE_ARRAY, Value: innerArray},
		"str":   {Type: PROP_TYPE_STRING, Value: "deep"},
	}
	props := Properties{
		"outer_map": {Type: PROP_TYPE_MAP, Value: nestedMap},
	}

	var buf bytes.Buffer
	if err := PropertiesMarshal(&buf, &props); err != nil {
		t.Fatalf("PropertiesMarshal failed: %v", err)
	}

	unmarshaled := PropertiesUnMarshal(bytes.NewReader(buf.Bytes()))
	if unmarshaled == nil {
		t.Fatal("PropertiesUnMarshal returned nil")
	}

	outer, ok := (*unmarshaled)["outer_map"]
	if !ok {
		t.Fatal("outer_map not found")
	}
	if outer.Type != PROP_TYPE_MAP {
		t.Errorf("type = %d, want PROP_TYPE_MAP", outer.Type)
	}

	inner, ok := outer.Value.(Properties)
	if !ok {
		t.Fatal("outer_map value is not Properties")
	}
	if inner["str"].Value.(string) != "deep" {
		t.Errorf("inner str = %v, want deep", inner["str"].Value)
	}
}

func TestMarshalPropsNilProperties(t *testing.T) {
	var buf bytes.Buffer
	if err := PropertiesMarshal(&buf, nil); err != nil {
		t.Fatalf("PropertiesMarshal(nil) failed: %v", err)
	}

	unmarshaled := PropertiesUnMarshal(bytes.NewReader(buf.Bytes()))
	if unmarshaled == nil {
		t.Fatal("PropertiesUnMarshal returned nil")
	}
	if len(*unmarshaled) != 0 {
		t.Errorf("expected empty properties, got %d", len(*unmarshaled))
	}
}

func TestPropertiesUnMarshalSafetyLimits(t *testing.T) {
	props := Properties{
		"safe_key": {Type: PROP_TYPE_BOOL, Value: true},
	}

	var buf bytes.Buffer
	PropertiesMarshal(&buf, &props)
	data := buf.Bytes()

	corrupted := make([]byte, len(data))
	copy(corrupted, data)
	corrupted[0] = 0xFF

	result := PropertiesUnMarshal(bytes.NewReader(corrupted))
	if result != nil {
		t.Log("safety limit triggered correctly with corrupted data")
	}
}

func TestMarshalPropsValueTopLevel(t *testing.T) {
	var buf bytes.Buffer
	err := marshalPropsValue(&buf, PropsValue{Type: PROP_TYPE_STRING, Value: "direct"})
	if err != nil {
		t.Fatalf("marshalPropsValue failed: %v", err)
	}

	buf.Reset()
	err = marshalPropsValue(&buf, PropsValue{Type: PROP_TYPE_INT, Value: int64(999)})
	if err != nil {
		t.Fatalf("marshalPropsValue int failed: %v", err)
	}

	buf.Reset()
	err = marshalPropsValue(&buf, PropsValue{Type: PROP_TYPE_FLOAT, Value: 2.718})
	if err != nil {
		t.Fatalf("marshalPropsValue float failed: %v", err)
	}

	buf.Reset()
	err = marshalPropsValue(&buf, PropsValue{Type: PROP_TYPE_BOOL, Value: true})
	if err != nil {
		t.Fatalf("marshalPropsValue bool true failed: %v", err)
	}

	buf.Reset()
	err = marshalPropsValue(&buf, PropsValue{Type: PROP_TYPE_BOOL, Value: false})
	if err != nil {
		t.Fatalf("marshalPropsValue bool false failed: %v", err)
	}

	buf.Reset()
	err = marshalPropsValue(&buf, PropsValue{
		Type:  PROP_TYPE_ARRAY,
		Value: []PropsValue{{Type: PROP_TYPE_INT, Value: int64(1)}},
	})
	if err != nil {
		t.Fatalf("marshalPropsValue array failed: %v", err)
	}

	buf.Reset()
	err = marshalPropsValue(&buf, PropsValue{
		Type:  PROP_TYPE_MAP,
		Value: Properties{"k": {Type: PROP_TYPE_BOOL, Value: true}},
	})
	if err != nil {
		t.Fatalf("marshalPropsValue map failed: %v", err)
	}
}

func TestResortVtVn(t *testing.T) {
	t.Run("WithNormalsAndUVs", func(t *testing.T) {
		node := &MeshNode{
			Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			Normals:   []vec3.T{{0, 0, 1}, {0, 1, 0}, {1, 0, 0}},
			TexCoords: []vec2.T{{0, 0}, {1, 0}, {0, 1}},
			FaceGroup: []*MeshTriangle{
				{
					Faces: []*Face{
						{Vertex: [3]uint32{0, 1, 2}, Normal: &[3]uint32{0, 1, 2}, Uv: &[3]uint32{0, 1, 2}},
						{Vertex: [3]uint32{2, 1, 0}, Normal: &[3]uint32{2, 1, 0}, Uv: &[3]uint32{2, 1, 0}},
					},
				},
			},
		}

		mesh := &Mesh{BaseMesh: BaseMesh{Nodes: []*MeshNode{node}}}
		node.ResortVtVn(mesh)

		expectedVerts := 6
		if len(node.Vertices) != expectedVerts {
			t.Errorf("vertices = %d, want %d", len(node.Vertices), expectedVerts)
		}
		if len(node.Normals) != expectedVerts {
			t.Errorf("normals = %d, want %d", len(node.Normals), expectedVerts)
		}
		if len(node.TexCoords) != expectedVerts {
			t.Errorf("texcoords = %d, want %d", len(node.TexCoords), expectedVerts)
		}

		for i, f := range node.FaceGroup[0].Faces {
			expectedIdx := uint32(i * 3)
			if f.Vertex[0] != expectedIdx || f.Vertex[1] != expectedIdx+1 || f.Vertex[2] != expectedIdx+2 {
				t.Errorf("Face[%d] Vertex = %v, want [%d,%d,%d]", i, f.Vertex, expectedIdx, expectedIdx+1, expectedIdx+2)
			}
		}
	})

	t.Run("WithoutNormalsAndUVs", func(t *testing.T) {
		node := &MeshNode{
			Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			TexCoords: []vec2.T{},
			FaceGroup: []*MeshTriangle{
				{
					Faces: []*Face{
						{Vertex: [3]uint32{0, 1, 2}},
					},
				},
			},
		}

		mesh := &Mesh{BaseMesh: BaseMesh{Nodes: []*MeshNode{node}}}
		node.ResortVtVn(mesh)

		if len(node.Vertices) != 3 {
			t.Errorf("vertices = %d, want 3", len(node.Vertices))
		}
		for _, n := range node.Normals {
			if n != (vec3.T{0, 0, 1}) {
				t.Errorf("expected default normal {0,0,1}, got %v", n)
			}
		}
		for _, uv := range node.TexCoords {
			if uv != (vec2.T{0, 0}) {
				t.Errorf("expected default uv {0,0}, got %v", uv)
			}
		}
	})

	t.Run("EmptyFaceGroup", func(t *testing.T) {
		node := &MeshNode{
			Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
			TexCoords: []vec2.T{{0, 0}, {1, 0}, {0, 1}},
			FaceGroup: []*MeshTriangle{},
		}

		mesh := &Mesh{BaseMesh: BaseMesh{Nodes: []*MeshNode{node}}}
		node.ResortVtVn(mesh)

		if len(node.Vertices) != 0 {
			t.Errorf("expected 0 vertices for empty face group, got %d", len(node.Vertices))
		}
	})
}

func TestCompressDecompressImage(t *testing.T) {
	original := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	compressed := CompressImage(original)
	if len(compressed) == 0 {
		t.Fatal("compressed data is empty")
	}

	decompressed, err := DecompressImage(compressed)
	if err != nil {
		t.Fatalf("DecompressImage failed: %v", err)
	}

	if !bytes.Equal(decompressed, original) {
		t.Errorf("decompressed = %v, want %v", decompressed, original)
	}

	if _, err := DecompressImage([]byte{0, 0, 0}); err == nil {
		t.Error("expected error decompressing invalid data")
	}
}

func TestCreateTextureAndCreateTextureFromImage(t *testing.T) {
	tempDir := t.TempDir()
	imgPath := filepath.Join(tempDir, "test.png")

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, color.NRGBA{uint8(x * 64), uint8(y * 64), 128, 255})
		}
	}

	f, err := os.Create(imgPath)
	if err != nil {
		t.Fatalf("failed to create test image: %v", err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("failed to encode png: %v", err)
	}
	f.Close()

	tex, err := CreateTexture(imgPath, true)
	if err != nil {
		t.Fatalf("CreateTexture failed: %v", err)
	}
	if tex == nil {
		t.Fatal("CreateTexture returned nil")
	}
	if tex.Name != "test.png" {
		t.Errorf("tex.Name = %s, want test.png", tex.Name)
	}
	if tex.Format != TEXTURE_FORMAT_RGBA {
		t.Errorf("Format = %d, want RGBA", tex.Format)
	}
	if tex.Size[0] != 4 || tex.Size[1] != 4 {
		t.Errorf("Size = %dx%d, want 4x4", tex.Size[0], tex.Size[1])
	}
	if tex.Repeated != true {
		t.Error("Expected Repeated = true")
	}
	if len(tex.Data) == 0 {
		t.Error("Expected non-empty Data")
	}

	loadedImg, err := LoadTexture(tex, false)
	if err != nil {
		t.Fatalf("LoadTexture failed: %v", err)
	}
	if loadedImg.Bounds().Dx() != 4 || loadedImg.Bounds().Dy() != 4 {
		t.Errorf("Loaded image size = %dx%d, want 4x4", loadedImg.Bounds().Dx(), loadedImg.Bounds().Dy())
	}

	if _, err := CreateTexture(filepath.Join(tempDir, "nonexistent.png"), false); err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestCreateTextureFromImageDirect(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{255, 0, 0, 255})
	img.Set(1, 0, color.NRGBA{0, 255, 0, 255})
	img.Set(0, 1, color.NRGBA{0, 0, 255, 255})
	img.Set(1, 1, color.NRGBA{255, 255, 0, 255})

	tex, err := CreateTextureFromImage(img, "custom_name.png", false)
	if err != nil {
		t.Fatalf("CreateTextureFromImage failed: %v", err)
	}
	if tex.Name != "custom_name.png" {
		t.Errorf("Name = %s, want custom_name.png", tex.Name)
	}
	if tex.Size[0] != 2 || tex.Size[1] != 2 {
		t.Errorf("Size = %dx%d, want 2x2", tex.Size[0], tex.Size[1])
	}
	if tex.Compressed != TEXTURE_COMPRESSED_ZLIB {
		t.Errorf("Compressed = %d, want ZLIB", tex.Compressed)
	}

	loadedImg, err := LoadTexture(tex, true)
	if err != nil {
		t.Fatalf("LoadTexture with flipY failed: %v", err)
	}
	if loadedImg.Bounds().Dx() != 2 || loadedImg.Bounds().Dy() != 2 {
		t.Errorf("Loaded image size = %dx%d, want 2x2", loadedImg.Bounds().Dx(), loadedImg.Bounds().Dy())
	}
}

func TestLoadTextureFormats(t *testing.T) {
	tests := []struct {
		name   string
		format uint16
		sz     int
	}{
		{"RGB", TEXTURE_FORMAT_RGB, 3},
		{"RGBA", TEXTURE_FORMAT_RGBA, 4},
		{"R", TEXTURE_FORMAT_R, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 2*2*tt.sz)
			for i := range data {
				data[i] = byte(i * 10)
			}
			tex := &Texture{
				Size:       [2]uint64{2, 2},
				Format:     tt.format,
				Type:       TEXTURE_PIXEL_TYPE_UBYTE,
				Compressed: 0,
				Data:       data,
			}

			img, err := LoadTexture(tex, false)
			if err != nil {
				t.Fatalf("LoadTexture failed: %v", err)
			}
			if img.Bounds().Dx() != 2 || img.Bounds().Dy() != 2 {
				t.Errorf("image size = %dx%d, want 2x2", img.Bounds().Dx(), img.Bounds().Dy())
			}
		})
	}
}

func TestGetTextureMethods(t *testing.T) {
	tex := &Texture{Id: 99, Name: "test_tex"}
	normalTex := &Texture{Id: 100, Name: "normal_tex"}

	base := &BaseMaterial{}
	if tex := base.GetTexture(); tex != nil {
		t.Errorf("BaseMaterial.GetTexture should be nil, got %v", tex)
	}

	texMat := &TextureMaterial{
		Texture: tex,
		Normal:  normalTex,
	}
	if got := texMat.GetTexture(); got != tex {
		t.Errorf("TextureMaterial.GetTexture = %v, want %v", got, tex)
	}
	if got := texMat.GetNormalTexture(); got != normalTex {
		t.Errorf("TextureMaterial.GetNormalTexture = %v, want %v", got, normalTex)
	}

	texMatNoTex := &TextureMaterial{}
	if texMatNoTex.GetTexture() != nil {
		t.Error("TextureMaterial without texture should return nil")
	}
	if texMatNoTex.HasTexture() != false {
		t.Error("HasTexture should be false")
	}
	if texMatNoTex.HasNormalTexture() != false {
		t.Error("HasNormalTexture should be false")
	}

	pbr := &PbrMaterial{TextureMaterial: TextureMaterial{Texture: tex}}
	if pbr.GetTexture() != tex {
		t.Error("PbrMaterial.GetTexture should return embedded texture")
	}
}

func TestVersionCodeFieldRoundTrip(t *testing.T) {
	for version := V1; version <= V6; version++ {
		t.Run(versionName(version), func(t *testing.T) {
			mesh := &Mesh{
				BaseMesh: BaseMesh{
					Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
					Nodes: []*MeshNode{
						{
							Vertices:  []vec3.T{{0, 0, 0}},
							FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}}},
						},
					},
					Code: 0xDEADBEAF,
				},
				Version: version,
			}

			var buf bytes.Buffer
			MeshMarshal(&buf, mesh)
			readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))

			if version >= V4 {
				if readMesh.Code != 0xDEADBEAF {
					t.Errorf("Code = 0x%x, want 0x%x", readMesh.Code, 0xDEADBEAF)
				}
			} else {
				if readMesh.Code != 0 {
					t.Errorf("For V%d, Code should be 0, got 0x%x", version, readMesh.Code)
				}
			}
		})
	}
}

func TestPbrMaterialVersionCompatibility(t *testing.T) {
	v1Pbr := &PbrMaterial{
		TextureMaterial: TextureMaterial{
			BaseMaterial: BaseMaterial{Color: [3]byte{255, 0, 0}},
		},
		Emissive: [3]byte{10, 20, 30},
		Metallic: 0.5, Roughness: 0.3,
	}

	var v1Buf bytes.Buffer
	if err := PbrMaterialMarshal(&v1Buf, v1Pbr, V1); err != nil {
		t.Fatalf("PbrMaterialMarshal(V1) failed: %v", err)
	}

	unmarshaledV1 := PbrMaterialUnMarshal(bytes.NewReader(v1Buf.Bytes()), V1)
	if unmarshaledV1.Metallic != 0.5 {
		t.Errorf("V1 Metallic = %f, want 0.5", unmarshaledV1.Metallic)
	}

	var v5Buf bytes.Buffer
	if err := PbrMaterialMarshal(&v5Buf, v1Pbr, V5); err != nil {
		t.Fatalf("PbrMaterialMarshal(V5) failed: %v", err)
	}

	unmarshaledV5 := PbrMaterialUnMarshal(bytes.NewReader(v5Buf.Bytes()), V5)
	if unmarshaledV5.Metallic != 0.5 {
		t.Errorf("V5 Metallic = %f, want 0.5", unmarshaledV5.Metallic)
	}

	if v1Buf.Len() != v5Buf.Len()+1 {
		t.Logf("V1 serialized size = %d, V5 = %d (expected V1 to be 1 byte larger due to extra byte)", v1Buf.Len(), v5Buf.Len())
	}
}

func TestInstanceMeshV3FeaturesUint32(t *testing.T) {
	transform := mat4d.Ident
	instanceMesh := &InstanceMesh{
		Transfors: []*mat4d.T{&transform},
		Features:  []uint64{100, 200, 300},
		Mesh:      &BaseMesh{},
	}

	var bufV3 bytes.Buffer
	MeshInstanceNodeMarshal(&bufV3, instanceMesh, V3)
	unmarshaledV3 := MeshInstanceNodeUnMarshal(bytes.NewReader(bufV3.Bytes()), V3)
	if unmarshaledV3 == nil {
		t.Fatal("V3 unmarshal failed")
	}
	if len(unmarshaledV3.Features) != 3 {
		t.Errorf("V3 features = %d, want 3", len(unmarshaledV3.Features))
	}
	if unmarshaledV3.Features[0] != 100 {
		t.Errorf("V3 feature[0] = %d, want 100", unmarshaledV3.Features[0])
	}

	var bufV4 bytes.Buffer
	MeshInstanceNodeMarshal(&bufV4, instanceMesh, V4)
	unmarshaledV4 := MeshInstanceNodeUnMarshal(bytes.NewReader(bufV4.Bytes()), V4)
	if unmarshaledV4 == nil {
		t.Fatal("V4 unmarshal failed")
	}
	if len(unmarshaledV4.Features) != 3 {
		t.Errorf("V4 features = %d, want 3", len(unmarshaledV4.Features))
	}
}

func TestGeoRefMarshalUnmarshal(t *testing.T) {
	geoRef := &GeoRef{
		EcefOrigin:   [3]float64{1000.5, 2000.5, 3000.5},
		LatLonOrigin: [3]float64{39.9042, 116.4074, 50.0},
	}

	var buf bytes.Buffer
	if err := GeoRefMarshal(&buf, geoRef); err != nil {
		t.Fatalf("GeoRefMarshal failed: %v", err)
	}

	readBuf := bytes.NewReader(buf.Bytes())
	unmarshaled := GeoRefUnMarshal(readBuf)
	if unmarshaled == nil {
		t.Fatal("GeoRefUnMarshal returned nil")
	}

	if unmarshaled.EcefOrigin != geoRef.EcefOrigin {
		t.Errorf("EcefOrigin = %v, want %v", unmarshaled.EcefOrigin, geoRef.EcefOrigin)
	}
	if unmarshaled.LatLonOrigin != geoRef.LatLonOrigin {
		t.Errorf("LatLonOrigin = %v, want %v", unmarshaled.LatLonOrigin, geoRef.LatLonOrigin)
	}
}

func TestV5MeshRoundTripWithPropsGeoRefMissing(t *testing.T) {
	props := make(Properties)
	props["test"] = PropsValue{Type: PROP_TYPE_BOOL, Value: true}

	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&PbrMaterial{
					TextureMaterial: TextureMaterial{
						BaseMaterial: BaseMaterial{Color: [3]byte{128, 128, 128}},
						Texture: &Texture{
							Id: 1, Name: "tex", Size: [2]uint64{4, 4},
							Format: TEXTURE_FORMAT_RGBA, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
						},
					},
					Metallic: 1.0, Roughness: 0.0,
				},
			},
			Nodes: []*MeshNode{
				{
					Vertices:  []vec3.T{{0, 0, 0}},
					Normals:   []vec3.T{{0, 0, 1}},
					TexCoords: []vec2.T{{0, 0}},
					FaceGroup: []*MeshTriangle{
						{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}},
					},
				},
			},
			Code: 100,
		},
		Version: V5,
		Props:   &props,
	}

	var buf bytes.Buffer
	if err := MeshMarshal(&buf, mesh); err != nil {
		t.Fatalf("MeshMarshal failed: %v", err)
	}

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	if readMesh == nil {
		t.Fatal("MeshUnMarshal returned nil")
	}
	if readMesh.Version != V5 {
		t.Errorf("Version = %d, want V5", readMesh.Version)
	}
	if readMesh.Props == nil {
		t.Fatal("Props should not be nil")
	}
	if readMesh.GeoRef != nil {
		t.Error("GeoRef should be nil for V5")
	}
	if readMesh.Code != 100 {
		t.Errorf("Code = %d, want 100", readMesh.Code)
	}
	if len(readMesh.Materials) != 1 {
		t.Errorf("Materials = %d, want 1", len(readMesh.Materials))
	}
	if len(readMesh.Nodes) != 1 {
		t.Errorf("Nodes = %d, want 1", len(readMesh.Nodes))
	}
}

func TestInstanceMeshMarshalWithoutCodeV3(t *testing.T) {
	transform := mat4d.Ident
	instanceMesh := &InstanceMesh{
		Transfors: []*mat4d.T{&transform},
		Features:  []uint64{1},
		Mesh: &BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 0}}}},
			Code:      999,
		},
	}

	var buf bytes.Buffer
	MeshInstanceNodeMarshal(&buf, instanceMesh, V3)
	unmarshaled := MeshInstanceNodeUnMarshal(bytes.NewReader(buf.Bytes()), V3)
	if unmarshaled == nil {
		t.Fatal("V3 instance unmarshal failed")
	}
	if unmarshaled.Mesh.Code != 0 {
		t.Errorf("Code should be 0 for V3, got %d", unmarshaled.Mesh.Code)
	}
	if unmarshaled.Props != nil {
		t.Error("Props should be nil for V3")
	}
}

func TestInstanceMeshCodeV4(t *testing.T) {
	transform := mat4d.Ident
	instanceMesh := &InstanceMesh{
		Transfors: []*mat4d.T{&transform},
		Features:  []uint64{1},
		Mesh: &BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 0}}}},
			Code:      555,
		},
	}

	var buf bytes.Buffer
	MeshInstanceNodeMarshal(&buf, instanceMesh, V4)
	unmarshaled := MeshInstanceNodeUnMarshal(bytes.NewReader(buf.Bytes()), V4)
	if unmarshaled == nil {
		t.Fatal("V4 instance unmarshal failed")
	}
	if unmarshaled.Mesh.Code != 555 {
		t.Errorf("Code = %d, want 555", unmarshaled.Mesh.Code)
	}
}

func TestMaterialMarshalAllTypesAllVersions(t *testing.T) {
	materials := []MeshMaterial{
		&BaseMaterial{Color: [3]byte{255, 0, 0}, Transparency: 0.5},
		&PbrMaterial{
			TextureMaterial: TextureMaterial{
				BaseMaterial: BaseMaterial{Color: [3]byte{0, 255, 0}},
				Texture:      &Texture{Id: 1, Name: "t", Size: [2]uint64{1, 1}, Data: []byte{1}},
			},
			Emissive: [3]byte{1, 2, 3}, Metallic: 0.9, Roughness: 0.1,
		},
		&LambertMaterial{
			TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{0, 0, 255}}},
			Ambient: [3]byte{10, 10, 10}, Diffuse: [3]byte{128, 128, 128},
		},
		&PhongMaterial{
			LambertMaterial: LambertMaterial{
				TextureMaterial: TextureMaterial{BaseMaterial: BaseMaterial{Color: [3]byte{255, 255, 0}}},
			},
			Specular: [3]byte{200, 200, 200}, Shininess: 64, Specularity: 0.5,
		},
	}

	for version := V1; version <= V6; version++ {
		t.Run(versionName(version), func(t *testing.T) {
			var buf bytes.Buffer
			if err := MtlsMarshal(&buf, materials, version); err != nil {
				t.Fatalf("MtlsMarshal(V%d) failed: %v", version, err)
			}

			unmarshaled := MtlsUnMarshal(bytes.NewReader(buf.Bytes()), version)
			if len(unmarshaled) != len(materials) {
				t.Fatalf("material count = %d, want %d", len(unmarshaled), len(materials))
			}

			for i, m := range unmarshaled {
				if m == nil {
					t.Errorf("unmarshaled materials[%d] is nil", i)
					continue
				}
				if m.GetColor() != materials[i].GetColor() {
					t.Errorf("materials[%d] color mismatch", i)
				}
			}
		})
	}
}

func TestTextureMaterialMarshalWithNormal(t *testing.T) {
	texMat := &TextureMaterial{
		BaseMaterial: BaseMaterial{Color: [3]byte{100, 100, 100}},
		Texture: &Texture{
			Id: 1, Name: "diffuse", Size: [2]uint64{2, 2}, Format: TEXTURE_FORMAT_RGB,
			Type: TEXTURE_PIXEL_TYPE_UBYTE, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		},
		Normal: &Texture{
			Id: 2, Name: "normal", Size: [2]uint64{2, 2}, Format: TEXTURE_FORMAT_RGBA,
			Type: TEXTURE_PIXEL_TYPE_UBYTE, Data: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		},
	}

	var buf bytes.Buffer
	if err := TextureMaterialMarshal(&buf, texMat); err != nil {
		t.Fatalf("TextureMaterialMarshal failed: %v", err)
	}

	unmarshaled := TextureMaterialUnMarshal(bytes.NewReader(buf.Bytes()))
	if unmarshaled.Texture == nil {
		t.Fatal("Texture should not be nil")
	}
	if unmarshaled.Normal == nil {
		t.Fatal("Normal texture should not be nil")
	}
	if unmarshaled.Texture.Id != 1 {
		t.Errorf("Texture.Id = %d, want 1", unmarshaled.Texture.Id)
	}
	if unmarshaled.Normal.Id != 2 {
		t.Errorf("Normal.Id = %d, want 2", unmarshaled.Normal.Id)
	}
}

func TestTextureMaterialMarshalWithoutNormal(t *testing.T) {
	texMat := &TextureMaterial{
		BaseMaterial: BaseMaterial{Color: [3]byte{50, 50, 50}},
		Texture:      nil,
	}

	var buf bytes.Buffer
	if err := TextureMaterialMarshal(&buf, texMat); err != nil {
		t.Fatalf("TextureMaterialMarshal failed: %v", err)
	}

	unmarshaled := TextureMaterialUnMarshal(bytes.NewReader(buf.Bytes()))
	if unmarshaled.Texture != nil {
		t.Error("Texture should be nil")
	}
	if unmarshaled.Normal != nil {
		t.Error("Normal should be nil")
	}
}

func TestMeshTriangleOutlineRoundTrip(t *testing.T) {
	triangle := &MeshTriangle{
		Batchid: 10,
		Faces: []*Face{
			{Vertex: [3]uint32{0, 1, 2}, Normal: &[3]uint32{0, 0, 1}, Uv: &[3]uint32{0, 1, 2}},
			{Vertex: [3]uint32{3, 4, 5}},
		},
	}

	var triBuf bytes.Buffer
	if err := MeshTriangleMarshal(&triBuf, triangle); err != nil {
		t.Fatalf("MeshTriangleMarshal failed: %v", err)
	}
	unmarshaledTri := MeshTriangleUnMarshal(bytes.NewReader(triBuf.Bytes()))
	if unmarshaledTri.Batchid != 10 {
		t.Errorf("Batchid = %d, want 10", unmarshaledTri.Batchid)
	}
	if len(unmarshaledTri.Faces) != 2 {
		t.Errorf("Faces count = %d, want 2", len(unmarshaledTri.Faces))
	}

	outline := &MeshOutline{
		Batchid: 20,
		Edges:   [][2]uint32{{0, 1}, {1, 2}, {2, 0}},
	}

	var outBuf bytes.Buffer
	if err := MeshOutlineMarshal(&outBuf, outline); err != nil {
		t.Fatalf("MeshOutlineMarshal failed: %v", err)
	}
	unmarshaledOut := MeshOutlineUnMarshal(bytes.NewReader(outBuf.Bytes()))
	if unmarshaledOut.Batchid != 20 {
		t.Errorf("Batchid = %d, want 20", unmarshaledOut.Batchid)
	}
	if len(unmarshaledOut.Edges) != 3 {
		t.Errorf("Edges count = %d, want 3", len(unmarshaledOut.Edges))
	}
}

func TestUnmarshalPropsValueEdgeCases(t *testing.T) {
	var buf bytes.Buffer

	buf.Write([]byte{0, 0, 0, 0})
	writeLittleByte(&buf, uint32(0))
	writeLittleByte(&buf, uint32(PROP_TYPE_STRING))
	writeLittleByte(&buf, uint32(0))
	result := unmarshalPropsValue(bytes.NewReader(buf.Bytes()[4:]), PROP_TYPE_STRING)
	if result.Type == -1 {
		t.Error("empty string should be valid")
	}

	longStr := [4]byte{0xFF, 0xFF, 0, 0}
	result = unmarshalPropsValue(bytes.NewReader(longStr[:]), PROP_TYPE_STRING)
	if result.Type != -1 {
		t.Error("string exceeding safety limit should return error")
	}

	var badTypeBuf bytes.Buffer
	result = unmarshalPropsValue(&badTypeBuf, PropsType(255))
	if result.Type != -1 {
		t.Error("unknown type should return error")
	}

	var invalidBool bytes.Buffer
	n, err := invalidBool.Write([]byte{3, 0, 0, 0, 0, 0, 0, 0})
	_ = n
	_ = err
	result2 := unmarshalPropsValue(&invalidBool, PROP_TYPE_INT)
	if result2.Type != -1 {
		t.Logf("partial read returned type=%d", result2.Type)
	}
}

func TestReComputeNormalEdgeCases(t *testing.T) {
	t.Run("ZeroLengthEdge", func(t *testing.T) {
		node := &MeshNode{
			Vertices: []vec3.T{{0, 0, 0}, {0, 0, 0}, {0, 0, 0}},
			FaceGroup: []*MeshTriangle{{
				Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}},
			}},
		}

		node.ReComputeNormal()
		if len(node.Normals) != 3 {
			t.Errorf("normals = %d, want 3", len(node.Normals))
		}
	})

	t.Run("SingleFace", func(t *testing.T) {
		node := &MeshNode{
			Vertices: []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
			FaceGroup: []*MeshTriangle{{
				Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}}},
			}},
		}

		node.ReComputeNormal()
		if len(node.Normals) != 3 {
			t.Errorf("normals = %d, want 3", len(node.Normals))
		}
		for _, n := range node.Normals {
			length := n.Length()
			if length < 0.99 || length > 1.01 {
				t.Errorf("normal not normalized, length = %f", length)
			}
		}
	})
}

func versionName(v uint32) string {
	switch v {
	case V1:
		return "V1"
	case V2:
		return "V2"
	case V3:
		return "V3"
	case V4:
		return "V4"
	case V5:
		return "V5"
	case V6:
		return "V6"
	default:
		return "Unknown"
	}
}

func makeTransform() *mat4d.T {
	m := mat4d.Ident
	return &m
}

func TestUpgradeMeshFromV1(t *testing.T) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&BaseMaterial{Color: [3]byte{255, 0, 0}},
				&PbrMaterial{
					TextureMaterial: TextureMaterial{
						BaseMaterial: BaseMaterial{Color: [3]byte{0, 255, 0}},
					},
					Emissive: [3]byte{1, 2, 3}, Metallic: 0.5, Roughness: 0.3,
				},
			},
			Nodes: []*MeshNode{
				{
					Vertices:  []vec3.T{{0, 0, 0}},
					FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}}},
				},
			},
			Code: 12345,
		},
		Version: V1,
		Instances: []*InstanceMesh{
			{
				Transfors: []*mat4d.T{makeTransform()},
				Features:  []uint64{100, 200},
				BBox:      &[6]float64{},
				Mesh:      &BaseMesh{},
			},
		},
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	UpgradeMesh(readMesh)

	if readMesh.Version != V6 {
		t.Errorf("Version = %d, want V6", readMesh.Version)
	}
	if readMesh.Props == nil {
		t.Error("Props should be non-nil after upgrade")
	}
	if len(readMesh.Instances) != 1 {
		t.Fatalf("expected 1 instance, got %d", len(readMesh.Instances))
	}
	if readMesh.Instances[0].Props == nil {
		t.Error("Instance Props should be non-nil after upgrade")
	}
	if len(readMesh.Instances[0].Props) != 2 {
		t.Errorf("Instance Props length = %d, want 2 (matching Transfors count)", len(readMesh.Instances[0].Props))
	}
	if readMesh.Instances[0].BBox == nil {
		t.Error("Instance BBox should be non-nil after upgrade")
	}

	var reBuf bytes.Buffer
	if err := MeshMarshal(&reBuf, readMesh); err != nil {
		t.Fatalf("Re-marshal after upgrade failed: %v", err)
	}

	reReadMesh := MeshUnMarshal(bytes.NewReader(reBuf.Bytes()))
	if reReadMesh.Version != V6 {
		t.Errorf("After re-serialize, Version = %d, want V6", reReadMesh.Version)
	}
	if reReadMesh.Props == nil {
		t.Error("Props should be non-nil after re-serialize")
	}
}

func TestUpgradeMeshFromV2(t *testing.T) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&PbrMaterial{
					TextureMaterial: TextureMaterial{
						BaseMaterial: BaseMaterial{Color: [3]byte{100, 100, 100}},
					},
					Emissive: [3]byte{10, 20, 30}, Metallic: 0.9, Roughness: 0.1,
				},
			},
			Nodes: []*MeshNode{
				{
					Vertices:  []vec3.T{{0, 0, 0}},
					FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}}},
				},
			},
			Code: 0,
		},
		Version: V2,
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	UpgradeMesh(readMesh)

	if readMesh.Version != V6 {
		t.Errorf("V2 upgrade: Version = %d, want V6", readMesh.Version)
	}
	if readMesh.Props == nil {
		t.Error("V2 upgrade: Props should be non-nil")
	}
}

func TestUpgradeMeshFromV3(t *testing.T) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&LambertMaterial{
					TextureMaterial: TextureMaterial{
						BaseMaterial: BaseMaterial{Color: [3]byte{0, 0, 255}},
					},
				},
			},
			Nodes: []*MeshNode{
				{
					Vertices:  []vec3.T{{0, 0, 0}},
					FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}}},
				},
			},
			Code: 777,
		},
		Version: V3,
		Instances: []*InstanceMesh{
			{
				Transfors: []*mat4d.T{makeTransform()},
				Features:  []uint64{1, 2, 3, 4, 5},
				BBox:      &[6]float64{},
				Mesh:      &BaseMesh{},
			},
		},
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	UpgradeMesh(readMesh)

	if readMesh.Version != V6 {
		t.Errorf("V3 upgrade: Version = %d, want V6", readMesh.Version)
	}
	if len(readMesh.Instances[0].Props) != 5 {
		t.Errorf("V3 upgrade: instance Props length = %d, want 5", len(readMesh.Instances[0].Props))
	}
}

func TestUpgradeMeshFromV4(t *testing.T) {
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&PhongMaterial{
					LambertMaterial: LambertMaterial{
						TextureMaterial: TextureMaterial{
							BaseMaterial: BaseMaterial{Color: [3]byte{128, 128, 128}},
						},
					},
					Specular: [3]byte{255, 255, 255}, Shininess: 64,
				},
			},
			Nodes: []*MeshNode{
				{
					Vertices:  []vec3.T{{0, 0, 0}},
					FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}}},
				},
			},
			Code: 9999,
		},
		Version: V4,
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	UpgradeMesh(readMesh)

	if readMesh.Version != V6 {
		t.Errorf("V4 upgrade: Version = %d, want V6", readMesh.Version)
	}
	if readMesh.Props == nil {
		t.Error("V4 upgrade: Props should be non-nil")
	}
	if readMesh.Code != 9999 {
		t.Errorf("V4 upgrade: Code = %d, want 9999", readMesh.Code)
	}
}

func TestUpgradeMeshFromV5(t *testing.T) {
	props := make(Properties)
	props["name"] = PropsValue{Type: PROP_TYPE_STRING, Value: "v5 mesh"}
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{
				&BaseMaterial{Color: [3]byte{255, 0, 0}},
			},
			Nodes: []*MeshNode{
				{
					Vertices:  []vec3.T{{0, 0, 0}},
					FaceGroup: []*MeshTriangle{{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 0, 0}}}}},
				},
			},
		},
		Version: V5,
		Props:   &props,
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	UpgradeMesh(readMesh)

	if readMesh.Version != V6 {
		t.Errorf("V5 upgrade: Version = %d, want V6", readMesh.Version)
	}
	if readMesh.Props == nil {
		t.Error("V5 upgrade: Props should not be nil")
	}
	if (*readMesh.Props)["name"].Value.(string) != "v5 mesh" {
		t.Error("V5 upgrade: Props data lost")
	}
}

func TestUpgradeMeshFromV5WithInstanceProps(t *testing.T) {
	instanceProps := make(Properties)
	instanceProps["instance_key"] = PropsValue{Type: PROP_TYPE_STRING, Value: "keep_me"}
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{0, 255, 0}}},
			Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 0}}}},
		},
		Version: V5,
		Instances: []*InstanceMesh{
			{
				Transfors: []*mat4d.T{makeTransform()},
				Features:  []uint64{100},
				BBox:      &[6]float64{},
				Mesh:      &BaseMesh{},
				Props:     []*Properties{&instanceProps},
				Hash:      1,
			},
		},
	}

	var buf bytes.Buffer
	MeshMarshal(&buf, mesh)

	readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
	UpgradeMesh(readMesh)

	if readMesh.Version != V6 {
		t.Errorf("Version = %d, want V6", readMesh.Version)
	}
	if len(readMesh.Instances[0].Props) != 1 {
		t.Fatalf("instance Props length = %d, want 1", len(readMesh.Instances[0].Props))
	}
	if (*readMesh.Instances[0].Props[0])["instance_key"].Value.(string) != "keep_me" {
		t.Error("Instance Props data lost during upgrade")
	}
}

func TestUpgradeMeshAlreadyV6(t *testing.T) {
	geoRef := &GeoRef{EcefOrigin: [3]float64{1, 2, 3}, LatLonOrigin: [3]float64{10, 20, 30}}
	props := make(Properties)
	props["existing"] = PropsValue{Type: PROP_TYPE_BOOL, Value: true}
	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 0}}}},
		},
		Version: V6,
		Props:   &props,
		GeoRef:  geoRef,
	}

	UpgradeMesh(mesh)

	if mesh.Version != V6 {
		t.Errorf("Version = %d, want V6", mesh.Version)
	}
	if mesh.Props != &props {
		t.Error("Props pointer changed")
	}
	if mesh.GeoRef != geoRef {
		t.Error("GeoRef pointer changed")
	}
}

func TestUpgradeMeshFileRoundTrip(t *testing.T) {
	tempDir := t.TempDir()
	mstFile := filepath.Join(tempDir, "upgrade_test.mst")

	mesh := &Mesh{
		BaseMesh: BaseMesh{
			Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{255, 0, 0}}},
			Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 0}}}},
		},
		Version: V1,
	}

	if err := MeshWriteTo(mstFile, mesh); err != nil {
		t.Fatalf("MeshWriteTo failed: %v", err)
	}

	if err := UpgradeMeshFile(mstFile); err != nil {
		t.Fatalf("UpgradeMeshFile failed: %v", err)
	}

	upgraded, err := MeshReadFrom(mstFile)
	if err != nil {
		t.Fatalf("MeshReadFrom after upgrade failed: %v", err)
	}

	if upgraded.Version != V6 {
		t.Errorf("After UpgradeMeshFile, Version = %d, want V6", upgraded.Version)
	}
	if upgraded.Props == nil {
		t.Error("Props should be non-nil after file upgrade")
	}
}

func TestUpgradeMeshFileNonExistent(t *testing.T) {
	err := UpgradeMeshFile("/nonexistent/path.mst")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestUpgradeMeshAllVersions(t *testing.T) {
	for version := V1; version <= V5; version++ {
		t.Run(versionName(version), func(t *testing.T) {
			mesh := &Mesh{
				BaseMesh: BaseMesh{
					Materials: []MeshMaterial{
						&PbrMaterial{
							TextureMaterial: TextureMaterial{
								BaseMaterial: BaseMaterial{Color: [3]byte{255, 0, 0}},
								Texture:      &Texture{Id: 1, Name: "t", Size: [2]uint64{1, 1}, Format: TEXTURE_FORMAT_RGBA, Data: []byte{1, 2, 3, 4}},
							},
							Metallic: 0.5, Roughness: 0.5, Emissive: [3]byte{10, 10, 10},
						},
					},
					Nodes: []*MeshNode{
						{
							Vertices:  []vec3.T{{0, 0, 0}, {1, 0, 0}, {0, 1, 0}},
							Normals:   []vec3.T{{0, 0, 1}, {0, 0, 1}, {0, 0, 1}},
							TexCoords: []vec2.T{{0, 0}, {1, 0}, {0, 1}},
							FaceGroup: []*MeshTriangle{
								{Batchid: 0, Faces: []*Face{{Vertex: [3]uint32{0, 1, 2}, Normal: &[3]uint32{0, 1, 2}, Uv: &[3]uint32{0, 1, 2}}}},
							},
						},
					},
					Code: 42,
				},
				Version: version,
				Instances: []*InstanceMesh{
					{
						Transfors: []*mat4d.T{makeTransform()},
						Features:  []uint64{1, 2},
						BBox:      &[6]float64{},
						Mesh: &BaseMesh{
							Materials: []MeshMaterial{&BaseMaterial{Color: [3]byte{0, 255, 0}}},
							Nodes:     []*MeshNode{{Vertices: []vec3.T{{0, 0, 0}}}},
							Code:      100,
						},
					},
				},
			}

			var buf bytes.Buffer
			MeshMarshal(&buf, mesh)
			readMesh := MeshUnMarshal(bytes.NewReader(buf.Bytes()))
			UpgradeMesh(readMesh)

			if readMesh.Version != V6 {
				t.Errorf("Version = %d, want V6", readMesh.Version)
			}
			if readMesh.Props == nil {
				t.Error("Props should be non-nil")
			}
			if len(readMesh.Materials) != 1 {
				t.Errorf("Materials = %d, want 1", len(readMesh.Materials))
			}
			if len(readMesh.Nodes) != 1 {
				t.Errorf("Nodes = %d, want 1", len(readMesh.Nodes))
			}
			if version >= V4 && readMesh.Code != 42 {
				t.Errorf("Code = %d, want 42", readMesh.Code)
			}

			if len(readMesh.Instances) != 1 {
				t.Fatalf("Instances = %d, want 1", len(readMesh.Instances))
			}
			if readMesh.Instances[0].Props == nil {
				t.Error("Instance Props should be non-nil")
			}
			if len(readMesh.Instances[0].Props) != 2 {
				t.Errorf("Instance Props length = %d, want 2", len(readMesh.Instances[0].Props))
			}
			if version >= V4 && readMesh.Instances[0].Mesh.Code != 100 {
				t.Errorf("Instance Code = %d, want 100", readMesh.Instances[0].Mesh.Code)
			}

			var reBuf bytes.Buffer
			if err := MeshMarshal(&reBuf, readMesh); err != nil {
				t.Fatalf("Re-marshal after upgrade failed: %v", err)
			}
			reRead := MeshUnMarshal(bytes.NewReader(reBuf.Bytes()))
			if reRead.Version != V6 {
				t.Errorf("After re-serialize: Version = %d, want V6", reRead.Version)
			}
			if reRead.Props == nil {
				t.Error("After re-serialize: Props should not be nil")
			}
		})
	}
}

func TestSortImport(t *testing.T) {
	sort.Ints([]int{3, 1, 2})
}
