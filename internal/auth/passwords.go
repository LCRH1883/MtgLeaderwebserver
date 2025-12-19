package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

type argon2Params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltLen     uint32
	keyLen      uint32
}

var defaultArgon2idParams = argon2Params{
	memory:      64 * 1024,
	iterations:  3,
	parallelism: 2,
	saltLen:     16,
	keyLen:      32,
}

func HashPassword(plaintext string) (string, error) {
	return hashPasswordWithParams(plaintext, defaultArgon2idParams)
}

func VerifyPassword(hash, plaintext string) (bool, error) {
	params, salt, key, err := parseArgon2idHash(hash)
	if err != nil {
		return false, err
	}

	otherKey := argon2.IDKey([]byte(plaintext), salt, params.iterations, params.memory, params.parallelism, params.keyLen)
	return subtle.ConstantTimeCompare(key, otherKey) == 1, nil
}

func hashPasswordWithParams(plaintext string, p argon2Params) (string, error) {
	salt := make([]byte, p.saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}

	key := argon2.IDKey([]byte(plaintext), salt, p.iterations, p.memory, p.parallelism, p.keyLen)

	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		p.memory,
		p.iterations,
		p.parallelism,
		b64.EncodeToString(salt),
		b64.EncodeToString(key),
	), nil
}

func parseArgon2idHash(hash string) (argon2Params, []byte, []byte, error) {
	parts := strings.Split(hash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return argon2Params{}, nil, nil, errors.New("invalid argon2id hash format")
	}
	if parts[2] != "v=19" {
		return argon2Params{}, nil, nil, errors.New("unsupported argon2 version")
	}

	var p argon2Params
	for _, kv := range strings.Split(parts[3], ",") {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			return argon2Params{}, nil, nil, errors.New("invalid argon2 params")
		}
		switch k {
		case "m":
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return argon2Params{}, nil, nil, errors.New("invalid argon2 memory param")
			}
			p.memory = uint32(n)
		case "t":
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return argon2Params{}, nil, nil, errors.New("invalid argon2 time param")
			}
			p.iterations = uint32(n)
		case "p":
			n, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				return argon2Params{}, nil, nil, errors.New("invalid argon2 parallelism param")
			}
			p.parallelism = uint8(n)
		default:
			return argon2Params{}, nil, nil, errors.New("unknown argon2 param")
		}
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return argon2Params{}, nil, nil, errors.New("invalid argon2 salt")
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return argon2Params{}, nil, nil, errors.New("invalid argon2 key")
	}
	p.saltLen = uint32(len(salt))
	p.keyLen = uint32(len(key))
	if p.saltLen == 0 || p.keyLen == 0 {
		return argon2Params{}, nil, nil, errors.New("invalid argon2 salt/key")
	}

	return p, salt, key, nil
}
