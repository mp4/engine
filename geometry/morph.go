// Copyright 2016 The G3N Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package geometry

import (
	"github.com/sansebasko/engine/gls"
	"github.com/sansebasko/engine/math32"
	"sort"
	"strconv"
)

// MorphGeometry represents a base geometry and its morph targets.
type MorphGeometry struct {
	BaseGeometry *Geometry   // The base geometry
	Targets      []*Geometry // The morph target geometries (containing deltas)
	Weights      []float32   // The weights for each morph target
	UniWeights   gls.Uniform // Texture unit uniform location cache
	MorphGeom    *Geometry   // Cache of the last CPU-morphed geometry
}

// MaxActiveMorphTargets is the maximum number of active morph targets.
const MaxActiveMorphTargets = 8

// NewMorphGeometry creates and returns a pointer to a new MorphGeometry.
func NewMorphGeometry(baseGeometry *Geometry) *MorphGeometry {

	mg := MorphGeometry{}
	mg.BaseGeometry = baseGeometry

	mg.Targets = make([]*Geometry, 0)
	mg.Weights = make([]float32, 0)

	mg.BaseGeometry.ShaderDefines.Set("MORPHTARGETS", strconv.Itoa(MaxActiveMorphTargets))
	mg.UniWeights.Init("morphTargetInfluences")
	return &mg
}

// GetGeometry satisfies the IGeometry interface.
func (mg *MorphGeometry) GetGeometry() *Geometry {

	return mg.BaseGeometry
}

// SetWeights sets the morph target weights.
func (mg *MorphGeometry) SetWeights(weights []float32) {

	if len(weights) != len(mg.Weights) {
		panic("weights have invalid length")
	}
	mg.Weights = weights
}


// AddMorphTargets add multiple morph targets to the morph geometry.
// Morph target deltas are calculated internally and the morph target geometries are altered to hold the deltas instead.
func (mg *MorphGeometry) AddMorphTargets(morphTargets ...*Geometry) {

	for i := range morphTargets {
		mg.Weights = append(mg.Weights, 0)
		// Calculate deltas for VertexPosition
		vertexIdx := 0
		baseVertices := mg.BaseGeometry.VBO(gls.VertexPosition).Buffer()
		morphTargets[i].OperateOnVertices(func(vertex *math32.Vector3) bool {
			var baseVertex math32.Vector3
			baseVertices.GetVector3(vertexIdx*3, &baseVertex)
			vertex.Sub(&baseVertex)
			vertexIdx++
			return false
		})
		// Calculate deltas for VertexNormal if attribute is present in target geometry
		// It is assumed that if VertexNormals are present in a target geometry then they are also present in the base geometry
		normalIdx := 0
		baseNormalsVBO := mg.BaseGeometry.VBO(gls.VertexNormal)
		if baseNormalsVBO != nil {
			baseNormals := baseNormalsVBO.Buffer()
			morphTargets[i].OperateOnVertexNormals(func(normal *math32.Vector3) bool {
				var baseNormal math32.Vector3
				baseNormals.GetVector3(normalIdx*3, &baseNormal)
				normal.Sub(&baseNormal)
				normalIdx++
				return false
			})
		}
		// TODO Calculate deltas for VertexTangents
	}
	mg.Targets = append(mg.Targets, morphTargets...)

	// Update all target attributes if we have few enough that we are able to send them
	// all to the shader without sorting and choosing the ones with highest current weight
	if len(mg.Targets) <= MaxActiveMorphTargets {
		mg.UpdateTargetAttributes(mg.Targets)
	}

}

// AddMorphTargetDeltas add multiple morph target deltas to the morph geometry.
func (mg *MorphGeometry) AddMorphTargetDeltas(morphTargetDeltas ...*Geometry) {

	for range morphTargetDeltas {
		mg.Weights = append(mg.Weights, 0)
	}
	mg.Targets = append(mg.Targets, morphTargetDeltas...)

	// Update all target attributes if we have few enough that we are able to send them
	// all to the shader without sorting and choosing the ones with highest current weight
	if len(mg.Targets) <= MaxActiveMorphTargets {
		mg.UpdateTargetAttributes(mg.Targets)
	}
}

