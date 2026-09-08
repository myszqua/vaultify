package vault

import (
	"crypto/x509/pkix"
	"encoding/asn1"
	"fmt"
	"testing"
)

func TestScratchASN1(_ *testing.T) {
	salt := []byte{1,2,3,4,5,6,7,8}
	pbeParamsDER, _ := asn1.Marshal(pbeParams{Salt: salt, Iterations: 100})
	fmt.Printf("pbeParams hex=%x\n", pbeParamsDER)

	epki := encryptedPrivateKeyInfo{
		AlgorithmIdentifier: pkix.AlgorithmIdentifier{
			Algorithm: oidPBEWithSHAAnd3KeyTripleDES,
			Parameters: asn1.RawValue{
				Class: 0, Tag: 16, IsCompound: true, Bytes: pbeParamsDER,
			},
		},
		EncryptedData: []byte{9,9,9,9},
	}
	kd, err := asn1.Marshal(epki)
	fmt.Printf("epki marshal err=%v hex=%x\n", err, kd)

	var parsed encryptedPrivateKeyInfo
	_, e := asn1.Unmarshal(kd, &parsed)
	fmt.Printf("epki parse err=%v paramsFullLen=%d paramsTag=%d paramsBytes=%x\n", e,
		len(parsed.AlgorithmIdentifier.Parameters.FullBytes),
		parsed.AlgorithmIdentifier.Parameters.Tag,
		parsed.AlgorithmIdentifier.Parameters.Bytes)

	var pp pbeParams
	_, e2 := asn1.Unmarshal(parsed.AlgorithmIdentifier.Parameters.FullBytes, &pp)
	fmt.Printf("params reparse err=%v salt=%x iter=%d\n", e2, pp.Salt, pp.Iterations)
}
