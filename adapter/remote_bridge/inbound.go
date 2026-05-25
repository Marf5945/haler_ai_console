package remote_bridge

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const EventInboundCommand = "remote_bridge:inbound_command"

const (
	InboundCommandApprove = "approve"
	InboundCommandCancel  = "cancel"
	InboundCommandStatus  = "status"
	InboundCommandStop    = "stop"
	InboundCommandUnknown = "unknown"
)

// InboundAdapterStatus lets the UI expose future platform hooks before each
// provider has a full webhook parser.
type InboundAdapterStatus struct {
	Platform  string `json:"platform"`
	Label     string `json:"label"`
	Ready     bool   `json:"ready"`
	Reason    string `json:"reason,omitempty"`
	Transport string `json:"transport,omitempty"`
}

// InboundCommandEvent is the safe, reduced message that enters the app. Raw
// platform payloads stay out of the UI and never become executable commands.
type InboundCommandEvent struct {
	ChannelID  string    `json:"channel_id"`
	Channel    string    `json:"channel"`
	SourceType string    `json:"source_type,omitempty"`
	SourceID   string    `json:"source_id,omitempty"`
	Command    string    `json:"command"`
	Text       string    `json:"text"`
	ReceivedAt time.Time `json:"received_at"`
}

type InboundParseResult struct {
	Events []InboundCommandEvent `json:"events"`
}

type lineWebhookPayload struct {
	Events []struct {
		Type       string `json:"type"`
		ReplyToken string `json:"replyToken"`
		Source     struct {
			Type    string `json:"type"`
			UserID  string `json:"userId"`
			GroupID string `json:"groupId"`
			RoomID  string `json:"roomId"`
		} `json:"source"`
		Message struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"message"`
	} `json:"events"`
}

func ListInboundAdapterStatuses() []InboundAdapterStatus {
	return []InboundAdapterStatus{
		{Platform: string(ChannelLINE), Label: ChannelLINE.Label(), Ready: true, Transport: "LINE Messaging API webhook"},
		{Platform: string(ChannelDiscord), Label: ChannelDiscord.Label(), Ready: true, Transport: "Discord Bot Gateway"},
		{Platform: string(ChannelTeams), Label: ChannelTeams.Label(), Ready: false, Reason: "interface only; bot/webhook parser pending"},
		{Platform: "messenger", Label: "Messenger", Ready: false, Reason: "interface only; Meta webhook parser pending"},
		{Platform: "wechat", Label: "WeChat", Ready: false, Reason: "interface only; WeChat official account parser pending"},
	}
}

func NormalizeInboundCommand(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	switch normalized {
	case "同意", "approve", "yes", "y", "ok":
		return InboundCommandApprove
	case "取消", "cancel", "no", "n":
		return InboundCommandCancel
	case "狀態", "状态", "status":
		return InboundCommandStatus
	case "停止", "stop":
		return InboundCommandStop
	default:
		return InboundCommandUnknown
	}
}

func VerifyLineSignature(channelSecret string, body []byte, signature string) bool {
	if strings.TrimSpace(channelSecret) == "" || strings.TrimSpace(signature) == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(channelSecret))
	_, _ = mac.Write(body)
	expected := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.TrimSpace(signature)))
}

func ParseLineInbound(channelID string, body []byte, signature string, channelSecret string) (InboundParseResult, error) {
	if !VerifyLineSignature(channelSecret, body, signature) {
		return InboundParseResult{}, fmt.Errorf("LINE signature verification failed")
	}

	var payload lineWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return InboundParseResult{}, fmt.Errorf("parse LINE webhook: %w", err)
	}

	result := InboundParseResult{Events: []InboundCommandEvent{}}
	for _, event := range payload.Events {
		if event.Type != "message" || event.Message.Type != "text" {
			continue
		}
		sourceID := event.Source.UserID
		if sourceID == "" {
			sourceID = event.Source.GroupID
		}
		if sourceID == "" {
			sourceID = event.Source.RoomID
		}
		text := strings.TrimSpace(event.Message.Text)
		result.Events = append(result.Events, InboundCommandEvent{
			ChannelID:  channelID,
			Channel:    string(ChannelLINE),
			SourceType: event.Source.Type,
			SourceID:   sourceID,
			Command:    NormalizeInboundCommand(text),
			Text:       text,
			ReceivedAt: time.Now(),
		})
	}
	return result, nil
}

func (s *Service) StoreInboundSecret(channelID string, channelSecret string) error {
	if strings.TrimSpace(channelSecret) == "" {
		return fmt.Errorf("channel secret is required")
	}
	if s.GetChannelByID(channelID) == nil {
		return fmt.Errorf("channel %s not found", channelID)
	}
	return s.secrets.Store(fmt.Sprintf("remote_bridge:%s:channel_secret", channelID), strings.TrimSpace(channelSecret))
}

func (s *Service) HandleInboundHTTPRequest(channelID string, headers http.Header, body []byte) (InboundParseResult, error) {
	binding := s.GetChannelByID(channelID)
	if binding == nil {
		return InboundParseResult{}, fmt.Errorf("channel %s not found", channelID)
	}
	if binding.Channel != ChannelLINE {
		return InboundParseResult{}, fmt.Errorf("%s inbound is not implemented yet", binding.Channel.Label())
	}
	secret, err := s.secrets.Load(fmt.Sprintf("remote_bridge:%s:channel_secret", channelID))
	if err != nil {
		return InboundParseResult{}, fmt.Errorf("LINE channel secret is not configured")
	}
	return ParseLineInbound(channelID, body, headers.Get("X-Line-Signature"), secret)
}
