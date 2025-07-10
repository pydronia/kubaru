package netutils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

func BasicAuthMiddleware(user, pass string, handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		b64, ok := strings.CutPrefix(authHeader, "Basic ")
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		decodedBytes, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		correctCredentials := []byte(user + ":" + pass)
		if subtle.ConstantTimeCompare(correctCredentials, decodedBytes) == 0 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted", charset="UTF-8"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func GenerateTlsCert(hosts string) error {
	// Most of this code is from https://go.dev/src/crypto/tls/generate_cert.go
	// Generate private key
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.Add(time.Hour * 24 * 365)

	// Generate random serial number
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return fmt.Errorf("Failed to generate serial number: %w", err)
	}

	certTemplate := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"PydroCo"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	for h := range strings.SplitSeq(hosts, ",") {
		if ip := net.ParseIP(h); ip != nil {
			certTemplate.IPAddresses = append(certTemplate.IPAddresses, ip)
		} else {
			certTemplate.DNSNames = append(certTemplate.DNSNames, h)
		}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, &certTemplate, &certTemplate, priv.Public(), priv)
	if err != nil {
		return fmt.Errorf("Failed to create certificate: %w", err)
	}
	certOut, err := os.Create("cert.pem")
	if err != nil {
		return fmt.Errorf("Failed to open cert.pem for writing: %w", err)
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("Failed to encode to cert.pem: %w", err)
	}
	if err := certOut.Close(); err != nil {
		return fmt.Errorf("Error closing cert.pem: %w", err)
	}
	log.Println("Wrote cert.pem")

	keyOut, err := os.Create("key.pem")
	if err != nil {
		return fmt.Errorf("Failed to open key.pem for writing: %w", err)
	}
	privBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return fmt.Errorf("Unable to marshal private key: %w", err)
	}
	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("Failed to write data to key.pem: %w", err)
	}
	if err := keyOut.Close(); err != nil {
		return fmt.Errorf("Error closing key.pem: %w", err)
	}
	log.Print("Wrote key.pem\n")
	return nil
}

func PrintUnicastAddresses() error {
	// Go through interfaces
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("Error getting interfaces: %w", err)
	}

	fmt.Println("Valid unicast addresses:")
	for _, iface := range ifaces {
		addresses, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, address := range addresses {
			ipnet, ok := address.(*net.IPNet)
			if ok && ipnet.IP.IsGlobalUnicast() {
				fmt.Println(" ", ipnet.IP)
			}
		}
	}
	fmt.Println()
	return nil
}
