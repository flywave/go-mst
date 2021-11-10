package mst

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jsbin "github.com/flywave/go-3jsbin"
	"github.com/flywave/go3d/mat4"
	"github.com/flywave/go3d/quaternion"
	"github.com/flywave/go3d/vec2"
	"github.com/flywave/go3d/vec3"
	"github.com/flywave/go3d/vec4"
	"github.com/ftrvxmtrx/tga"
	"golang.org/x/image/bmp"
)

func ThreejsBin2Mst(fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	mstPath := strings.Replace(fpath, ".json", ".mst", 1)

	// if _, err := os.Stat(mstPath); !os.IsNotExist(err) {
	// 	return err
	// }
	jsobj, err := jsbin.ThreeJSObjFromJson(f)
	if err != nil {
		fmt.Println(err.Error())
	}
	binpath, _ := filepath.Split(fpath)
	binpath = filepath.Join(binpath, jsobj.BinBuffer)
	bf, _ := os.Open(binpath)
	binobj, _ := jsbin.Decode(bf)

	mesh := NewMesh()
	nd := &MeshNode{}
	mat := mat4.Ident

	var sc float32 = 1
	rot := jsobj.Topology.Rotation
	if len(rot) != 0 {
		quat := quaternion.FromVec4(&vec4.T{float32(rot[0]), float32(rot[1]), float32(rot[2]), float32(rot[3])})
		mat.AssignQuaternion(&quat)
	}
	if jsobj.Topology.Scale != 0 {
		sc = float32(jsobj.Topology.Scale)
		mat.Scaled(float32(sc))
	}
	off := jsobj.Topology.Offset
	if len(off) != 0 {
		mat.Translate(&vec3.T{float32(off[0]), float32(off[1]), float32(off[2])})
	}
	for i := range binobj.Vectilers {
		v := (*vec3.T)(&binobj.Vectilers[i])
		mat.MulVec3(v)
		nd.Vertices = append(nd.Vertices, *v)
	}

	if len(binobj.Normals) > 0 {
		for i := range binobj.Normals {
			nl := &vec3.T{float32(binobj.Normals[i][0] / 127), float32(binobj.Normals[i][1] / 127), float32(binobj.Normals[i][2] / 127)}
			if nl.IsZero() {
				nl[2] = 1
			}
			nd.Normals = append(nd.Normals, *nl)
		}
	}

	if len(binobj.UVs) > 0 {
		for i := range binobj.UVs {
			uv := (*vec2.T)(&binobj.UVs[i])
			nd.TexCoords = append(nd.TexCoords, *uv)
		}
	}

	mtlcount := len(jsobj.Materials)
	nd.FaceGroup = make([]*MeshTriangle, mtlcount)
	for i := range nd.FaceGroup {
		g := &MeshTriangle{}
		g.Batchid = int32(i)
		nd.FaceGroup[i] = g
	}
	if binobj.Header.TriFlatCount > 0 {
		mtls := binobj.FlatTriangle.Material
		for i, id := range mtls {
			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: binobj.FlatTriangle.Vertices[i],
			}
			g.Faces = append(g.Faces, f)
		}
	}
	if binobj.Header.TriFlatUVCount > 0 {
		mtls := binobj.FlatUVTriangle.Material
		for i, id := range mtls {
			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: binobj.FlatUVTriangle.Vertices[i],
				Uv:     &binobj.FlatUVTriangle.Uvs[i],
			}
			g.Faces = append(g.Faces, f)
		}
	}

	if binobj.Header.TriSmoothCount > 0 {
		mtls := binobj.SmoothTriangle.Material
		for i, id := range mtls {
			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: binobj.SmoothTriangle.Vertices[i],
				Normal: &binobj.SmoothTriangle.Normals[i],
			}
			g.Faces = append(g.Faces, f)
		}
	}

	if binobj.Header.TriSmoothUVCount > 0 {
		mtls := binobj.SmoothUVTriangle.Material
		for i, id := range mtls {
			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: binobj.SmoothUVTriangle.Vertices[i],
				Uv:     &binobj.SmoothUVTriangle.Uvs[i],
				Normal: &binobj.SmoothUVTriangle.Normals[i],
			}
			g.Faces = append(g.Faces, f)
		}
	}

	if binobj.Header.QuadFlatCount > 0 {
		mtls := binobj.FlatQuad.Material
		for i, id := range mtls {
			vt := binobj.FlatQuad.Vertices[i]

			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: [3]uint32{vt[0], vt[1], vt[2]},
			}
			g.Faces = append(g.Faces, f)
			f = &Face{
				Vertex: [3]uint32{vt[2], vt[3], vt[0]},
			}
			g.Faces = append(g.Faces, f)
		}
	}

	if binobj.Header.QuadFlatUVCount > 0 {
		mtls := binobj.FlatUVQuad.Material
		for i, id := range mtls {
			vt := binobj.FlatUVQuad.Vertices[i]
			uv := binobj.FlatUVQuad.Uvs[i]

			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: [3]uint32{vt[0], vt[1], vt[2]},
				Uv:     &[3]uint32{uv[0], uv[1], uv[2]},
			}
			g.Faces = append(g.Faces, f)
			f = &Face{
				Vertex: [3]uint32{vt[2], vt[3], vt[0]},
				Uv:     &[3]uint32{uv[2], uv[3], uv[0]},
			}
			g.Faces = append(g.Faces, f)
		}
	}

	if binobj.Header.QuadSmoothCount > 0 {
		mtls := binobj.SmoothQuad.Material
		for i, id := range mtls {
			vt := binobj.SmoothQuad.Vertices[i]
			nl := binobj.SmoothQuad.Normals[i]

			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: [3]uint32{vt[0], vt[1], vt[2]},
				Normal: &[3]uint32{nl[0], nl[1], nl[2]},
			}
			g.Faces = append(g.Faces, f)
			f = &Face{
				Vertex: [3]uint32{vt[2], vt[3], vt[0]},
				Normal: &[3]uint32{nl[2], nl[3], nl[0]},
			}
			g.Faces = append(g.Faces, f)
		}
	}

	if binobj.Header.QuadSmoothUVCount > 0 {
		mtls := binobj.SmoothUVQuad.Material
		for i, id := range mtls {
			vt := binobj.SmoothUVQuad.Vertices[i]
			nl := binobj.SmoothUVQuad.Normals[i]
			uv := binobj.SmoothUVQuad.Uvs[i]

			g := nd.FaceGroup[int(id)]
			f := &Face{
				Vertex: [3]uint32{vt[0], vt[1], vt[2]},
				Normal: &[3]uint32{nl[0], nl[1], nl[2]},
				Uv:     &[3]uint32{uv[0], uv[1], uv[2]},
			}
			g.Faces = append(g.Faces, f)
			f = &Face{
				Vertex: [3]uint32{vt[2], vt[3], vt[0]},
				Normal: &[3]uint32{nl[2], nl[3], nl[0]},
				Uv:     &[3]uint32{uv[2], uv[3], uv[0]},
			}
			g.Faces = append(g.Faces, f)
		}
	}

	for id, mtl := range jsobj.Materials {
		ml := &PbrMaterial{}
		if len(mtl.ColorDiffuse) != 0 {
			ml.Color[0] = byte(mtl.ColorDiffuse[0] * 255.0)
			ml.Color[1] = byte(mtl.ColorDiffuse[1] * 255.0)
			ml.Color[2] = byte(mtl.ColorDiffuse[2] * 255.0)
		}
		ml.Transparency = 1 - float32(mtl.Opacity)

		if mtl.MapDiffuse != "" {
			dir, _ := filepath.Split(fpath)
			tex, err := convertTex(filepath.Join(dir, mtl.MapDiffuse), id)
			if err == nil {
				ml.Texture = tex
			}
		}
		mesh.Materials = append(mesh.Materials, ml)
	}
	nd.ResortVtVn()
	mesh.Nodes = append(mesh.Nodes, nd)
	wt, _ := os.Create(mstPath)
	MeshMarshal(wt, mesh)
	wt.Close()
	return nil
}

