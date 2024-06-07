package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"time"
)

func main() {

	// Génération de la clé privée
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)

	// Création du certificat
	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Mon Organisation"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)

	// Encodage en PEM
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	})

	// Enregistrement du certificat
	certOut, _ := os.Create("cert.pem")
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	certOut.Close()

	// Vérification du certificat
	cert, _ := pem.Decode(certPEM)
	x509Cert, _ := x509.ParseCertificate(cert.Bytes)

	opts := x509.VerifyOptions{
		DNSName: "test.domain.com",
		Roots:   x509.NewCertPool(),
	}

	opts.Roots.AddCert(x509Cert)

	_, err := x509Cert.Verify(opts)
	if err != nil {
		panic("certificat invalide")
	} else {
		println("certificat valide!")
	}
}
