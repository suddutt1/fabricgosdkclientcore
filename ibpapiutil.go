package fabricgosdkclientcore

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
func (ibpc *IBPClient) Getcertificates(peerID string) {
	postBodyTemplate := `
	{
		"peer_names": [
		  "{peerID}"
		]
	  }
	`
	postBody := strings.Replace(postBodyTemplate, "{peerID}", peerID, -1)
	url := ibpc.constructURL("/networks/{networkID}/certificates/fetch")

	ibpc.postRequest(url, postBody)
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
func (ibpc *IBPClient) constructURL(api string) string {
	actualAPI := strings.Replace(api, "{networkID}", ibpc.NetworkID, 1)
	finalURL := fmt.Sprintf("%s/api/v1%s", ibpc.URL, actualAPI)
	return finalURL

}

func (ibpc *IBPClient) postRequest(url, json string) bool {
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
		return false
	}

	responseString, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Status : %s , Response : %s\n", resp.Status, responseString)
	if resp.StatusCode == 200 {
		return true
	}
	return false
}
func (ibpc *IBPClient) getAuthHeaderValue() string {
	auth := ibpc.Key + ":" + ibpc.Secret
	encodedStr := base64.StdEncoding.EncodeToString([]byte(auth))
	authHeaderValue := fmt.Sprintf("Basic %s", encodedStr)
	return authHeaderValue
}
