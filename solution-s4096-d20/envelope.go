// GENERATED for ciphertext_multiplication/ckks@1.0.0. Do not edit — `--update` rewrites it.
//
// The cryptographic envelope: parameters, keys, encryption, decryption. The
// only place a secret key exists. The measured call never enters this file.
//
// Everything comes from config.jsonc. Lattigo takes its parameters as lists
// of prime sizes rather than as a depth and a scale, so the config names the
// sizes and the counts and this file assembles the lists: LogQ is the first
// modulus followed by `mult_depth` primes of `scale_mod_size`, and LogP is
// `special_mod_count` primes of `special_mod_size`, which is the auxiliary
// modulus of the hybrid key switch.
//
// The security floor is enforced here and not trusted to the author: a
// Lattigo parameter literal carries no security level and the library builds
// whatever it is handed, so the envelope holds log QP against the HE
// standard's table for the ring and refuses before a key is generated.
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
	"github.com/tuneinsight/lattigo/v6/schemes/ckks"
)

// Config is config.jsonc, parsed. Fields left out take these defaults.
type Config struct {
	Scheme          string `json:"scheme"`
	Security        string `json:"security"`
	LogN            int    `json:"log_n"`
	MultDepth       int    `json:"mult_depth"`
	ScaleModSize    int    `json:"scale_mod_size"`
	FirstModSize    int    `json:"first_mod_size"`
	SpecialModCount int    `json:"special_mod_count"`
	SpecialModSize  int    `json:"special_mod_size"`
	RotationIndexes []int  `json:"rotation_key_indexes"`
}

// The HE standard's table: the widest modulus a ring degree carries at 128
// bits classic. The envelope's floor.
var maxLogQP128Classic = map[int]float64{
	11: 54, 12: 109, 13: 218, 14: 438, 15: 881, 16: 1782, 17: 3564,
}

var comments = regexp.MustCompile(`(?m)//.*$`)

// LoadConfig reads JSONC: `//` to end of line is a comment, and nothing else
// about the file is unusual.
func LoadConfig(path string) (Config, error) {
	cfg := Config{
		Scheme:          "ckks",
		Security:        "HEStd_128_classic",
		LogN:            14,
		MultDepth:       2,
		ScaleModSize:    45,
		FirstModSize:    60,
		SpecialModCount: 1,
		SpecialModSize:  61,
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("config.jsonc: %w", err)
	}
	if err := json.Unmarshal(comments.ReplaceAll(raw, nil), &cfg); err != nil {
		return cfg, fmt.Errorf("config.jsonc: %w", err)
	}
	return cfg, nil
}

// Env is what an answer computes with: the parameters, the encoder, the
// evaluator and the public evaluation keys. No secret key is reachable from
// here.
type Env struct {
	Params    ckks.Parameters
	Encoder   *ckks.Encoder
	Evaluator *ckks.Evaluator
	Keys      *rlwe.MemEvaluationKeySet
	// What the parameters came out as, for the record.
	LogQP float64
}

// Slots is how many values one packing carries.
func (e *Env) Slots() int {
	return e.Params.MaxSlots()
}

// EncodeReals packs real values, zero-padded to the slot count, at the top of
// the ladder and the default scale. A layout that wants the imaginary lane or
// another level encodes against e.Encoder itself.
func (e *Env) EncodeReals(values []float64) (*rlwe.Plaintext, error) {
	if len(values) > e.Slots() {
		return nil, fmt.Errorf("%d values into %d slots", len(values), e.Slots())
	}
	padded := make([]float64, e.Slots())
	copy(padded, values)
	pt := ckks.NewPlaintext(e.Params, e.Params.MaxLevel())
	if err := e.Encoder.Encode(padded, pt); err != nil {
		return nil, err
	}
	return pt, nil
}

// DecodeReals reads the real lane of a decrypted packing, first count slots.
func (e *Env) DecodeReals(pt *rlwe.Plaintext, count int) ([]float64, error) {
	out := make([]float64, e.Slots())
	if err := e.Encoder.Decode(pt, out); err != nil {
		return nil, err
	}
	if count > len(out) {
		count = len(out)
	}
	return out[:count], nil
}

