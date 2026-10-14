// GENERATED for ciphertext_multiplication/ckks@1.0.0. Do not edit — `--update` rewrites it.
//
//	./solution <point directory>
//
// Reads the directory a bundle prepared, answers every case in it, and writes
// what each one cost.
//
//	manifest.json     the point, and how many cases      read
//	cases/000000/     one file per argument              read
//	out/000000/       one file per result                written
//	out/results.json  what each case cost                written
//
// Only the call to Run is timed. Reading and writing are outside the window,
// which is also why seeing the inputs early is not a way to answer early:
// nothing of yours runs during the read.
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/tuneinsight/lattigo/v6/core/rlwe"
)

// The manifest a bundle writes: the point under "point", and how many
// cases. The point is kept as raw JSON and decoded on its own, so that a
// manifest which writes the point's fields at the top level is read as
// well — the C++ driver scans the whole file and both shapes exist.
type manifest struct {
	Point json.RawMessage `json:"point"`
	Cases int             `json:"cases"`
}

func numbered(i int) string { return fmt.Sprintf("%06d", i) }

func readTensor[T any](dir, name string, shape []int64) (Tensor[T], error) {
	t := Tensor[T]{Shape: shape}
	raw, err := os.ReadFile(filepath.Join(dir, name+".bin"))
	if err != nil {
		return t, err
	}
	var zero T
	width := int(binarySize(zero))
	count := int(t.Count())
	if len(raw) != count*width {
		return t, fmt.Errorf("%s: %d bytes for %d values", name, len(raw), count)
	}
	t.Data = make([]T, count)
	if err := decode(raw, t.Data); err != nil {
		return t, err
	}
	return t, nil
}

func readOne[T any](dir, name string) (T, error) {
	var value T
	raw, err := os.ReadFile(filepath.Join(dir, name+".bin"))
	if err != nil {
		return value, err
	}
	if len(raw) != int(binarySize(value)) {
		return value, fmt.Errorf("%s: %d bytes for one value", name, len(raw))
	}
	out := []T{value}
	if err := decode(raw, out); err != nil {
		return value, err
	}
	return out[0], nil
}

func writeTensor[T any](dir, name string, t Tensor[T]) error {
	return os.WriteFile(filepath.Join(dir, name+".bin"), encode(t.Data), 0o644)
}

func writeOne[T any](dir, name string, value T) error {
	return os.WriteFile(filepath.Join(dir, name+".bin"), encode([]T{value}), 0o644)
}

// Little-endian on the wire, and on every platform this runs on.
func binarySize(v any) int64 {
	switch v.(type) {
	case float64, int64, uint64:
		return 8
	case float32, int32, uint32:
		return 4
	case int16, uint16:
		return 2
	case int8, uint8:
		return 1
	}
	return 8
}

func decode[T any](raw []byte, into []T) error {
	for i := range into {
		switch target := any(&into[i]).(type) {
		case *float64:
			*target = math.Float64frombits(binary.LittleEndian.Uint64(raw[i*8:]))
		case *float32:
			*target = math.Float32frombits(binary.LittleEndian.Uint32(raw[i*4:]))
		case *int64:
			*target = int64(binary.LittleEndian.Uint64(raw[i*8:]))
		case *uint64:
			*target = binary.LittleEndian.Uint64(raw[i*8:])
		case *int32:
			*target = int32(binary.LittleEndian.Uint32(raw[i*4:]))
		case *uint32:
			*target = binary.LittleEndian.Uint32(raw[i*4:])
		default:
			return fmt.Errorf("no wire form for %T", into[i])
		}
	}
	return nil
}

func encode[T any](values []T) []byte {
	if len(values) == 0 {
		return nil
	}
	width := int(binarySize(values[0]))
	raw := make([]byte, len(values)*width)
	for i, v := range values {
		switch value := any(v).(type) {
		case float64:
			binary.LittleEndian.PutUint64(raw[i*8:], math.Float64bits(value))
		case float32:
			binary.LittleEndian.PutUint32(raw[i*4:], math.Float32bits(value))
		case int64:
			binary.LittleEndian.PutUint64(raw[i*8:], uint64(value))
		case uint64:
			binary.LittleEndian.PutUint64(raw[i*8:], value)
		case int32:
			binary.LittleEndian.PutUint32(raw[i*4:], uint32(value))
		case uint32:
			binary.LittleEndian.PutUint32(raw[i*4:], value)
		}
	}
	return raw
}

type caseResult struct {
	Index          int      `json:"i"`
	Seconds        *float64 `json:"seconds"`
	Status         string   `json:"status"`
	CiphertextsIn  int      `json:"ciphertexts_in,omitempty"`
	CiphertextsOut int      `json:"ciphertexts_out,omitempty"`
	EncodingS      float64  `json:"encoding_s,omitempty"`
	EncryptS       float64  `json:"encrypt_s,omitempty"`
	DecryptS       float64  `json:"decrypt_s,omitempty"`
	DecodingS      float64  `json:"decoding_s,omitempty"`
	ResultLevel    *int     `json:"result_level,omitempty"`
	Note           string   `json:"note,omitempty"`
}

func crashed(i int, why string) caseResult {
	if len(why) > 200 {
		why = why[:200]
	}
	return caseResult{Index: i, Status: "crashed", Note: why}
}