// ActiveMorphTargets sorts the morph targets by weight and returns the top n morph targets with largest weight.
func (mg *MorphGeometry) ActiveMorphTargets() ([]*Geometry, []float32) {

	numTargets := len(mg.Targets)
	if numTargets == 0 {
		return nil, nil
	}

	if numTargets <= MaxActiveMorphTargets {
		// No need to sort - just return the targets and weights directly
		return mg.Targets, mg.Weights
	} else {
		// Need to sort them by weight and only return the top N morph targets with largest weight (N = MaxActiveMorphTargets)
		// TODO test this (more than [MaxActiveMorphTargets] morph targets)
		sortedMorphTargets := make([]*Geometry, numTargets)
		copy(sortedMorphTargets, mg.Targets)
		sort.Slice(sortedMorphTargets, func(i, j int) bool {
			return mg.Weights[i] > mg.Weights[j]
		})

		sortedWeights := make([]float32, numTargets)
		copy(sortedWeights, mg.Weights)
		sort.Slice(sortedWeights, func(i, j int) bool {
			return mg.Weights[i] > mg.Weights[j]
		})
		return sortedMorphTargets, sortedWeights
	}
}

// SetIndices sets the indices array for this geometry.
func (mg *MorphGeometry) SetIndices(indices math32.ArrayU32) {

	mg.BaseGeometry.SetIndices(indices)
	for i := range mg.Targets {
		mg.Targets[i].SetIndices(indices)
	}
}

// ComputeMorphed computes a morphed geometry from the provided morph target weights.
// Note that morphing is usually computed by the GPU in shaders.
// This CPU implementation allows users to obtain an instance of a morphed geometry
// if so desired (loosing morphing ability).
func (mg *MorphGeometry) ComputeMorphed(weights []float32) *Geometry {

	morphed := NewGeometry()
	// TODO
	return morphed
}

// Dispose releases, if possible, OpenGL resources, C memory
// and VBOs associated with the base geometry and morph targets.
func (mg *MorphGeometry) Dispose() {

	mg.BaseGeometry.Dispose()
	for i := range mg.Targets {
		mg.Targets[i].Dispose()
	}
}

// UpdateTargetAttributes updates the attribute names of the specified morph targets in order.
func (mg *MorphGeometry) UpdateTargetAttributes(morphTargets []*Geometry) {

	for i, mt := range morphTargets {
		mt.SetAttributeName(gls.VertexPosition, "MorphPosition"+strconv.Itoa(i))
		mt.SetAttributeName(gls.VertexNormal, "MorphNormal"+strconv.Itoa(i))
		mt.SetAttributeName(gls.VertexTangent, "MorphTangent"+strconv.Itoa(i))
	}
}

// RenderSetup is called by the renderer before drawing the geometry.
func (mg *MorphGeometry) RenderSetup(gs *gls.GLS) {

	mg.BaseGeometry.RenderSetup(gs)

	// Sort weights and find top 8 morph targets with largest current weight (8 is the max sent to shader)
	activeMorphTargets, activeWeights := mg.ActiveMorphTargets()

	// If the morph geometry has more targets than the shader supports we need to update attribute names
	// as weights change - we only send the top morph targets with highest weights
	if len(mg.Targets) > MaxActiveMorphTargets {
		mg.UpdateTargetAttributes(activeMorphTargets)
	}

	// Transfer morphed geometry VBOs
	for _, mt := range activeMorphTargets {
		for _, vbo := range mt.VBOs() {
			vbo.Transfer(gs)
		}
	}

	// Transfer active weights uniform
	location := mg.UniWeights.Location(gs)
	gs.Uniform1fv(location, int32(len(activeWeights)), activeWeights)
}
