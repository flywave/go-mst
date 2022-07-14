package mst

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"io"

	mat4d "github.com/flywave/go3d/float64/mat4"

	"github.com/qmuntal/gltf"
	"github.com/qmuntal/gltf/ext/specular"
)

const GLTF_VERSION = "2.0"

func MstToGltf(msts []*Mesh) (*gltf.Document, error) {
	doc := CreateDoc()
	for _, mst := range msts {
		e := BuildGltf(doc, mst, false)
		if e != nil {
			return nil, e
		}
	}
	return doc, nil
}

func MstToGltfWithOutline(msts []*Mesh) (*gltf.Document, error) {
	doc := CreateDoc()
	for _, mst := range msts {
		e := BuildGltf(doc, mst, true)
		if e != nil {
			return nil, e
		}
	}
	return doc, nil
}
func CreateDoc() *gltf.Document {
	doc := &gltf.Document{}
	doc.Asset.Version = GLTF_VERSION
	srcIndex := uint32(0)
	doc.Scene = &srcIndex
	doc.Scenes = append(doc.Scenes, &gltf.Scene{})
	doc.Buffers = append(doc.Buffers, &gltf.Buffer{})
	return doc
}

type calcSizeWriter struct {
	writer io.Writer
	Size   int
}

func (w *calcSizeWriter) Write(p []byte) (n int, err error) {
	si := len(p)
	w.writer.Write(p)
	w.Size += int(si)
	return si, nil
}

func (w *calcSizeWriter) Bytes() []byte {
	return w.writer.(*bytes.Buffer).Bytes()
}

func (w *calcSizeWriter) GetSize() int {
	return len(w.Bytes())
}

func newSizeWriter() calcSizeWriter {
	wt := bytes.NewBuffer([]byte{})
	return calcSizeWriter{Size: int(0), writer: wt}
}

func calcPadding(offset, paddingUnit int) int {
	padding := offset % paddingUnit
	if padding != 0 {
		padding = paddingUnit - padding
	}
	return padding
}

func GetGltfBinary(doc *gltf.Document, paddingUnit int) ([]byte, error) {
	w := newSizeWriter()
	enc := gltf.NewEncoder(w.writer)
	enc.AsBinary = true
	if err := enc.Encode(doc); err != nil {
		return nil, err
	}
	padding := calcPadding(w.Size, paddingUnit)
	if padding == 0 {
		return w.Bytes(), nil
	}
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = 0x20
	}
	w.Write(pad)
	return w.Bytes(), nil
}

func BuildGltf(doc *gltf.Document, mh *Mesh, exportOutline bool) error {
	err := buildGltf(doc, &mh.BaseMesh, true, exportOutline)
	if err != nil {
		return err
	}
	for _, inst := range mh.InstanceNode {
		meshId := uint32(len(doc.Meshes))
		buildGltf(doc, inst.Mesh, false, false)
		for _, mt := range inst.Transfors {
			position, quat, scale := mat4d.Decompose(mt)
			nd := gltf.Node{
				Mesh:        &meshId,
				Translation: [3]float32{float32(position[0]), float32(position[1]), float32(position[2])},
				Rotation:    [4]float32{float32(quat[0]), float32(quat[1]), float32(quat[2]), float32(quat[3])},
				Scale:       [3]float32{float32(scale[0]), float32(scale[1]), float32(scale[2])},
			}
			doc.Nodes = append(doc.Nodes, &nd)
			doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, uint32(len(doc.Nodes)-1))
		}
	}

	return nil
}

type buildContext struct {
	mtlSize uint32
	bvIndex uint32
	bvPos   uint32
	bvTex   uint32
	bvNorm  uint32
}

