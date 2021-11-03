package mst

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"io"

	"github.com/qmuntal/gltf"
)

const GLTF_VERSION = "2.0"

func MstToGltf(msts []*Mesh) (*gltf.Document, error) {
	doc := CreateDoc()
	for _, mst := range msts {
		e := BuildGltf(doc, mst)
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

func BuildGltf(doc *gltf.Document, mh *Mesh) error {
	mtlMap := make(map[uint32]uint32)
	var prewCreateMtlCount uint32 = 0
	meshCount := len(doc.Meshes)
	for _, nd := range mh.Nodes {
		var bt []byte
		bvSize := len(doc.BufferViews)
		buf := bytes.NewBuffer(bt)
		indecs := &gltf.BufferView{}
		indecs.ByteOffset = uint32(buf.Len())
		for _, g := range nd.FaceGroup {
			binary.Write(buf, binary.LittleEndian, g.Faces)
		}
		indecs.ByteLength = uint32(buf.Len()) - indecs.ByteOffset
		indecs.Buffer = uint32(len(doc.Buffers))
		doc.BufferViews = append(doc.BufferViews, indecs)

		buffer := &gltf.Buffer{}
		buffer.ByteLength = uint32(buf.Len())
		buffer.Data = buf.Bytes()
		buffer.EmbeddedResource()
		doc.Buffers = append(doc.Buffers, buffer)

		buf2 := bytes.NewBuffer(bt)
		postions := &gltf.BufferView{}
		postions.ByteOffset = uint32(buf2.Len())
		binary.Write(buf2, binary.LittleEndian, nd.Vertices)
		postions.ByteLength = uint32(buf2.Len()) - postions.ByteOffset
		postions.Buffer = uint32(len(doc.Buffers))
		bvPos := uint32(len(doc.BufferViews))
		doc.BufferViews = append(doc.BufferViews, postions)

		buffer2 := &gltf.Buffer{}
		buffer2.ByteLength = uint32(buf2.Len())
		buffer2.Data = buf2.Bytes()
		buffer2.EmbeddedResource()
		doc.Buffers = append(doc.Buffers, buffer2)

		texcood := &gltf.BufferView{}
		bvTexc := uint32(len(doc.BufferViews))
		if len(nd.TexCoords) > 0 {
			buf3 := bytes.NewBuffer(bt)
			texcood.ByteOffset = uint32(buf3.Len())
			binary.Write(buf3, binary.LittleEndian, nd.TexCoords)
			texcood.ByteLength = uint32(buf3.Len()) - texcood.ByteOffset
			texcood.Buffer = uint32(len(doc.Buffers))
			doc.BufferViews = append(doc.BufferViews, texcood)

			buffer3 := &gltf.Buffer{}
			buffer3.ByteLength = uint32(buf3.Len())
			buffer3.Data = buf3.Bytes()
			buffer3.EmbeddedResource()
			doc.Buffers = append(doc.Buffers, buffer3)
		}

		normalView := &gltf.BufferView{}
		bvNl := uint32(len(doc.BufferViews))
		if len(nd.Normals) > 0 {
			buf4 := bytes.NewBuffer(bt)
			normalView.ByteOffset = uint32(buf4.Len())
			binary.Write(buf4, binary.LittleEndian, nd.Normals)
			normalView.ByteLength = uint32(buf4.Len()) - normalView.ByteOffset
			normalView.Buffer = uint32(len(doc.Buffers))
			doc.BufferViews = append(doc.BufferViews, normalView)

			buffer4 := &gltf.Buffer{}
			buffer4.ByteLength = uint32(buf4.Len())
			buffer4.Data = buf4.Bytes()
			buffer4.EmbeddedResource()

			doc.Buffers = append(doc.Buffers, buffer4)
		}

		mesh := &gltf.Mesh{}
		doc.Scenes[0].Nodes = append(doc.Scenes[0].Nodes, uint32(len(doc.Nodes)))
		nde := &gltf.Node{}
		l := (uint32)(len(doc.Meshes))
		nde.Mesh = &l
		doc.Nodes = append(doc.Nodes, nde)

		aftIndices := uint32(len(nd.FaceGroup))
		idx := uint32(len(doc.Accessors))
		indexPos := aftIndices + idx
		indexTc := indexPos + 1
		indexNl := indexPos + 2
		var start uint32 = 0
		for i := range nd.FaceGroup {
			patch := nd.FaceGroup[i]
			mtl := mh.Materials[int(patch.Batchid)]
			var mtl_id uint32
			if m, ok := mtlMap[uint32(patch.Batchid)]; ok {
				mtl_id = m
			} else {
				mtl_id = uint32(len(doc.Materials)) + prewCreateMtlCount
				prewCreateMtlCount++
				mtlMap[uint32(patch.Batchid)] = mtl_id
			}
			ps := &gltf.Primitive{}
			ps.Material = &mtl_id
			if ps.Attributes == nil {
				ps.Attributes = make(gltf.Attribute)
			}
			index := uint32(i) + idx
			ps.Indices = &index

			ps.Attributes["POSITION"] = indexPos
			if len(nd.TexCoords) > 0 && mtl.HasTexture() {
				ps.Attributes["TEXCOORD_0"] = indexTc
			}
			if len(nd.Normals) > 0 {
				ps.Attributes["NORMAL"] = indexNl
			}
			ps.Mode = gltf.PrimitiveTriangles
			mtlId := uint32(patch.Batchid) + uint32(len(doc.Materials))
			ps.Material = &mtlId
			mesh.Primitives = append(mesh.Primitives, ps)

			indexacc := &gltf.Accessor{}
			indexacc.ComponentType = gltf.ComponentUint

			indexacc.ByteOffset = start * 12
			indexacc.Count = uint32(len(patch.Faces)) * 3
			start += uint32(len(patch.Faces))
			bfindex := uint32(bvSize)
			indexacc.BufferView = &bfindex
			doc.Accessors = append(doc.Accessors, indexacc)
		}

		posacc := &gltf.Accessor{}
		posacc.ComponentType = gltf.ComponentFloat
		posacc.Type = gltf.AccessorMat3
		posacc.Count = uint32(len(nd.Vertices))
		posacc.ByteOffset = postions.ByteOffset

		posacc.BufferView = &bvPos
		box := nd.GetBoundbox()
		posacc.Min = []float32{float32(box[0]), float32(box[1]), float32(box[2])}
		posacc.Max = []float32{float32(box[3]), float32(box[4]), float32(box[5])}
		doc.Accessors = append(doc.Accessors, posacc)

		if len(nd.TexCoords) > 0 {
			texacc := &gltf.Accessor{}
			texacc.ComponentType = gltf.ComponentFloat
			texacc.Type = gltf.AccessorVec2
			texacc.Count = uint32(len(nd.TexCoords))
			texacc.ByteOffset = texcood.ByteOffset
			texacc.BufferView = &bvTexc
			doc.Accessors = append(doc.Accessors, texacc)
		}

		if len(nd.Normals) > 0 {
			nlacc := &gltf.Accessor{}
			nlacc.ComponentType = gltf.ComponentFloat
			nlacc.Type = gltf.AccessorVec3
			nlacc.Count = uint32(len(nd.Normals))
			nlacc.ByteOffset = normalView.ByteOffset
			nlacc.BufferView = &bvNl
			doc.Accessors = append(doc.Accessors, nlacc)
		}
		doc.Meshes = append(doc.Meshes, mesh)
	}
	for _, inst := range mh.InstanceNode {
		meshId := inst.MeshNodeId + uint32(meshCount)
		for _, mt := range inst.Transfors {
			ay := *mt.Array()
			nd := gltf.Node{
				Mesh: &meshId,
				Matrix: [16]float32{
					float32(ay[0]), float32(ay[1]), float32(ay[2]), float32(ay[3]),
					float32(ay[4]), float32(ay[5]), float32(ay[6]), float32(ay[7]),
					float32(ay[8]), float32(ay[9]), float32(ay[10]), float32(ay[11]),
					float32(ay[12]), float32(ay[13]), float32(ay[14]), float32(ay[15]),
				},
			}
			doc.Nodes = append(doc.Nodes, &nd)
		}
	}
	e := fillMaterials(doc, mh)
	if e != nil {
		return e
	}
	return nil
}

func fillMaterials(doc *gltf.Document, mesh *Mesh) error {
	texMap := make(map[int32]uint32)
	for i := range mesh.Materials {
		mtl := mesh.Materials[i]

		gm := &gltf.Material{DoubleSided: true, AlphaMode: gltf.AlphaMask}
		gm.PBRMetallicRoughness = &gltf.PBRMetallicRoughness{}
		cl := &[4]float32{1, 1, 1, 1}
		var texMtl *TextureMaterial
		switch ml := mtl.(type) {
		case *BaseMaterial:
			cl = &[4]float32{float32(ml.Color[0] / 255), float32(ml.Color[1] / 255), float32(ml.Color[2] / 255), 1 - float32(ml.Transparency)}
		case *PbrMaterial:
			cl = &[4]float32{float32(ml.Color[0] / 255), float32(ml.Color[1] / 255), float32(ml.Color[2] / 255), 1 - float32(ml.Transparency)}
			mc := float32(ml.Metallic)
			gm.PBRMetallicRoughness.MetallicFactor = &mc
			rs := float32(ml.Roughness)
			gm.PBRMetallicRoughness.RoughnessFactor = &rs
			gm.EmissiveFactor[0] = float32(ml.Emissive[0])
			gm.EmissiveFactor[1] = float32(ml.Emissive[1])
			gm.EmissiveFactor[2] = float32(ml.Emissive[2])
			texMtl = &ml.TextureMaterial
		case *LambertMaterial:
			cl = &[4]float32{float32(ml.Color[0] / 255), float32(ml.Color[1] / 255), float32(ml.Color[2] / 255), 1 - float32(ml.Transparency)}
			texMtl = &ml.TextureMaterial
		case *PhongMaterial:
			cl = &[4]float32{float32(ml.Color[0] / 255), float32(ml.Color[1] / 255), float32(ml.Color[2] / 255), 1 - float32(ml.Transparency)}
			texMtl = &ml.TextureMaterial
		case *TextureMaterial:
			cl = &[4]float32{float32(ml.Color[0] / 255), float32(ml.Color[1] / 255), float32(ml.Color[2] / 255), 1 - float32(ml.Transparency)}
			texMtl = ml
		}

		if texMtl != nil && texMtl.Texture != nil {
			if idx, ok := texMap[texMtl.Texture.Id]; ok {
				gm.PBRMetallicRoughness.BaseColorTexture = &gltf.TextureInfo{Index: idx}
				continue
			}

			spCount := uint32(len(doc.Samplers))
			imCount := uint32(len(doc.Images))

			tx := &gltf.Texture{Sampler: &spCount, Source: &imCount}

			gimg := &gltf.Image{}
			gimg.MimeType = "image/png"
			imgIndex := uint32(len(doc.BufferViews))
			gimg.BufferView = &imgIndex

			img, e := LoadTexture(texMtl.Texture, true)
			if e != nil {
				return e
			}
			var bt []byte
			buf := bytes.NewBuffer(bt)
			png.Encode(buf, img)

			imgBuffView := &gltf.BufferView{}
			imgBuffView.ByteOffset = 0
			imgBuffView.ByteLength = uint32(buf.Len())
			imgBuffView.Buffer = uint32(len(doc.Buffers))

			buffer := &gltf.Buffer{}
			buffer.ByteLength = uint32(buf.Len())
			buffer.Data = buf.Bytes()
			buffer.EmbeddedResource()

			doc.BufferViews = append(doc.BufferViews, imgBuffView)
			doc.Buffers = append(doc.Buffers, buffer)

			doc.Images = append(doc.Images, gimg)

			var sp *gltf.Sampler
			// if texMtl.Texture.Repeated {
			sp = &gltf.Sampler{WrapS: gltf.WrapRepeat, WrapT: gltf.WrapRepeat}
			// } else {
			// 	sp = &gltf.Sampler{WrapS: gltf.ClampToEdge, WrapT: gltf.ClampToEdge}
			// }
			doc.Samplers = append(doc.Samplers, sp)

			texIndex := uint32(len(doc.Textures))
			gm.PBRMetallicRoughness.BaseColorTexture = &gltf.TextureInfo{Index: texIndex}
			doc.Textures = append(doc.Textures, tx)
		} else {
			gm.PBRMetallicRoughness.BaseColorFactor = cl
		}

		doc.Materials = append(doc.Materials, gm)
	}
	return nil
}