func readDir(root, path string, ext_filter []string) ([]string, error) {
	root = filepath.Clean(root)
	path = filepath.Clean(path)
	res := []string{}
	fs, er := ioutil.ReadDir(path)
	if er != nil {
		return res, er
	}
	for _, info := range fs {
		ph := filepath.Join(path, info.Name())
		if info.IsDir() {
			ls, err := readDir(root, ph, ext_filter)
			if err != nil {
				return nil, err
			}
			res = append(res, ls...)
		} else {
			if len(ext_filter) > 0 {
				for _, ext := range ext_filter {
					et := filepath.Ext(ph)
					if et == ext {
						res = append(res, strings.Replace(ph, root, "", 1))
						break
					}
				}
			} else {
				res = append(res, strings.Replace(ph, root, "", 1))
			}
		}
	}
	return res, nil
}

func convertTex(path string, texId int) (*Texture, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	_, ft, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	f.Seek(0, 0)
	img, err := readImage(f, ft)
	if err != nil {
		return nil, err
	}
	bd := img.Bounds()
	buf := []byte{}
	for y := 0; y < bd.Dy(); y++ {
		for x := 0; x < bd.Dx(); x++ {
			cl := img.At(x, y)
			r, g, b, a := color.RGBAModel.Convert(cl).RGBA()
			buf = append(buf, byte(r), byte(g), byte(b), byte(a))
		}
	}
	_, name := filepath.Split(path)
	t := &Texture{}
	t.Id = int32(texId)
	t.Name = name
	t.Format = TEXTURE_FORMAT_RGBA
	t.Size = [2]uint64{uint64(bd.Dx()), uint64(bd.Dy())}
	t.Compressed = TEXTURE_COMPRESSED_ZLIB
	t.Data = CompressImage(buf)
	return t, nil
}

func readImage(rd io.Reader, ft string) (image.Image, error) {
	switch ft {
	case "jpeg", "jpg":
		return jpeg.Decode(rd)
	case "tga":
		return tga.Decode(rd)
	case "png":
		return png.Decode(rd)
	case "gif":
		return gif.Decode(rd)
	case "bmp":
		return bmp.Decode(rd)
	default:
		return nil, errors.New("unknow format")
	}
}