// Envelope is the Env an answer sees and the secret it does not.
type Envelope struct {
	Env       *Env
	encryptor *rlwe.Encryptor
	decryptor *rlwe.Decryptor
}

// Seal builds the parameters, generates the keys, and refuses a set that does
// not reach the declared security level.
func Seal(cfg Config) (*Envelope, error) {
	if cfg.Scheme != "ckks" {
		return nil, fmt.Errorf("this envelope speaks ckks, not %q", cfg.Scheme)
	}
	if cfg.Security != "HEStd_128_classic" {
		return nil, fmt.Errorf("unknown security floor %q", cfg.Security)
	}

	logQ := make([]int, 0, cfg.MultDepth+1)
	logQ = append(logQ, cfg.FirstModSize)
	for i := 0; i < cfg.MultDepth; i++ {
		logQ = append(logQ, cfg.ScaleModSize)
	}
	logP := make([]int, 0, cfg.SpecialModCount)
	for i := 0; i < cfg.SpecialModCount; i++ {
		logP = append(logP, cfg.SpecialModSize)
	}

	literal := ckks.ParametersLiteral{
		LogN:            cfg.LogN,
		LogQ:            logQ,
		LogP:            logP,
		LogDefaultScale: cfg.ScaleModSize,
	}

	params, err := ckks.NewParametersFromLiteral(literal)
	if err != nil {
		return nil, fmt.Errorf("parameters: %w", err)
	}

	// The floor, before any key exists.
	ceiling, known := maxLogQP128Classic[cfg.LogN]
	if !known {
		return nil, fmt.Errorf("log_n %d is not a standard ring degree", cfg.LogN)
	}
	if got := math.Ceil(params.LogQP()); got > ceiling {
		return nil, fmt.Errorf(
			"log QP %.0f bits needs a larger ring: 2^%d carries at most %.0f at 128 bits classic",
			got, cfg.LogN, ceiling)
	}

	kgen := rlwe.NewKeyGenerator(params)
	sk := kgen.GenSecretKeyNew()
	pk := kgen.GenPublicKeyNew(sk)
	keys := []rlwe.EvaluationKey{}
	_ = keys
	rlk := kgen.GenRelinearizationKeyNew(sk)
	evk := rlwe.NewMemEvaluationKeySet(rlk)
	if len(cfg.RotationIndexes) > 0 {
		galoisElements := make([]uint64, 0, len(cfg.RotationIndexes))
		for _, k := range cfg.RotationIndexes {
			galoisElements = append(galoisElements, params.GaloisElement(k))
		}
		for _, gk := range kgen.GenGaloisKeysNew(galoisElements, sk) {
			evk.GaloisKeys[gk.GaloisElement] = gk
		}
	}

	env := &Env{
		Params:    params,
		Encoder:   ckks.NewEncoder(params),
		Evaluator: ckks.NewEvaluator(params, evk),
		Keys:      evk,
		LogQP:     params.LogQP(),
	}
	return &Envelope{
		Env:       env,
		encryptor: rlwe.NewEncryptor(params, pk),
		decryptor: rlwe.NewDecryptor(params, sk),
	}, nil
}

// Encrypt turns the packings an answer produced into ciphertexts.
func (e *Envelope) Encrypt(pts []*rlwe.Plaintext) ([]*rlwe.Ciphertext, error) {
	out := make([]*rlwe.Ciphertext, 0, len(pts))
	for _, pt := range pts {
		ct, err := e.encryptor.EncryptNew(pt)
		if err != nil {
			return nil, err
		}
		out = append(out, ct)
	}
	return out, nil
}

// Decrypt reads the answer's ciphertexts back into packings.
func (e *Envelope) Decrypt(cts []*rlwe.Ciphertext) []*rlwe.Plaintext {
	out := make([]*rlwe.Plaintext, 0, len(cts))
	for _, ct := range cts {
		out = append(out, e.decryptor.DecryptNew(ct))
	}
	return out
}
