package vault

import (
	"bytes"
	"crypto/cipher"
	"crypto/des" //nolint:gosec // PKCS#12 test fixtures use 3DES
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" //nolint:gosec // PKCS#12 PBKDF test vectors use SHA-1
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/myszqua/vaultify/vault/config"
	"github.com/myszqua/vaultify/vault/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	oidDataContentType            = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}
	oidSHA1                       = asn1.ObjectIdentifier{1, 3, 14, 3, 2, 26}
	oidCertBag                    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 3}
	oidPKCS8ShroudedKeyBag        = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 10, 1, 2}
	oidCertTypeX509Certificate    = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 22, 1}
	oidPBEWithSHAAnd3KeyTripleDES = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 3}
)

type pfxPdu struct {
	Version  int
	AuthSafe contentInfo
	MacData  macData
}

type contentInfo struct {
	ContentType asn1.ObjectIdentifier
	Content     asn1.RawValue `asn1:"tag:0,explicit,optional"`
}

type macData struct {
	Mac        digestInfo
	MacSalt    []byte
	Iterations int `asn1:"optional,default:1"`
}

type digestInfo struct {
	Algorithm pkix.AlgorithmIdentifier
	Digest    []byte
}

type safeBag struct {
	ID           asn1.ObjectIdentifier
	Value        asn1.RawValue     `asn1:"tag:0,explicit"`
	Attributes   []pkcs12Attribute `asn1:"set,optional"`
}

type pkcs12Attribute struct {
	ID    asn1.ObjectIdentifier
	Value asn1.RawValue `asn1:"set"`
}

type pbeParams struct {
	Salt       []byte
	Iterations int
}

type certBag struct {
	ID   asn1.ObjectIdentifier
	Data []byte `asn1:"tag:0,explicit"`
}

type encryptedPrivateKeyInfo struct {
	AlgorithmIdentifier pkix.AlgorithmIdentifier
	EncryptedData       []byte
}

// bmpString encodes s as a UCS-2 byte string with a null terminator.
func bmpString(s string) []byte {
	var ret []byte
	for _, r := range s {
		ret = append(ret, byte(r/256), byte(r%256))
	}

	return append(ret, 0, 0)
}

func fillWithRepeats(pattern []byte, v int) []byte {
	if len(pattern) == 0 {
		return nil
	}

	outputLen := v * ((len(pattern) + v - 1) / v)

	return bytes.Repeat(pattern, (outputLen+len(pattern)-1)/len(pattern))[:outputLen]
}

// pkcs12PBKDF implements the PKCS#12 key derivation (RFC 7292 appendix B.2) with SHA-1.
func pkcs12PBKDF(salt, password []byte, iterations int, id byte, size int) []byte {
	const u, v = 20, 64

	hash := func(in []byte) []byte {
		s := sha1.Sum(in) //nolint:gosec // PKCS#12 PBKDF uses SHA-1

		return s[:]
	}

	D := bytes.Repeat([]byte{id}, v)
	S := fillWithRepeats(salt, v)
	P := fillWithRepeats(password, v)
	I := []byte{}
	I = append(I, S...)
	I = append(I, P...)

	c := (size + u - 1) / u
	A := make([]byte, 0, c*u)

	var IjBuf []byte

	for i := 0; i < c; i++ {
		Ai := hash(append(D, I...))
		for j := 1; j < iterations; j++ {
			Ai = hash(Ai)
		}

		A = append(A, Ai...)

		if i < c-1 {
			var B []byte
			for len(B) < v {
				B = append(B, Ai...)
			}

			B = B[:v]

			Bbi := new(big.Int).SetBytes(B)

			for j := 0; j < len(I)/v; j++ {
				Ij := new(big.Int).SetBytes(I[j*v : (j+1)*v])
				Ij.Add(Ij, Bbi)
				Ij.Add(Ij, big.NewInt(1))
				Ijb := Ij.Bytes()

				if len(Ijb) > v {
					Ijb = Ijb[len(Ijb)-v:]
				}

				if len(Ijb) < v {
					if IjBuf == nil {
						IjBuf = make([]byte, v)
					}

					bytesShort := v - len(Ijb)
					for k := 0; k < bytesShort; k++ {
						IjBuf[k] = 0
					}

					copy(IjBuf[bytesShort:], Ijb)
					Ijb = IjBuf
				}

				copy(I[j*v:(j+1)*v], Ijb)
			}
		}
	}

	return A[:size]
}

func pbe3DESEncrypt(salt, password []byte, iterations int, plaintext []byte) []byte {
	key := pkcs12PBKDF(salt, password, iterations, 1, 24)
	iv := pkcs12PBKDF(salt, password, iterations, 2, 8)

	psLen := 8 - len(plaintext)%8
	padded := append(append([]byte{}, plaintext...), bytes.Repeat([]byte{byte(psLen)}, psLen)...)

	block, _ := des.NewTripleDESCipher(key) //nolint:gosec // PKCS#12 uses 3DES
	out := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, padded)

	return out
}