func buildMeshBuffer(ctx *buildContext, buffer *gltf.Buffer, bufferViews []*gltf.BufferView, nd *MeshNode) []*gltf.BufferView {
	var bt []byte
	buf := bytes.NewBuffer(bt)
	ctx.bvIndex = uint32(len(bufferViews))
	indecs := &gltf.BufferView{}
	startLen := buffer.ByteLength
	indecs.ByteOffset = startLen
	for _, g := range nd.FaceGroup {
		for _, f := range g.Faces {
			binary.Write(buf, binary.LittleEndian, f.Vertex)
		}
	}
	indecs.ByteLength = uint32(buf.Len())
	indecs.Buffer = 0
	bufferViews = append(bufferViews, indecs)

	postions := &gltf.BufferView{}
	postions.ByteOffset = uint32(buf.Len()) + startLen
	binary.Write(buf, binary.LittleEndian, nd.Vertices)
	postions.ByteLength = uint32(buf.Len()) - postions.ByteOffset + startLen
	postions.Buffer = 0
	ctx.bvPos = uint32(len(bufferViews))
	bufferViews = append(bufferViews, postions)

	texcood := &gltf.BufferView{}
	ctx.bvTex = uint32(len(bufferViews))
	if len(nd.TexCoords) > 0 {
		texcood.ByteOffset = uint32(buf.Len()) + startLen
		binary.Write(buf, binary.LittleEndian, nd.TexCoords)
		texcood.ByteLength = uint32(buf.Len()) - texcood.ByteOffset + startLen
		texcood.Buffer = 0
		bufferViews = append(bufferViews, texcood)
	}

	normalView := &gltf.BufferView{}
	ctx.bvNorm = uint32(len(bufferViews))
	if len(nd.Normals) > 0 {
		normalView.ByteOffset = uint32(buf.Len()) + startLen
		binary.Write(buf, binary.LittleEndian, nd.Normals)
		normalView.ByteLength = uint32(buf.Len()) - normalView.ByteOffset + startLen
		normalView.Buffer = 0
		bufferViews = append(bufferViews, normalView)
	}
	buffer.ByteLength += uint32(buf.Len())
	buffer.Data = append(buffer.Data, buf.Bytes()...)

	return bufferViews
}

func buildOutlineBuffer(ctx *buildContext, buffer *gltf.Buffer, bufferViews []*gltf.BufferView, nd *MeshNode) []*gltf.BufferView {
	var bt []byte
	buf := bytes.NewBuffer(bt)
	ctx.bvIndex = uint32(len(bufferViews))
	indecs := &gltf.BufferView{}
	startLen := buffer.ByteLength
	indecs.ByteOffset = startLen
	for _, g := range nd.EdgeGroup {
		for _, f := range g.Edges {
			binary.Write(buf, binary.LittleEndian, f)
		}
	}
	indecs.ByteLength = uint32(buf.Len())
	indecs.Buffer = 0
	bufferViews = append(bufferViews, indecs)

	postions := &gltf.BufferView{}
	postions.ByteOffset = uint32(buf.Len()) + startLen
	binary.Write(buf, binary.LittleEndian, nd.Vertices)
	postions.ByteLength = uint32(buf.Len()) - postions.ByteOffset + startLen
	postions.Buffer = 0
	ctx.bvPos = uint32(len(bufferViews))
	bufferViews = append(bufferViews, postions)

	buffer.ByteLength += uint32(buf.Len())
	buffer.Data = append(buffer.Data, buf.Bytes()...)

	return bufferViews
}

