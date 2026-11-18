package util

import (
	"github.com/prometheus/client_golang/prometheus"
)

// legacyKeyIDLabel is the key_id label value used for decrypts served
// through the legacy AES-CBC path (rows predating the keyring, routed by
// column rather than by a stored key id).
const legacyKeyIDLabel = "legacy"

// decryptsTotal counts credential decrypts by the key id that served them,
// and is exposed on the existing /metrics endpoint via the default
// prometheus registry.
//
// During any key rotation the old id's series trending to zero is the
// contract signal that the rotation is complete and the old key can leave
// ENCRYPTION_KEYS. Note that the boot-time encryption self-test ticks every
// configured key once per process start. See the encryption migration
// runbook.
var decryptsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "sources_decrypts_total",
	Help: "Credential decrypts by encryption key id. A key id flat at zero over a full scheduling cycle means its key can be retired.",
}, []string{"key_id"})

func init() {
	prometheus.MustRegister(decryptsTotal)
}
