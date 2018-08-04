package fabricgosdkclientcore_test

import (
	"testing"

	ibputil "github.com/suddutt1/fabricgosdkclientcore"
)

const _IBP_CONFIG = `
	{
		"url": "https://ibmblockchain-staging-starter.stage1.ng.bluemix.net",
		"network_id": "n406ad74aa3da4f4bbcc6f2bbbf90194a",
		"key": "org1",
		"secret": "o4ctj1pkVpIzFVQn0htbAv9sBK1OdLQUVQLut0L0iRCb6L_ZfyTx_Tr8HvH2Cg-8"
	} 

	`

func Test_IBPClient(t *testing.T) {

	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.AddAdminCerts("org1", "org1admin1-cert", "org1-peer1", "./tmp/state-store/admin@org1-cert.pem")

}
func Test_GetIBPAdminCerts(t *testing.T) {
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	if resp := ibpClient.Getcertificates("org1-peer1"); resp != nil {
		t.Logf("\n%s\n", ibputil.PrettyPrintJSON(resp))
	}
}

func Test_GenerateCertKeyEntry(t *testing.T) {
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.GenerateCertKeyEntry("./tmp/state-store/admin@org1-cert.pem", "./tmp/msp/keystore/e932bded77486552ac36d743322c38eb6ae5ff77a3db473cad060c4fcbe3349a_sk")
}