func createTestPFX(t *testing.T, password string) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-client"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	encPassword := bmpString(password)

	salt := make([]byte, 8)
	_, _ = rand.Read(salt)

	const iterations = 100

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	certBagDER, err := asn1.Marshal(certBag{ID: oidCertTypeX509Certificate, Data: certDER})
	require.NoError(t, err)
	certBagOuter, err := asn1.Marshal([]safeBag{{
		ID:    oidCertBag,
		Value: asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: certBagDER},
	}})
	require.NoError(t, err)

	encryptedPkcs8 := pbe3DESEncrypt(salt, encPassword, iterations, pkcs8Key)
	epki := encryptedPrivateKeyInfo{
		AlgorithmIdentifier: pkix.AlgorithmIdentifier{
			Algorithm: oidPBEWithSHAAnd3KeyTripleDES,
			Parameters: asn1.RawValue{
				Class: 0, Tag: 16, IsCompound: true,
				Bytes: mustASN1(t, pbeParams{Salt: salt, Iterations: iterations}),
			},
		},
		EncryptedData: encryptedPkcs8,
	}
	keyBagDER, err := asn1.Marshal(epki)
	require.NoError(t, err)
	keyBagOuter, err := asn1.Marshal([]safeBag{{
		ID:    oidPKCS8ShroudedKeyBag,
		Value: asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: keyBagDER},
	}})
	require.NoError(t, err)

	authenticatedSafe := []contentInfo{
		{ContentType: oidDataContentType, Content: contentOctets(t, certBagOuter)},
		{ContentType: oidDataContentType, Content: contentOctets(t, keyBagOuter)},
	}
	authenticatedSafeDER, err := asn1.Marshal(authenticatedSafe)
	require.NoError(t, err)

	macSalt := make([]byte, 8)
	_, _ = rand.Read(macSalt)
	macKey := pkcs12PBKDF(macSalt, encPassword, iterations, 3, 20)
	mac := hmac.New(sha1.New, macKey)
	mac.Write(authenticatedSafeDER)
	macDigest := mac.Sum(nil)

	pfx := pfxPdu{
		Version: 3,
		AuthSafe: contentInfo{
			ContentType: oidDataContentType,
			Content:     contentOctets(t, authenticatedSafeDER),
		},
		MacData: macData{
			Mac: digestInfo{
				Algorithm: pkix.AlgorithmIdentifier{Algorithm: oidSHA1},
				Digest:    macDigest,
			},
			MacSalt:    macSalt,
			Iterations: iterations,
		},
	}

	pfxDER, err := asn1.Marshal(pfx)
	require.NoError(t, err)

	return pfxDER
}

// contentOctets returns an asn1.RawValue that marshals as an explicit [0] wrapped
// OCTET STRING of data, matching the contentInfo "Content" field structure
// ([0] EXPLICIT OCTET STRING).
func contentOctets(t *testing.T, data []byte) asn1.RawValue {
	t.Helper()

	octetTLV, err := asn1.Marshal(data)
	require.NoError(t, err)

	return asn1.RawValue{Class: 2, Tag: 0, IsCompound: true, Bytes: octetTLV}
}

func mustASN1(t *testing.T, v interface{}) []byte {
	t.Helper()

	b, err := asn1.Marshal(v)
	require.NoError(t, err)

	return b
}

func TestLoadPFXCertificate(t *testing.T) {
	t.Run("Empty env values returns nil config", func(t *testing.T) {
		cfg, err := loadPFXCertificate(
			func() (string, error) { return "", models.ErrEmptyENVField },
			func() (string, error) { return "", models.ErrEmptyENVField },
		)
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("Empty password returns nil config", func(t *testing.T) {
		cfg, err := loadPFXCertificate(
			func() (string, error) { return "/some/place", nil },
			func() (string, error) { return "", models.ErrEmptyENVField },
		)
		require.NoError(t, err)
		assert.Nil(t, cfg)
	})

	t.Run("Path getter error", func(t *testing.T) {
		_, err := loadPFXCertificate(
			func() (string, error) { return "", assert.AnError },
			func() (string, error) { return "pw", nil },
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Password getter error", func(t *testing.T) {
		_, err := loadPFXCertificate(
			func() (string, error) { return "/some/place", nil },
			func() (string, error) { return "", assert.AnError },
		)
		require.Error(t, err)
		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("Missing pfx file", func(t *testing.T) {
		_, err := loadPFXCertificate(
			func() (string, error) { return "/nonexistent/file.pfx", nil },
			func() (string, error) { return "pw", nil },
		)
		require.Error(t, err)
	})

	t.Run("Invalid pfx data", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.pfx")
		require.NoError(t, os.WriteFile(path, []byte("not a real pfx"), 0o600))

		_, err := loadPFXCertificate(
			func() (string, error) { return path, nil },
			func() (string, error) { return "pw", nil },
		)
		require.Error(t, err)
	})

	t.Run("Mismatched password errors", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "client.pfx")
		require.NoError(t, os.WriteFile(path, createTestPFX(t, "rightpass"), 0o600))

		_, err := loadPFXCertificate(
			func() (string, error) { return path, nil },
			func() (string, error) { return "wrongpass", nil },
		)
		require.Error(t, err)
	})
}

func TestNewDebugModeUsesLoggingRoundTripper(t *testing.T) {
	cfg := &config.Client{
		Address:         "https://vault.example.com:8200",
		Timeout:         time.Second,
		RetryBackoffMin: time.Millisecond,
		RetryBackoffMax: 10 * time.Millisecond,
		DebugMode:       true,
	}
	s, err := New(cfg, mockLogger{})
	require.NoError(t, err)
	require.NotNil(t, s)
	assert.NotNil(t, s.client)
}
