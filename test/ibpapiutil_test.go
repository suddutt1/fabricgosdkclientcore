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

func Test_IBPAddAdminCert(t *testing.T) {

	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.AddAdminCerts("org1", "org1remote-admin-cert", "org1-peer1", "./tmp/state-store/suddutt6@org1-cert.pem")

}
func Test_GetIBPAdminCerts(t *testing.T) {
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	if resp := ibpClient.Getcertificates("org1-peer1"); resp != nil {
		t.Logf("\n%s\n", ibputil.PrettyPrintJSON(resp))
	}
}

func Test_GenerateCertKeyEntry(t *testing.T) {
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.GenerateCertKeyEntry("./tmp/state-store/suddutt6@org1-cert.pem", "./tmp/msp/keystore/0de283c12fd24a28414580aa806ff11054bcd8da601182d4aff0573774a13f9a_sk")
}
func Test_StopPeer(t *testing.T) {
	ibputil.SetVerbose(true)
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.StopPeer("org1-peer1")
}
func Test_StartPeer(t *testing.T) {
	ibputil.SetVerbose(true)
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.StartPeer("org1-peer1")
}
func Test_SyncChannel(t *testing.T) {
	ibputil.SetVerbose(true)
	ibpClient := ibputil.NewIBPClient([]byte(_IBP_CONFIG))
	ibpClient.SyncChannel("defaultchannel")
}
