package fabricgosdkclientcore_test

import (
	"testing"
	"time"

	ibputil "github.com/suddutt1/fabricgosdkclientcore"
)

func Test_IBPClient(t *testing.T) {
	config := `
	{
		"url": "https://ibmblockchain-staging-starter.stage1.ng.bluemix.net",
		"network_id": "n406ad74aa3da4f4bbcc6f2bbbf90194a",
		"key": "org1",
		"secret": "o4ctj1pkVpIzFVQn0htbAv9sBK1OdLQUVQLut0L0iRCb6L_ZfyTx_Tr8HvH2Cg-8"
	} 

	`
	ibpClient := ibputil.NewIBPClient([]byte(config))
	ibpClient.AddAdminCerts("org1", "org1admin1-cert", "org1-peer1", "./tmp/state-store/admin@org1-cert.pem")
	time.Sleep(10 * time.Second)
	ibpClient.Getcertificates("org1-peer1")
}
