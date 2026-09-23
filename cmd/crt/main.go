package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"
)

const (
	serialNumberNum = 1658
	orgName         = "My Home"
	countryCode     = "KZT"
	periodYear      = 10
	rsaLen          = 4096
	crtType         = "CERTIFICATE"
	pkType          = "RSA PRIVATE KEY"
	crtDir          = "certs"
	crtName         = "cert.pem"
	pkName          = "private.pem"
)

func main() {
	// Создание шаблона сертификата
	cert := &x509.Certificate{
		// Уникальный номер сертификата
		SerialNumber: big.NewInt(serialNumberNum),
		// Информация о владельце сертификата
		Subject: pkix.Name{
			Organization: []string{orgName},
			Country:      []string{countryCode},
		},
		// Разрешения сертификата для IP адресов
		IPAddresses: []net.IP{
			net.IPv4(127, 0, 0, 1),
			net.IPv6loopback,
		},
		// Сертификат валиден начиная с времени создания
		NotBefore: time.Now(),
		// Период валидности сертификата
		NotAfter:     time.Now().AddDate(periodYear, 0, 0),
		SubjectKeyId: []byte{1, 2, 3, 4, 5, 6},
		// Устанавливаем использование ключа для цифровой подписи,
		// а также клиентской и серверной авторизации
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}

	// Создаем приватный RSA ключ
	pk, err := rsa.GenerateKey(rand.Reader, rsaLen)
	if err != nil {
		log.Fatal(err)
	}

	// Создаем сертификат
	crtBytes, err := x509.CreateCertificate(rand.Reader, cert, cert, &pk.PublicKey, pk)
	if err != nil {
		log.Fatal(err)
	}

	// Оборачиваем сертификат в контейнер PEM
	var crtPEM bytes.Buffer
	err = pem.Encode(&crtPEM, &pem.Block{
		Type:  crtType,
		Bytes: crtBytes,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Оборачиваем ключ в контейнер PEM
	var pkPEM bytes.Buffer
	err = pem.Encode(&pkPEM, &pem.Block{
		Type:  pkType,
		Bytes: x509.MarshalPKCS1PrivateKey(pk),
	})
	if err != nil {
		log.Fatal(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	dir := filepath.Join(wd, crtDir)

	if err = os.WriteFile(filepath.Join(dir, crtName), crtPEM.Bytes(), 0o600); err != nil {
		log.Fatal(err)
	}

	if err = os.WriteFile(filepath.Join(dir, pkName), pkPEM.Bytes(), 0o600); err != nil {
		log.Fatal(err)
	}

	log.Printf("Certs were generated into %s dir\n", dir)
}
