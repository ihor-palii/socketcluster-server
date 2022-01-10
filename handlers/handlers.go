package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/subscribers"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/utils"
	"net/http"
)

type WSMessage struct {
	CID   int             `json:"cid"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func HandleHandshakeMsg(client *subscribers.Client, msg *WSMessage, ch chan<-string) error {
	handshake := map[string]interface{}{
		"rid": msg.CID,
		"data": map[string]interface{}{
			"id":              client.Id,
			"pingTimeout":     20000,
			"isAuthenticated": false,
		},
	}
	err := client.Connection.WriteJSON(handshake)
	if err != nil {
		return err
	}

	// put pong message to channel to start first ping message sending
	ch <- "#2"
	return nil
}


type RegisterRequest struct {
	Language string `json:"language"`
}

type RegisterResponse struct {
	Message string                 `json:"message"`
	Data    []RegisterResponseData `json:"data"`
}
type RegisterResponseData struct {
	ContactUUID  string `json:"contact_uuid"`
	ContactToken string `json:"contact_token"`
	ContactUrn   string `json:"contact_urn"`
}

func HandleRegisterUser(client *subscribers.Client, msg* WSMessage) error {
	reqData := &RegisterRequest{}
	err := json.Unmarshal(msg.Data, reqData)
	if err != nil {
		return err
	}

	registerUrl := fmt.Sprintf("%s/c/wch/%s/register", client.HostApi, client.ChannelUUID)
	postBody, _ := json.Marshal(map[string]string{
		"urn":      client.Id,
		"user_token": client.UserToken,
		"language": utils.GetLanguage(reqData.Language),
	})
	req, _ := http.NewRequest(http.MethodPost, registerUrl, bytes.NewReader(postBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr, err := utils.MakeHTTPRequest(req)
	if err != nil {
		return err
	}

	registerResponse := &RegisterResponse{}
	err = json.Unmarshal(rr.Body, registerResponse)
	if err != nil {
		return err
	}

	finalData := map[string]string{
		"urn": registerResponse.Data[0].ContactUrn,
		"uuid": registerResponse.Data[0].ContactUUID,
		"token": registerResponse.Data[0].ContactToken,
	}
	finalDataEncoded, err := json.Marshal(finalData)
	if err != nil {
		return err
	}
	responseJSON := map[string]interface{}{
		"rid": msg.CID,
		"error": string(finalDataEncoded),
	}

	err = client.Connection.WriteJSON(responseJSON)
	if err != nil {
		return err
	}
	return nil
}

type GetHistoryRequest struct {
	UserToken string `json:"userToken"`
}

type GetHistoryResponse struct {
	Message string                     `json:"message"`
	Data    [][]GetHistoryResponseData `json:"data"`
}

type GetHistoryResponseData struct {
	Message     string      `json:"message"`
	Origin      string      `json:"origin"`
	Metadata    interface{} `json:"metadata"`
	Attachments interface{} `json:"attachments"`
}

func HandleGetHistory (client *subscribers.Client, msg *WSMessage) error {
	reqData := &GetHistoryRequest{}
	err := json.Unmarshal(msg.Data, reqData)
	if err != nil {
		return err
	}

	getHistoryUrl := fmt.Sprintf("%s/c/wch/%s/history", client.HostApi, client.ChannelUUID)
	postBody, _ := json.Marshal(map[string]string{"user_token": client.UserToken})
	req, _ := http.NewRequest(http.MethodPost, getHistoryUrl, bytes.NewReader(postBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rr, err := utils.MakeHTTPRequest(req)
	if err != nil {
		return err
	}

	historyResponse := &GetHistoryResponse{}
	err = json.Unmarshal(rr.Body, historyResponse)
	if err != nil {
		return err
	}

	finalDataEncoded, err := json.Marshal(historyResponse.Data[0])
	if err != nil {
		return err
	}
	responseJSON := map[string]interface{}{
		"rid": msg.CID,
		"error": string(finalDataEncoded),
	}

	err = client.Connection.WriteJSON(responseJSON)
	if err != nil {
		return err
	}
	return nil
}
