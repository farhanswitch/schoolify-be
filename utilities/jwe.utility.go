package utilities

import (
	"encoding/json"
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

// Decrypt parses and decrypts a compact JWE token, returning the original claims map.
func Decrypt(token string, secret []byte) (map[string]interface{}, error) {
	object, err := jose.ParseEncrypted(token)
	if err != nil {
		return nil, err
	}

	decrypted, err := object.Decrypt(secret)
	if err != nil {
		return nil, err
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(decrypted, &claims); err != nil {
		return nil, err
	}

	return claims, nil
}

// Encrypt generates a compact JWE token from claims using AES-256-GCM symmetric encryption.
// The secret key must be exactly 32 bytes.
func Encrypt(claims map[string]interface{}, secret []byte) (string, error) {
	encrypter, err := jose.NewEncrypter(
		jose.A256GCM,
		jose.Recipient{Algorithm: jose.DIRECT, Key: secret},
		nil,
	)
	if err != nil {
		return "", err
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	jwe, err := encrypter.Encrypt(payload)
	if err != nil {
		return "", err
	}

	return jwe.CompactSerialize()
}
