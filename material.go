package mst

import "github.com/flywave/go3d/vec3"

// BaseMaterial 基础材质
type BaseMaterial struct {
	Color        [3]byte `json:"color"`
	Transparency float32 `json:"transparency"`
}

func (m *BaseMaterial) HasTexture() bool {
	return false
}

func (m *BaseMaterial) GetEmissive() [3]byte {
	return [3]byte{0, 0, 0}
}

func (m *BaseMaterial) GetTexture() *Texture {
	return nil
}

func (m *BaseMaterial) GetColor() [3]byte {
	return m.Color
}

// TextureMaterial 纹理材质
type TextureMaterial struct {
	BaseMaterial
	Texture *Texture `json:"texture,omitempty"`
	Normal  *Texture `json:"normal,omitempty"`
}

func (m *TextureMaterial) HasTexture() bool {
	return m.Texture != nil
}

func (m *TextureMaterial) GetTexture() *Texture {
	return m.Texture
}

func (m *TextureMaterial) HasNormalTexture() bool {
	return m.Normal != nil
}

func (m *TextureMaterial) GetNormalTexture() *Texture {
	return m.Normal
}

type PbrMaterial struct {
	TextureMaterial
	Emissive            [3]byte `json:"emissive"`
	Metallic            float32 `json:"metallic"`
	Roughness           float32 `json:"roughness"`
	Reflectance         float32 `json:"reflectance"`
	AmbientOcclusion    float32 `json:"ambientOcclusion"`
	ClearCoat           float32 `json:"clearCoat"`
	ClearCoatRoughness  float32 `json:"clearCoatRoughness"`
	ClearCoatNormal     [3]byte `json:"clearCoatNormal"`
	Anisotropy          float32 `json:"anisotropy"`
	AnisotropyDirection vec3.T  `json:"anisotropyDirection"`
	Thickness           float32 `json:"thickness"`       // subsurface only
	SubSurfacePower     float32 `json:"subSurfacePower"` // subsurface only
	SheenColor          [3]byte `json:"sheenColor"`      // cloth only
	SubSurfaceColor     [3]byte `json:"subSurfaceColor"` // subsurface or cloth
}

func (m *PbrMaterial) GetEmissive() [3]byte {
	return m.Emissive
}

type LambertMaterial struct {
	TextureMaterial
	Ambient  [3]byte `json:"ambient"`
	Diffuse  [3]byte `json:"diffuse"`
	Emissive [3]byte `json:"emissive"`
}

type PhongMaterial struct {
	LambertMaterial
	Specular    [3]byte `json:"specular"`
	Shininess   float32 `json:"shininess"`
	Specularity float32 `json:"specularity"`
}

func (m *LambertMaterial) GetEmissive() [3]byte {
	return m.Emissive
}
