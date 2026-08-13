package storage

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidSignature = errors.New("invalid or expired signed URL")

// LocalStore keeps files on disk privately. Access only via short-lived signed paths.
type LocalStore struct {
	root   string
	secret []byte
}

func NewLocalStore(root, secret string) (*LocalStore, error) {
	if root == "" {
		root = filepath.Join(os.TempDir(), "homegauge-docs")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	if secret == "" {
		secret = "dev-only-change-me-homegauge-session"
	}
	return &LocalStore{root: root, secret: []byte(secret)}, nil
}

func (s *LocalStore) Put(ctx context.Context, key string, r io.Reader) (size int64, checksum string, err error) {
	_ = ctx
	key = cleanKey(key)
	full := filepath.Join(s.root, key)
	if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
		return 0, "", err
	}
	tmp := full + ".tmp"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return 0, "", err
	}
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(f, h), r)
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return 0, "", err
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return 0, "", closeErr
	}
	if err := os.Rename(tmp, full); err != nil {
		_ = os.Remove(tmp)
		return 0, "", err
	}
	return n, hex.EncodeToString(h.Sum(nil)), nil
}

func (s *LocalStore) Open(ctx context.Context, key string) (*os.File, error) {
	_ = ctx
	return os.Open(filepath.Join(s.root, cleanKey(key)))
}

func (s *LocalStore) Sign(key string, ttl time.Duration) (token string, expiresAt time.Time) {
	exp := time.Now().Add(ttl)
	payload := fmt.Sprintf("%s|%d", cleanKey(key), exp.Unix())
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	token = base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + sig
	return token, exp
}

func (s *LocalStore) Verify(token string) (key string, err error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return "", ErrInvalidSignature
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInvalidSignature
	}
	payload := string(raw)
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(payload))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return "", ErrInvalidSignature
	}
	bits := strings.Split(payload, "|")
	if len(bits) != 2 {
		return "", ErrInvalidSignature
	}
	unix, err := strconv.ParseInt(bits[1], 10, 64)
	if err != nil || time.Now().Unix() > unix {
		return "", ErrInvalidSignature
	}
	return bits[0], nil
}

func NewObjectKey(userID, docType string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("users/%s/%s/%s-%s", userID, docType, time.Now().Format("20060102"), hex.EncodeToString(b)), nil
}

func cleanKey(key string) string {
	key = strings.ReplaceAll(key, "\\", "/")
	key = strings.TrimPrefix(key, "/")
	return filepath.Clean(key)
}