func buildOutline(ctx *buildContext, accessors []*gltf.Accessor, nd *MeshNode) (*gltf.Mesh, []*gltf.Accessor) {
	mesh := &gltf.Mesh{}
	aftIndices := uint32(len(nd.EdgeGroup))
	idx := uint32(len(accessors))
	indexPos := aftIndices + idx
	var start uint32 = 0
	for i := range nd.EdgeGroup {
		patch := nd.EdgeGroup[i]
		batchId := patch.Batchid
		if batchId < 0 {
			batchId = 0
		}
		mtl_id := uint32(batchId) + ctx.mtlSize

		ps := &gltf.Primitive{}
		ps.Material = &mtl_id
		if ps.Attributes == nil {
			ps.Attributes = make(gltf.Attribute)
		}
		index := uint32(i) + idx
		ps.Indices = &index

		ps.Attributes["POSITION"] = indexPos

		ps.Mode = gltf.PrimitiveLineStrip
		mesh.Primitives = append(mesh.Primitives, ps)

		indexacc := &gltf.Accessor{}
		indexacc.ComponentType = gltf.ComponentUint

		indexacc.ByteOffset = start * 8
		indexacc.Count = uint32(len(patch.Edges)) * 2
		start += uint32(len(patch.Edges))
		bfindex := ctx.bvIndex
		indexacc.BufferView = &bfindex
		accessors = append(accessors, indexacc)
	}

	posacc := &gltf.Accessor{}
	posacc.ComponentType = gltf.ComponentFloat
	posacc.Type = gltf.AccessorVec3
	posacc.Count = uint32(len(nd.Vertices))

	posacc.BufferView = &ctx.bvPos
	box := nd.GetBoundbox()
	posacc.Min = []float32{float32(box[0]), float32(box[1]), float32(box[2])}
	posacc.Max = []float32{float32(box[3]), float32(box[4]), float32(box[5])}
	accessors = append(accessors, posacc)

	return mesh, accessors
}

func buildMesh(ctx *buildContext, accessors []*gltf.Accessor, nd *MeshNode) (*gltf.Mesh, []*gltf.Accessor) {
	mesh := &gltf.Mesh{}
	aftIndices := uint32(len(nd.FaceGroup))
	idx := uint32(len(accessors))
	indexPos := aftIndices + idx
	var start uint32 = 0
	for i := range nd.FaceGroup {
		tmp := indexPos
		patch := nd.FaceGroup[i]
		batchId := patch.Batchid
		if batchId < 0 {
			batchId = 0
		}
		mtl_id := uint32(batchId) + ctx.mtlSize

		ps := &gltf.Primitive{}
		ps.Material = &mtl_id
		if ps.Attributes == nil {
			ps.Attributes = make(gltf.Attribute)
		}
		index := uint32(i) + idx
		ps.Indices = &index

		ps.Attributes["POSITION"] = indexPos
		if len(nd.TexCoords) > 0 {
			tmp++
			ps.Attributes["TEXCOORD_0"] = tmp
		}
		if len(nd.Normals) > 0 {
			tmp++
			ps.Attributes["NORMAL"] = tmp
		}
		ps.Mode = gltf.PrimitiveTriangles
		mesh.Primitives = append(mesh.Primitives, ps)

		indexacc := &gltf.Accessor{}
		indexacc.ComponentType = gltf.ComponentUint

		indexacc.ByteOffset = start * 12
		indexacc.Count = uint32(len(patch.Faces)) * 3
		start += uint32(len(patch.Faces))
		bfindex := ctx.bvIndex
		indexacc.BufferView = &bfindex
		accessors = append(accessors, indexacc)
	}

	posacc := &gltf.Accessor{}
	posacc.ComponentType = gltf.ComponentFloat
	posacc.Type = gltf.AccessorVec3
	posacc.Count = uint32(len(nd.Vertices))

	posacc.BufferView = &ctx.bvPos
	box := nd.GetBoundbox()
	posacc.Min = []float32{float32(box[0]), float32(box[1]), float32(box[2])}
	posacc.Max = []float32{float32(box[3]), float32(box[4]), float32(box[5])}
	accessors = append(accessors, posacc)

	if len(nd.TexCoords) > 0 {
		texacc := &gltf.Accessor{}
		texacc.ComponentType = gltf.ComponentFloat
		texacc.Type = gltf.AccessorVec2
		texacc.Count = uint32(len(nd.TexCoords))
		texacc.BufferView = &ctx.bvTex
		accessors = append(accessors, texacc)
	}

	if len(nd.Normals) > 0 {
		nlacc := &gltf.Accessor{}
		nlacc.ComponentType = gltf.ComponentFloat
		nlacc.Type = gltf.AccessorVec3
		nlacc.Count = uint32(len(nd.Normals))
		nlacc.BufferView = &ctx.bvNorm
		accessors = append(accessors, nlacc)
	}
	return mesh, accessors
}

