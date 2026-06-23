package remote_bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"ui_console/shared/eventbus"
)

const discordInboundMaxChars = 2000

type DiscordGatewayConfig struct {
	ChannelBindingID string
	BotToken         string
	GuildID          string
	ChannelID        string
}

type DiscordGatewayManager struct {
	mu       sync.Mutex
	service  *Service
	eventBus *eventbus.Bus
	stops    map[string]context.CancelFunc
}

type discordGatewayEnvelope struct {
	Op int             `json:"op"`
	T  string          `json:"t,omitempty"`
	S  *int64          `json:"s,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

type discordHello struct {
	HeartbeatInterval float64 `json:"heartbeat_interval"`
}

type discordMessageCreate struct {
	ID        string `json:"id"`
	GuildID   string `json:"guild_id"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
	Author    struct {
		ID  string `json:"id"`
		Bot bool   `json:"bot"`
	} `json:"author"`
}

type discordFatalGatewayError struct {
	Status  string
	Message string
}

func (e discordFatalGatewayError) Error() string {
	return e.Message
}

func NewDiscordGatewayManager(service *Service, bus *eventbus.Bus) *DiscordGatewayManager {
	return &DiscordGatewayManager{service: service, eventBus: bus, stops: map[string]context.CancelFunc{}}
}

func (m *DiscordGatewayManager) Sync(parent context.Context) {
	if m == nil || m.service == nil {
		return
	}
	configs := m.service.ListActiveDiscordGatewayConfigs()
	wanted := make(map[string]DiscordGatewayConfig, len(configs))
	for _, cfg := range configs {
		wanted[cfg.ChannelBindingID] = cfg
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	for id, stop := range m.stops {
		if _, ok := wanted[id]; !ok {
			stop()
			delete(m.stops, id)
		}
	}
	for id, cfg := range wanted {
		if _, ok := m.stops[id]; ok {
			continue
		}
		ctx, cancel := context.WithCancel(parent)
		m.stops[id] = cancel
		go m.run(ctx, cfg)
	}
}

func (m *DiscordGatewayManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, stop := range m.stops {
		stop()
		delete(m.stops, id)
	}
}

func (m *DiscordGatewayManager) run(ctx context.Context, cfg DiscordGatewayConfig) {
	delay := 2 * time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		if err := m.runOnce(ctx, cfg); err != nil {
			if fatal, ok := err.(discordFatalGatewayError); ok {
				m.eventBus.Emit("remote_bridge:discord_gateway_status", map[string]string{
					"channel_id": cfg.ChannelBindingID,
					"status":     fatal.Status,
					"error":      fatal.Message,
				})
				return
			}
			m.eventBus.Emit("remote_bridge:discord_gateway_status", map[string]string{
				"channel_id": cfg.ChannelBindingID,
				"status":     "error",
				"error":      err.Error(),
			})
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(jitterDuration(delay)):
			if delay < 60*time.Second {
				delay *= 2
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
			}
		}
	}
}

func jitterDuration(base time.Duration) time.Duration {
	// Small jitter prevents all gateway loops from reconnecting at the same time.
	factor := 0.8 + rand.Float64()*0.4
	return time.Duration(float64(base) * factor)
}

func (m *DiscordGatewayManager) runOnce(ctx context.Context, cfg DiscordGatewayConfig) error {
	header := http.Header{}
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, "wss://gateway.discord.gg/?v=10&encoding=json", header)
	if err != nil {
		return err
	}
	defer conn.Close()

	var seq *int64
	helloRaw := discordGatewayEnvelope{}
	if err := conn.ReadJSON(&helloRaw); err != nil {
		return err
	}
	var hello discordHello
	_ = json.Unmarshal(helloRaw.D, &hello)
	if hello.HeartbeatInterval <= 0 {
		hello.HeartbeatInterval = 45000
	}
	identify := map[string]interface{}{
		"op": 2,
		"d": map[string]interface{}{
			"token":   strings.TrimSpace(cfg.BotToken),
			"intents": 33281,
			"properties": map[string]string{
				"os":      "ai-console",
				"browser": "ai-console",
				"device":  "ai-console",
			},
		},
	}
	if err := conn.WriteJSON(identify); err != nil {
		return err
	}

	heartbeatDone := make(chan struct{})
	defer close(heartbeatDone)
	go func() {
		ticker := time.NewTicker(time.Duration(hello.HeartbeatInterval) * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-heartbeatDone:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = conn.WriteJSON(map[string]interface{}{"op": 1, "d": seq})
			}
		}
	}()

	m.eventBus.Emit("remote_bridge:discord_gateway_status", map[string]string{
		"channel_id": cfg.ChannelBindingID,
		"status":     "connected",
	})
	for {
		var env discordGatewayEnvelope
		if err := conn.ReadJSON(&env); err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok && closeErr.Code == 4004 {
				return discordFatalGatewayError{Status: "auth_failed", Message: "Discord authentication failed; check Bot Token"}
			}
			return err
		}
		if env.S != nil {
			seq = env.S
		}
		if env.Op == 9 {
			return discordFatalGatewayError{Status: "invalid_session", Message: "Discord gateway invalid session; re-enable the channel to retry"}
		}
		if env.Op != 0 || env.T != "MESSAGE_CREATE" {
			continue
		}
		var msg discordMessageCreate
		if err := json.Unmarshal(env.D, &msg); err != nil {
			continue
		}
		if msg.Author.Bot || msg.GuildID != cfg.GuildID || msg.ChannelID != cfg.ChannelID {
			continue
		}
		text := strings.TrimSpace(msg.Content)
		if text == "" {
			m.eventBus.Emit("remote_bridge:discord_gateway_status", map[string]string{
				"channel_id": cfg.ChannelBindingID,
				"status":     "message_content_empty",
				"error":      "Discord 已收到訊息事件，但訊息內容是空的；請在 Discord Developer Portal 的 Bot 頁面啟用 Message Content Intent",
			})
			continue
		}
		if len([]rune(text)) > discordInboundMaxChars {
			text = string([]rune(text)[:discordInboundMaxChars])
		}
		m.eventBus.Emit(EventInboundCommand, InboundCommandEvent{
			ChannelID:  cfg.ChannelBindingID,
			Channel:    string(ChannelDiscord),
			SourceType: "discord_user",
			SourceID:   msg.Author.ID,
			Command:    NormalizeInboundCommand(text),
			Text:       text,
			ReceivedAt: time.Now(),
		})
	}
}

func (s *Service) ListActiveDiscordGatewayConfigs() []DiscordGatewayConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.load(); err != nil {
		return nil
	}
	var result []DiscordGatewayConfig
	for _, binding := range s.bindings {
		if binding.Channel != ChannelDiscord || !binding.Active || binding.Revoked || binding.IsExpired() {
			continue
		}
		token, errToken := s.secrets.Load(fmt.Sprintf("remote_bridge:%s:bot_token", binding.ID))
		guildID, errGuild := s.secrets.Load(fmt.Sprintf("remote_bridge:%s:guild_id", binding.ID))
		channelID, errChannel := s.secrets.Load(fmt.Sprintf("remote_bridge:%s:channel_id", binding.ID))
		if errToken != nil || errGuild != nil || errChannel != nil {
			continue
		}
		result = append(result, DiscordGatewayConfig{
			ChannelBindingID: binding.ID,
			BotToken:         token,
			GuildID:          strings.TrimSpace(guildID),
			ChannelID:        strings.TrimSpace(channelID),
		})
	}
	return result
}
