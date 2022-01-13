package webchat

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/greatnonprofits-nfp/ccl-chatbot/server/v2/utils"
	"net/http"
)

type WSMessage struct {
	CID   int             `json:"cid"`
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

func HandleHandshakeMsg(client *Client, msg *WSMessage) error {
	client.send <- map[string]interface{}{
		"rid": msg.CID,
		"data": map[string]interface{}{
			"id":              client.Id,
			"pingTimeout":     20000,
			"isAuthenticated": false,
		},
	}
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

func HandleRegisterUser(client *Client, msg *WSMessage) error {
	reqData := &RegisterRequest{}
	err := json.Unmarshal(msg.Data, reqData)
	if err != nil {
		return err
	}

	registerUrl := fmt.Sprintf("%s/c/wch/%s/register", client.HostApi, client.ChannelUUID)
	postBody, _ := json.Marshal(map[string]string{
		"urn":        client.Id,
		"user_token": client.UserToken,
		"language":   utils.GetLanguage(reqData.Language),
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

	response := map[string]string{
		"urn":   registerResponse.Data[0].ContactUrn,
		"uuid":  registerResponse.Data[0].ContactUUID,
		"token": registerResponse.Data[0].ContactToken,
	}
	responseEncoded, err := json.Marshal(response)
	if err != nil {
		return err
	}
	client.send <- map[string]interface{}{
		"rid":   msg.CID,
		"error": string(responseEncoded),
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

func HandleGetHistory(client *Client, msg *WSMessage) error {
	reqData := &GetHistoryRequest{}
	err := json.Unmarshal(msg.Data, reqData)
	if err != nil {
		return err
	}

	if reqData.UserToken != client.UserToken {
		return errors.New("Tokens do not match. ")
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

	responseEncoded, err := json.Marshal(historyResponse.Data[0])
	if err != nil {
		return err
	}
	client.send <- map[string]interface{}{
		"rid":   msg.CID,
		"error": string(responseEncoded),
	}
	return nil
}

type SendMessageRequest struct {
	Text     string `json:"text"`
	UserURN  string `json:"userUrn"`
	UserUUID string `json:"userUuid"`
}

func HandleSendMessageToChannel(client *Client, msg *WSMessage) error {
	reqData := &SendMessageRequest{}
	err := json.Unmarshal(msg.Data, reqData)
	if err != nil {
		return err
	}

	getHistoryUrl := fmt.Sprintf("%s/c/wch/%s/receive", client.HostApi, client.ChannelUUID)
	postBody, _ := json.Marshal(map[string]string{
		"from":           reqData.UserURN,
		"text":           reqData.Text,
		"attachment_url": "",
	})
	req, _ := http.NewRequest(http.MethodPost, getHistoryUrl, bytes.NewReader(postBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	_, err = utils.MakeHTTPRequest(req)
	if err != nil {
		return err
	}
	return nil
}

type SubscribeRequest struct {
	Channel string `json:"channel"`
}

func HandleSubscribe(client *Client, msg *WSMessage) error {
	reqData := &SubscribeRequest{}
	err := json.Unmarshal(msg.Data, reqData)
	if err != nil {
		return err
	}

	client.UserUrn = reqData.Channel
	client.hub.register <- client
	return nil
}

func HandleWSMessage(client *Client, msg *WSMessage) (error, string) {
	if msg.Event == "#handshake" {
		return HandleHandshakeMsg(client, msg), "Failed to send handshake response message:"
	} else if msg.Event == "registerUser" {
		return HandleRegisterUser(client, msg), "Failed to process register user:"
	} else if msg.Event == "getHistory" {
		return HandleGetHistory(client, msg), "Failed to get history:"
	} else if msg.Event == "sendMessageToChannel" {
		return HandleSendMessageToChannel(client, msg), "Failed to send message:"
	} else if msg.Event == "#subscribe" {
		return HandleSubscribe(client, msg), "Failed to subscribe:"
	}
	return nil, ""
}