func buildGltf(doc *gltf.Document, mh *BaseMesh, appendNode bool, exportOutline bool) error {
	ctx := &buildContext{}
	ctx.mtlSize = uint32(len(doc.Materials))

	for _, nd := range mh.Nodes {
		if appendNode {
			doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, uint32(len(doc.Nodes)))
			node := &gltf.Node{}
			l := (uint32)(len(doc.Meshes))
			node.Mesh = &l
			doc.Nodes = append(doc.Nodes, node)
		}

		if exportOutline && len(nd.EdgeGroup) > 0 {
			doc.BufferViews = buildOutlineBuffer(ctx, doc.Buffers[0], doc.BufferViews, nd)

			var mesh *gltf.Mesh
			mesh, doc.Accessors = buildOutline(ctx, doc.Accessors, nd)
			doc.Meshes = append(doc.Meshes, mesh)
		} else {
			doc.BufferViews = buildMeshBuffer(ctx, doc.Buffers[0], doc.BufferViews, nd)

			var mesh *gltf.Mesh
			mesh, doc.Accessors = buildMesh(ctx, doc.Accessors, nd)
			doc.Meshes = append(doc.Meshes, mesh)
		}
	}

	err := fillMaterials(doc, mh.Materials)
	if err != nil {
		return err
	}

	return nil
}

func buildTextureBuffer(doc *gltf.Document, buffer *gltf.Buffer, texture *Texture) (*gltf.Texture, error) {
	spCount := uint32(len(doc.Samplers))
	imCount := uint32(len(doc.Images))

	tx := &gltf.Texture{Sampler: &spCount, Source: &imCount}

	gimg := &gltf.Image{}
	gimg.MimeType = "image/png"
	imgIndex := uint32(len(doc.BufferViews))
	gimg.BufferView = &imgIndex

	img, e := LoadTexture(texture, true)
	if e != nil {
		return nil, e
	}
	var bt []byte
	buf := bytes.NewBuffer(bt)
	png.Encode(buf, img)

	imgBuffView := &gltf.BufferView{}
	imgBuffView.ByteOffset = buffer.ByteLength
	imgBuffView.ByteLength = uint32(buf.Len())
	imgBuffView.Buffer = 0
	buffer.ByteLength += uint32(buf.Len())
	buffer.Data = append(buffer.Data, buf.Bytes()...)

	doc.BufferViews = append(doc.BufferViews, imgBuffView)
	doc.Images = append(doc.Images, gimg)

	var sp *gltf.Sampler
	if texture.Repeated {
		sp = &gltf.Sampler{WrapS: gltf.WrapRepeat, WrapT: gltf.WrapRepeat}
	} else {
		sp = &gltf.Sampler{WrapS: gltf.WrapClampToEdge, WrapT: gltf.WrapClampToEdge}
	}
	doc.Samplers = append(doc.Samplers, sp)

	return tx, nil
}

