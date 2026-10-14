// One CKKS ciphertext multiplication with Lattigo: tensor, relinearise,
// rescale.
//
//   inp.A: tensor<slots x f64>   (secret)
//   inp.B: tensor<slots x f64>   (secret)
//   out.C: tensor<slots x f64>   (secret) — the product slot by slot
//
// `MulRelinNew` tensors and relinearises; the rescale is a separate call in
// this library and it is ours to make, which puts all three stages the
// specification names inside the measured window. Lattigo has no automatic
// scaling technique to hide one of them in the next operation.
package main

import "github.com/tuneinsight/lattigo/v6/core/rlwe"

func Init(p *Point, env *Env) any {
	// Nothing to prepare: the envelope generated the relinearisation key, and
	// the driver has already said what the parameters came out as.
	return nil
}

func Encoding(env *Env, inp *Inputs) ([]*rlwe.Plaintext, error) {
	// Slot i holds element i, padded to the ring's slot count. Nothing is
	// folded into the layout: a layout that helped one multiplication along
	// would answer a different question than the kernel asks.
	a, err := env.EncodeReals(inp.A.Data)
	if err != nil {
		return nil, err
	}
	b, err := env.EncodeReals(inp.B.Data)
	if err != nil {
		return nil, err
	}
	return []*rlwe.Plaintext{a, b}, nil
}

func Run(state any, env *Env, cts []*rlwe.Ciphertext) ([]*rlwe.Ciphertext, error) {
	// Tensor and relinearise: the product of two degree-1 ciphertexts is
	// degree 2, and the key switch brings it back under the envelope's key.
	product, err := env.Evaluator.MulRelinNew(cts[0], cts[1])
	if err != nil {
		return nil, err
	}
	// Rescale: the product carries the scale twice over, and dividing it out
	// costs a level and leaves the answer at the operands' own scale.
	if err := env.Evaluator.Rescale(product, product); err != nil {
		return nil, err
	}
	return []*rlwe.Ciphertext{product}, nil
}

func Decoding(p *Point, env *Env, pts []*rlwe.Plaintext) (*Outputs, error) {
	values, err := env.DecodeReals(pts[0], int(p.Slots))
	if err != nil {
		return nil, err
	}
	return &Outputs{C: Tensor[float64]{Shape: []int64{int64(p.Slots)}, Data: values}}, nil
}

func Free(state any) {}
