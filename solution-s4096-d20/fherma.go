// GENERATED from ciphertext_multiplication/ckks@1.0.0. Do not edit — `--update` rewrites it.
//
// The types your answer is written against, derived from the signature: one
// field per value parameter, per argument, per result. A tensor is typed,
// because the signature already settled what its elements are.
package main

// Tensor is row-major, and its shape is the signature's.
type Tensor[T any] struct {
	Shape []int64
	Data  []T
}

// Count is how many elements the shape holds.
func (t Tensor[T]) Count() int64 {
	total := int64(1)
	for _, d := range t.Shape {
		total *= d
	}
	return total
}

// Point is the parameter set one measurement is taken at.
type Point struct {
	Slots uint32 `json:"slots"` // u32
	Depth uint32 `json:"depth"` // u32
	Security uint32 `json:"security"` // u32
	Precision uint32 `json:"precision"` // u32
}

// Inputs are the arguments, in the order the signature names them.
type Inputs struct {
	A Tensor[float64] // Tensor(dims=(Name(name='slots'),), element=Scalar(name='f64'))
	B Tensor[float64] // Tensor(dims=(Name(name='slots'),), element=Scalar(name='f64'))
}

// Outputs are the results.
type Outputs struct {
	C Tensor[float64] // Tensor(dims=(Name(name='slots'),), element=Scalar(name='f64'))
}
