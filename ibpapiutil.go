package fabricgosdkclientcore

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

//IBPClient structure represents IBP Client details and configurations
type IBPClient struct {
	URL       string `json:"url"`
	NetworkID string `json:"network_id"`
	Key       string `json:"key"`
	Secret    string `json:"secret"`
}

//NewIBPClient creates a new construcor
func NewIBPClient(configJSON []byte) *IBPClient {

	ibpClient := new(IBPClient)
	json.Unmarshal(configJSON, &ibpClient)
	return ibpClient
}

//Getcertificates returns all the admin certificates
func (ibpc *IBPClient) Getcertificates(peerID string) []byte {
	postBodyTemplate := `
	{
		"peer_names": [
		  "{peerID}"
		]
	  }
	`
	postBody := strings.Replace(postBodyTemplate, "{peerID}", peerID, -1)
	url := ibpc.constructURL("/networks/{networkID}/certificates/fetch")

	if isOk, response := ibpc.postRequest(url, postBody); isOk {
		return response
	}
	return nil
}
func (ibpc *IBPClient) AddAdminCerts(orgMSPID, certName, peerID, certPath string) {

	certBytes, _ := ioutil.ReadFile(certPath)
	requestObj := make(map[string]interface{})
	requestObj["msp_id"] = orgMSPID
	requestObj["adminCertName"] = certName
	requestObj["adminCertificate"] = string(certBytes)
	requestObj["peer_names"] = []string{peerID}
	requestObj["SKIP_CACHE"] = true

	postBodyBytes, _ := json.MarshalIndent(requestObj, "", "  ")
	fmt.Println(string(postBodyBytes))
	url := ibpc.constructURL("/networks/{networkID}/certificates")
	ibpc.postRequest(url, string(postBodyBytes))

}

//StopPeer stops a peer
func (ibpc *IBPClient) StopPeer(peerID string) {
	url := ibpc.constructURL("/networks/{networkID}/nodes/" + peerID + "/stop")
	ibpc.postRequest(url, "{}")
}

//StartPeer starts the peer
func (ibpc *IBPClient) StartPeer(peerID string) {
	url := ibpc.constructURL("/networks/{networkID}/nodes/" + peerID + "/start")
	ibpc.postRequest(url, "{}")
}

//SyncChannel syncs the channel certificates
func (ibpc *IBPClient) SyncChannel(channelID string) {
	url := ibpc.constructURL("/networks/{networkID}/channels/" + channelID + "/sync")
	ibpc.postRequest(url, "{}")
}
func (ibpc *IBPClient) constructURL(api string) string {
	actualAPI := strings.Replace(api, "{networkID}", ibpc.NetworkID, 1)
	finalURL := fmt.Sprintf("%s/api/v1%s", ibpc.URL, actualAPI)
	return finalURL

}

func (ibpc *IBPClient) postRequest(url, json string) (bool, []byte) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	fmt.Println("URL: ", url)

	postReq, _ := http.NewRequest("POST", url, strings.NewReader(json))

	header := map[string][]string{
		"Content-Type":  {"application/json"},
		"accept":        {"application/json"},
		"authorization": {ibpc.getAuthHeaderValue()},
	}
	postReq.Header = header

	resp, err := client.Do(postReq)
	if err != nil {
		fmt.Printf("Error in response %v\n", err)
		return false, nil
	}

	responseString, _ := ioutil.ReadAll(resp.Body)
	if isVerbose() {
		fmt.Printf("Status : %s , Response : \n%s\n", resp.Status, responseString)
	}
	if resp.StatusCode == 200 {
		return true, responseString
	}
	return false, nil
}
func (ibpc *IBPClient) getAuthHeaderValue() string {
	auth := ibpc.Key + ":" + ibpc.Secret
	encodedStr := base64.StdEncoding.EncodeToString([]byte(auth))
	authHeaderValue := fmt.Sprintf("Basic %s", encodedStr)
	return authHeaderValue
}

//GeneratyeCertKeyEntry prints the cert key value entry
func (ibpc *IBPClient) GenerateCertKeyEntry(certPath, privKeyPath string) {
	certBytes, _ := ioutil.ReadFile(certPath)
	keyBytes, _ := ioutil.ReadFile(privKeyPath)
	output := make(map[string]interface{})
	pemCert := make(map[string]string)
	pemCert["pem"] = string(certBytes)
	output["cert"] = pemCert
	pemKey := make(map[string]interface{})
	pemKey["pem"] = string(keyBytes)
	output["key"] = pemKey
	finalOutput, _ := json.MarshalIndent(output, "", " ")
	fmt.Println(string(finalOutput))
}

func isVerbose() bool {
	if len(os.Getenv("VERBOSE")) > 0 && strings.EqualFold(os.Getenv("VERBOSE"), "TRUE") {
		return true
	}
	return false
}

//SetVerbose set the verbose output option to true or false
func SetVerbose(flag bool) {
	if flag {
		os.Setenv("VERBOSE", "true")
	} else {
		os.Unsetenv("VERBOSE")
	}
}

//PrettyPrintJSON pretty prints the input bytes if it is a json string
func PrettyPrintJSON(input []byte) []byte {
	var genericObject interface{}
	err := json.Unmarshal(input, &genericObject)
	if err != nil {
		fmt.Println("Can not pretty print")
		return nil
	}
	output, err := json.MarshalIndent(genericObject, "", "  ")
	if err != nil {
		fmt.Println("Can not pretty print")
		return nil
	}
	return output
}
