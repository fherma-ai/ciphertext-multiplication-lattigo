# Ciphertext multiplication on Lattigo

An answer to `ciphertext-multiplication/ckks@1.0.0` on FHERMA with Lattigo
6.2.0, in Go. One directory per point, because the envelope builds the
parameters from `config.jsonc` and never reads the point.

The measured call is the two library calls the specification names:

```go
product, err := env.Evaluator.MulRelinNew(cts[0], cts[1])  // tensor, relinearise
err = env.Evaluator.Rescale(product, product)              // the level it costs
```

Lattigo has no automatic scaling technique, so the rescale is always the
author's call and always inside the window — there is nothing to hide it in.

## The parameters, written as sizes

Lattigo takes a parameter literal: the modulus chain as a list of prime
sizes, and the auxiliary modulus of the key switch as another. So the
configuration here names sizes and counts, and the envelope assembles the
lists.

| | `solution-s32768-d10` | `solution-s4096-d20` |
| --- | --- | --- |
| `log_n` | 16 | 16 |
| `mult_depth` | 10 | 20 |
| `scale_mod_size` | 50 | 48 |
| `first_mod_size` | 60 | 60 |
| `special_mod_count` × size | 4 × 60 | 6 × 60 |
| log Q · P | 800 | 1380 |

Those are the same numbers the OpenFHE rows come to at the same points —
`mult_depth` 10 at a 50-bit scale with a three-digit key switch is log QP 800
— so the two libraries are measured on one parameter set said in two
notations. The deep point takes a 48-bit scale for a reason that belongs to
neither library: the FIDESlib image carries a patched OpenFHE 1.5.1 which
refuses twenty 50-bit levels, and all four answers to this point moved
together rather than one of them differing in two things at once.

## The floor the library does not enforce

A Lattigo parameter literal carries no security level, and
`NewParametersFromLiteral` builds whatever it is handed. The envelope
therefore holds `log QP` against the HE standard's table for the ring and
refuses to seal before a key is generated, which is the guarantee OpenFHE
gives for itself. Without it this row's "128 bits" would be a claim rather
than a check — which is exactly what it is for the DESILO row, and the board
says so there.

## Building it

The image carries the library in its module cache and the build is offline:

```sh
docker run --rm --network none -v "$PWD/solution-s32768-d10":/solution -w /solution \
  fherma/lattigo:6.2.0 sh -c "go build -mod=mod -o solution . && ./solution <point dir>"
```

`-mod=mod` is not decoration: the default since Go 1.16 is `-mod=readonly`,
which refuses a module `go.sum` does not already list, and a generated
scaffold has no network to compute hashes with. With `-mod=mod` the build
writes `go.sum` from the cache it was given.