func fillMaterials(doc *gltf.Document, mts []MeshMaterial) error {
	texMap := make(map[int32]uint32)
	for i := range mts {
		mtl := mts[i]

		gm := &gltf.Material{DoubleSided: true, AlphaMode: gltf.AlphaMask}
		gm.PBRMetallicRoughness = &gltf.PBRMetallicRoughness{BaseColorFactor: &[4]float32{1, 1, 1, 1}}
		var texMtl *TextureMaterial
		var cl *[4]float32
		switch ml := mtl.(type) {
		case *BaseMaterial:
			cl = &[4]float32{float32(ml.Color[0]) / 255, float32(ml.Color[1]) / 255, float32(ml.Color[2]) / 255, 1 - float32(ml.Transparency)}
		case *PbrMaterial:
			cl = &[4]float32{float32(ml.Color[0]) / 255, float32(ml.Color[1]) / 255, float32(ml.Color[2]) / 255, 1 - float32(ml.Transparency)}
			mc := float32(ml.Metallic)
			gm.PBRMetallicRoughness.MetallicFactor = &mc
			rs := float32(ml.Roughness)
			gm.PBRMetallicRoughness.RoughnessFactor = &rs
			gm.EmissiveFactor[0] = float32(ml.Emissive[0]) / 255
			gm.EmissiveFactor[1] = float32(ml.Emissive[1]) / 255
			gm.EmissiveFactor[2] = float32(ml.Emissive[2]) / 255
			texMtl = &ml.TextureMaterial
		case *LambertMaterial:
			cl = &[4]float32{float32(ml.Color[0]) / 255, float32(ml.Color[1]) / 255, float32(ml.Color[2]) / 255, 1 - float32(ml.Transparency)}
			texMtl = &ml.TextureMaterial

			spmtl := &specular.PBRSpecularGlossiness{
				DiffuseFactor: &[4]float32{float32(ml.Diffuse[0]) / 255, float32(ml.Diffuse[1]) / 255, float32(ml.Diffuse[2]) / 255, 1},
			}

			gm.EmissiveFactor[0] = float32(ml.Emissive[0]) / 255
			gm.EmissiveFactor[1] = float32(ml.Emissive[1]) / 255
			gm.EmissiveFactor[2] = float32(ml.Emissive[2]) / 255

			gm.Extensions[specular.ExtensionName] = spmtl
		case *PhongMaterial:
			cl = &[4]float32{float32(ml.Color[0]) / 255, float32(ml.Color[1]) / 255, float32(ml.Color[2]) / 255, 1 - float32(ml.Transparency)}
			texMtl = &ml.TextureMaterial

			spmtl := &specular.PBRSpecularGlossiness{
				DiffuseFactor:    &[4]float32{float32(ml.Diffuse[0]) / 255, float32(ml.Diffuse[1]) / 255, float32(ml.Diffuse[2]) / 255, 1},
				SpecularFactor:   &[3]float32{float32(ml.Specular[0]) / 255, float32(ml.Specular[1]) / 255, float32(ml.Specular[2]) / 255},
				GlossinessFactor: &ml.Shininess,
			}

			gm.EmissiveFactor[0] = float32(ml.Emissive[0]) / 255
			gm.EmissiveFactor[1] = float32(ml.Emissive[1]) / 255
			gm.EmissiveFactor[2] = float32(ml.Emissive[2]) / 255

			gm.Extensions[specular.ExtensionName] = spmtl
		case *TextureMaterial:
			texMtl = ml
			cl = &[4]float32{float32(ml.Color[0]) / 255, float32(ml.Color[1]) / 255, float32(ml.Color[2]) / 255, 1 - float32(ml.Transparency)}
		}

		if texMtl.HasTexture() {
			if idx, ok := texMap[texMtl.Texture.Id]; ok {
				gm.PBRMetallicRoughness.BaseColorTexture = &gltf.TextureInfo{Index: idx}
			} else {

				tex, err := buildTextureBuffer(doc, doc.Buffers[0], texMtl.Texture)

				if err != nil {
					return err
				}

				texIndex := uint32(len(doc.Textures))
				gm.PBRMetallicRoughness.BaseColorTexture = &gltf.TextureInfo{Index: texIndex}
				doc.Textures = append(doc.Textures, tex)
			}
		} else {
			gm.PBRMetallicRoughness.BaseColorFactor = cl
		}

		doc.Materials = append(doc.Materials, gm)
	}
	return nil
}
