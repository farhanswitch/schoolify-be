package utilities

import (
	"errors"

	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

// EncryptAES will encrypt the claims into JWE token using AES-256 algorithm.
// 'secretKey' must consist of 32 byte.
func JWEEncryptAES(claims map[string]interface{}, secretKey []byte) (string, error) {
	// Validating AES-256 key
	if len(secretKey) != 32 {
		return "", errors.New("secret key must be 32 bytes long for AES-256")
	}

	encrypter, err := jose.NewEncrypter(
		jose.A256GCM, // Content encryption
		jose.Recipient{
			Algorithm: jose.A256GCMKW, // Key encryption
			Key:       secretKey,
		},
		(&jose.EncrypterOptions{}).WithContentType("JWT"),
	)
	if err != nil {
		return "", err
	}

	// Create and encrypting JWT
	jweToken, err := jwt.Encrypted(encrypter).Claims(claims).CompactSerialize()
	if err != nil {
		return "", err
	}

	return jweToken, nil
}

// DecryptAES will decrypt JWE token that encrypted using AES-256 algorithm.
func JWEDecryptAES(jweToken string, secretKey []byte) (map[string]interface{}, error) {

	if len(secretKey) != 32 {
		return nil, errors.New("secret key must be 32 bytes long for AES-256")
	}

	parsed, err := jwt.ParseEncrypted(jweToken)
	if err != nil {
		return nil, err
	}

	// Decrypting the claims using secret key
	var decryptedClaims map[string]interface{}
	err = parsed.Claims(secretKey, &decryptedClaims)
	if err != nil {
		return nil, err
	}

	return decryptedClaims, nil
}
