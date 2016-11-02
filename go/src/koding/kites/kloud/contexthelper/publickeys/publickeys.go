package publickeys

import (
	"crypto/sha256"
	"encoding/base64"

	"golang.org/x/crypto/ssh"
	"golang.org/x/net/context"
)

type key int

const (
	publicKey key = 0

	// name of the key saved on remote provider
	DeployKeyName = "kloud-deployment"
)

type Keys struct {
	PublicKey  string
	PrivateKey string
	KeyName    string
}

func (k *Keys) Thumbprint() (string, error) {
	key, err := ssh.ParsePrivateKey([]byte(k.PrivateKey))
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(key.PublicKey().Marshal())

	return base64.RawStdEncoding.EncodeToString(hash[:]), nil
}

func FromContext(ctx context.Context) (*Keys, bool) {
	c, ok := ctx.Value(publicKey).(*Keys)
	return c, ok
}

func NewContext(ctx context.Context, keys *Keys) context.Context {
	return context.WithValue(ctx, publicKey, keys)
}
