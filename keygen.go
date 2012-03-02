package dns

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"math/big"
	"strconv"
)

// Empty interface that is used as a wrapper around all possible
// private key implementations from the crypto package.
type PrivateKey interface{}

// Generate generates a DNSKEY of the given bit size.
// The public part is put inside the DNSKEY record. 
// The Algorithm in the key must be set as this will define
// what kind of DNSKEY will be generated.
// The ECDSA algorithms imply a fixed keysize, in that case
// bits should be set to the size of the algorithm.
func (r *RR_DNSKEY) Generate(bits int) (PrivateKey, error) {
	switch r.Algorithm {
	case RSAMD5, RSASHA1, RSASHA256, RSASHA1NSEC3SHA1:
		if bits < 512 || bits > 4096 {
			return nil, ErrKeySize
		}
	case RSASHA512:
		if bits < 1024 || bits > 4096 {
			return nil, ErrKeySize
		}
	case ECDSAP256SHA256Y:
		if bits != 256 {
			return nil, ErrKeySize
		}
	case ECDSAP384SHA384Y:
		if bits != 384 {
			return nil, ErrKeySize
		}
	}

	switch r.Algorithm {
	case RSAMD5, RSASHA1, RSASHA256, RSASHA512, RSASHA1NSEC3SHA1:
		priv, err := rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return nil, err
		}
		r.setPublicKeyRSA(priv.PublicKey.E, priv.PublicKey.N)
		return priv, nil
	case ECDSAP256SHA256Y, ECDSAP384SHA384Y:
		var c elliptic.Curve
		switch r.Algorithm {
		case ECDSAP256SHA256Y:
			c = elliptic.P256()
		case ECDSAP384SHA384Y:
			c = elliptic.P384()
		}
		priv, err := ecdsa.GenerateKey(c, rand.Reader)
		if err != nil {
			return nil, err
		}
		r.setPublicKeyCurve(priv.PublicKey.X, priv.PublicKey.Y)
		return priv, nil
	default:
		return nil, ErrAlg
	}
	return nil, nil // Dummy return
}

// PrivateKeyString converts a PrivateKey to a string. This
// string has the same format as the private-key-file of BIND9 (Private-key-format: v1.3). 
// It needs some info from the key (hashing, keytag), so its a method of the RR_DNSKEY.
func (r *RR_DNSKEY) PrivateKeyString(p PrivateKey) (s string) {
	switch t := p.(type) {
	case *rsa.PrivateKey:
		algorithm := strconv.Itoa(int(r.Algorithm)) + " (" + Alg_str[r.Algorithm] + ")"
		modulus := unpackBase64(t.PublicKey.N.Bytes())
		e := big.NewInt(int64(t.PublicKey.E))
		publicExponent := unpackBase64(e.Bytes())
		privateExponent := unpackBase64(t.D.Bytes())
		prime1 := unpackBase64(t.Primes[0].Bytes())
		prime2 := unpackBase64(t.Primes[1].Bytes())
		// Calculate Exponent1/2 and Coefficient as per: http://en.wikipedia.org/wiki/RSA#Using_the_Chinese_remainder_algorithm
		// and from: http://code.google.com/p/go/issues/detail?id=987
		one := big.NewInt(1)
		minusone := big.NewInt(-1)
		p_1 := big.NewInt(0).Sub(t.Primes[0], one)
		q_1 := big.NewInt(0).Sub(t.Primes[1], one)
		exp1 := big.NewInt(0).Mod(t.D, p_1)
		exp2 := big.NewInt(0).Mod(t.D, q_1)
		coeff := big.NewInt(0).Exp(t.Primes[1], minusone, t.Primes[0])

		exponent1 := unpackBase64(exp1.Bytes())
		exponent2 := unpackBase64(exp2.Bytes())
		coefficient := unpackBase64(coeff.Bytes())

		s = "Private-key-format: v1.3\n" +
			"Algorithm: " + algorithm + "\n" +
			"Modules: " + modulus + "\n" +
			"PublicExponent: " + publicExponent + "\n" +
			"PrivateExponent: " + privateExponent + "\n" +
			"Prime1: " + prime1 + "\n" +
			"Prime2: " + prime2 + "\n" +
			"Exponent1: " + exponent1 + "\n" +
			"Exponent2: " + exponent2 + "\n" +
			"Coefficient: " + coefficient + "\n"
	case *ecdsa.PrivateKey:
		s = "TODO"
	}
	return
}
