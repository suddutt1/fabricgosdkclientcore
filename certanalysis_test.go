package fabricgosdkclientcore_test

import (
	x509 "crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"
)

func Test_AnalyzeCerts(t *testing.T) {
	certPath := "/home/suddutt1/projects/producttracer/network/crypto-config/peerOrganizations/manuf.net/users/Admin@manuf.net/msp/admincerts/" + "Admin@manuf.net-cert.pem"
	signatureBytes, err := ioutil.ReadFile(certPath)
	if err == nil {
		signatureString := string(signatureBytes)
		fmt.Printf("Signature %s", signatureString)
		pos := strings.Index(signatureString, "-----BEGIN CERTIFICATE-----")
		if pos != -1 {
			actualSignature := signatureString[pos : len(signatureString)-1]
			fmt.Printf("Only Signature %s", actualSignature)
			block, _ := pem.Decode([]byte(actualSignature))
			if block != nil {
				certificate, err := x509.ParseCertificate(block.Bytes)
				if err != nil {
					fmt.Printf("Error in parsing certificate %v", err)
				} else {
					fmt.Printf("\nCert Subject Common Name %v", certificate.Subject.CommonName)
					fmt.Printf("\nCert Issuer Common Name %v", certificate.Issuer.CommonName)
					fmt.Printf("\nCert Version %v\n", certificate.Version)
					for _, extInfo := range certificate.Extensions {
						fmt.Printf("\n%+v %+v", extInfo.Id, extInfo.Value)
					}
				}
				fmt.Println("")
			}
		}

	} else {
		fmt.Printf("Error in GetCreator %v", err)
	}
}