// Written after every case, not at the end: a process killed on its timeout
// has still done the cases before it, and a file written once at the end
// would throw them away.
func report(out string, envelopeSeconds, initSeconds float64, cases []caseResult) {
	body := struct {
		EnvelopeS float64      `json:"envelope_s"`
		InitS     float64      `json:"init_s"`
		Cases     []caseResult `json:"cases"`
	}{envelopeSeconds, initSeconds, cases}
	raw, err := json.Marshal(body)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(out, "results.json"), raw, 0o644)
}

func main() {
	if err := answer(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func answer() error {
	if len(os.Args) < 2 {
		return fmt.Errorf("usage: %s <point directory>", os.Args[0])
	}
	root := os.Args[1]

	raw, err := os.ReadFile(filepath.Join(root, "manifest.json"))
	if err != nil {
		return err
	}
	var said manifest
	if err := json.Unmarshal(raw, &said); err != nil {
		return err
	}
	pointJSON := raw
	if len(said.Point) > 0 {
		pointJSON = said.Point
	}
	var p Point
	if err := json.Unmarshal(pointJSON, &p); err != nil {
		return fmt.Errorf("manifest.json: the point: %w", err)
	}

	outRoot := filepath.Join(root, "out")
	if err := os.MkdirAll(outRoot, 0o755); err != nil {
		return err
	}

	// The config travels with the solution, not the point; the point's own
	// copy wins when a runner lays one there.
	configPath := "config.jsonc"
	if _, err := os.Stat(filepath.Join(root, "config.jsonc")); err == nil {
		configPath = filepath.Join(root, "config.jsonc")
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return err
	}

	sealing := time.Now()
	envelope, err := Seal(cfg)
	if err != nil {
		return err
	}
	envelopeSeconds := time.Since(sealing).Seconds()
	env := envelope.Env
	fmt.Printf("ring 2^%d, %d slots, log QP %.0f bits\n", cfg.LogN, env.Slots(), env.LogQP)

	setup := time.Now()
	state := Init(&p, env)
	initSeconds := time.Since(setup).Seconds()
	defer Free(state)

	cases := make([]caseResult, 0, said.Cases)
	report(outRoot, envelopeSeconds, initSeconds, cases)

	for i := 0; i < said.Cases; i++ {
		caseDir := filepath.Join(root, "cases", numbered(i))
		answers := filepath.Join(outRoot, numbered(i))
		if err := os.MkdirAll(answers, 0o755); err != nil {
			return err
		}

		var inp Inputs
		inp.A, err = readTensor[float64](caseDir, "a", []int64{int64(p.Slots)})
		if err != nil {
			cases = append(cases, crashed(i, "reading the case: "+err.Error()))
			report(outRoot, envelopeSeconds, initSeconds, cases)
			continue
		}
		inp.B, err = readTensor[float64](caseDir, "b", []int64{int64(p.Slots)})
		if err != nil {
			cases = append(cases, crashed(i, "reading the case: "+err.Error()))
			report(outRoot, envelopeSeconds, initSeconds, cases)
			continue
		}

		mark := time.Now()
		pts, err := Encoding(env, &inp)
		if err != nil {
			cases = append(cases, crashed(i, "encoding: "+err.Error()))
			report(outRoot, envelopeSeconds, initSeconds, cases)
			continue
		}
		encodingSeconds := time.Since(mark).Seconds()

		mark = time.Now()
		cts, err := envelope.Encrypt(pts)
		if err != nil {
			cases = append(cases, crashed(i, "encrypting: "+err.Error()))
			report(outRoot, envelopeSeconds, initSeconds, cases)
			continue
		}
		encryptSeconds := time.Since(mark).Seconds()

		started := time.Now()
		out, err := Run(state, env, cts)
		seconds := time.Since(started).Seconds()
		if err != nil {
			cases = append(cases, crashed(i, "run: "+err.Error()))
			report(outRoot, envelopeSeconds, initSeconds, cases)
			continue
		}

		var level *int
		if len(out) > 0 {
			at := out[0].Level()
			level = &at
		}

		mark = time.Now()
		decrypted := envelope.Decrypt(out)
		decryptSeconds := time.Since(mark).Seconds()

		mark = time.Now()
		answer, err := Decoding(&p, env, decrypted)
		if err != nil {
			cases = append(cases, crashed(i, "decoding: "+err.Error()))
			report(outRoot, envelopeSeconds, initSeconds, cases)
			continue
		}
		decodingSeconds := time.Since(mark).Seconds()

		if err := writeTensor(answers, "c", answer.C); err != nil {
			return err
		}

		elapsed := seconds
		cases = append(cases, caseResult{
			Index:          i,
			Seconds:        &elapsed,
			Status:         "ok",
			CiphertextsIn:  len(cts),
			CiphertextsOut: len(out),
			EncodingS:      encodingSeconds,
			EncryptS:       encryptSeconds,
			DecryptS:       decryptSeconds,
			DecodingS:      decodingSeconds,
			ResultLevel:    level,
		})
		report(outRoot, envelopeSeconds, initSeconds, cases)
	}

	var _ = rlwe.Ciphertext{}
	return nil
}
