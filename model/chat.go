package model

import "time"

const (
	ConversationBank   = "bank"
	ConversationDirect = "direct"

	MaxMessageLength = 2000
)

type ChatUser struct {
	Id        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type Message struct {
	Id             int64     `json:"id"`
	ConversationId int       `json:"conversation_id"`
	SenderId       int       `json:"sender_id"`
	SenderName     string    `json:"sender_name"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

type Conversation struct {
	Id            int        `json:"id"`
	Type          string     `json:"type"`
	BloodBankId   int        `json:"blood_bank_id,omitempty"`
	BloodBankName string     `json:"blood_bank_name,omitempty"`
	PostId        int        `json:"post_id,omitempty"`
	PostTitle     string     `json:"post_title,omitempty"`
	OtherUser     *ChatUser  `json:"other_user,omitempty"`
	LastMessage   *Message   `json:"last_message"`
	UnreadCount   int        `json:"unread_count"`
	LastMessageAt *time.Time `json:"last_message_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// DirectConversationRequest is the body of POST /conversation/direct.
type DirectConversationRequest struct {
	PostId int `json:"post_id"`
	UserId int `json:"user_id"`
}

// SendMessageRequest is the body of POST /conversation/{id}/message.
type SendMessageRequest struct {
	Body     string `json:"body"`
	ClientId string `json:"client_id"`
}

// Notification is pushed to online users over the websocket.
type Notification struct {
	Kind  string         `json:"kind"`
	Title string         `json:"title"`
	Body  string         `json:"body"`
	Data  map[string]any `json:"data,omitempty"`
}

// Notification kinds
const (
	NotifyBankApproved      = "bank_approved"
	NotifyBankRejected      = "bank_rejected"
	NotifyBankSuspended     = "bank_suspended"
	NotifyJoinRequested     = "join_requested"
	NotifyJoinApproved      = "join_approved"
	NotifyJoinRejected      = "join_rejected"
	NotifyMemberRemoved     = "member_removed"
	NotifyRoleChanged       = "role_changed"
	NotifyAdminTransferred  = "admin_transferred"
	NotifyDonationConfirmed = "donation_confirmed"
	NotifyBadgeEarned       = "badge_earned"
	NotifyPostInterest      = "post_interest"
)

// WSEvent is a message sent from the server to a websocket client.
type WSEvent struct {
	Type           string        `json:"type"` // message | typing | read | notification | error
	ConversationId int           `json:"conversation_id,omitempty"`
	Message        *Message      `json:"message,omitempty"`
	UserId         int           `json:"user_id,omitempty"`
	UserName       string        `json:"user_name,omitempty"`
	MessageId      int64         `json:"message_id,omitempty"`
	ClientId       string        `json:"client_id,omitempty"`
	Notification   *Notification `json:"notification,omitempty"`
	Error          string        `json:"error,omitempty"`
}

// WSInbound is a message sent from a websocket client to the server.
type WSInbound struct {
	Type           string `json:"type"` // send | typing | read
	ConversationId int    `json:"conversation_id"`
	Body           string `json:"body"`
	ClientId       string `json:"client_id"`
	MessageId      int64  `json:"message_id"`
}
