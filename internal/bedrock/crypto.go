package bedrock

import (
	"crypto/aes"
	"encoding/binary"
	"fmt"
	"os"
)

var SkinpackKey = []byte("s5s5ejuDru4uchuF2drUFuthaspAbepE")
var SkinpackIV = SkinpackKey[:16]

const Magic = 0x9BCFB9FC
const HeaderSize = 0x100

func cfb8Process(key, iv, data []byte, decrypt bool) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	out := make([]byte, len(data))
	t := make([]byte, blockSize)
	copy(t, iv)
	buf := make([]byte, blockSize)
	for i, b := range data {
		block.Encrypt(buf, t)
		out[i] = b ^ buf[0]
		if decrypt {
			t = append(t[1:], b)
		} else {
			t = append(t[1:], out[i])
		}
	}
	return out, nil
}

func CFB8Encrypt(key, iv, data []byte) ([]byte, error) {
	return cfb8Process(key, iv, data, false)
}

func CFB8Decrypt(key, iv, data []byte) ([]byte, error) {
	return cfb8Process(key, iv, data, true)
}

func ReadEncryptedHeader(path string) (uuid string, ciphertext []byte, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, err
	}
	if len(data) < HeaderSize {
		return "", nil, fmt.Errorf("file too small for header: %d bytes", len(data))
	}
	uuidLen := int(data[16])
	if 17+uuidLen > len(data) {
		return "", nil, fmt.Errorf("invalid uuid length: %d", uuidLen)
	}
	uuid = string(data[17 : 17+uuidLen])
	ciphertext = data[HeaderSize:]
	return uuid, ciphertext, nil
}

func WriteEncryptedHeaderBytes(uuidStr string, plaintext, key []byte) ([]byte, error) {
	iv := key[:16]
	ciphertext, err := CFB8Encrypt(key, iv, plaintext)
	if err != nil {
		return nil, err
	}

	uuidBytes := []byte(uuidStr)
	header := make([]byte, HeaderSize)

	binary.LittleEndian.PutUint32(header[0:4], 0)
	binary.LittleEndian.PutUint32(header[4:8], Magic)
	binary.LittleEndian.PutUint64(header[8:16], 0)

	header[16] = byte(len(uuidBytes))
	copy(header[17:], uuidBytes)

	result := make([]byte, 0, HeaderSize+len(ciphertext))
	result = append(result, header...)
	result = append(result, ciphertext...)
	return result, nil
}

func CFB8DecryptData(data, fileKey []byte) ([]byte, error) {
	if len(data) >= 8 && binary.LittleEndian.Uint32(data[4:8]) == Magic {
		ciphertext := data[HeaderSize:]
		key := SkinpackKey
		if fileKey != nil {
			key = fileKey
		}
		return CFB8Decrypt(key, key[:16], ciphertext)
	}
	if fileKey != nil {
		return CFB8Decrypt(fileKey, fileKey[:16], data)
	}
	return nil, fmt.Errorf("no header and no key provided")
}
