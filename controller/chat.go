package controller

import (
	"strings"
	"unicode/utf8"

	dao "github.com/mdhasib01/go-rest-starter/dao"
	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
)

const (
	defaultMessagePageSize = 50
	maxMessagePageSize     = 100
)

var rateLimitedError = model.NewError(itn.ErrorRateLimited, 429)

func GetConversations(session model.Session) ([]model.Conversation, error) {
	return dao.GetConversations(session.UserId)
}

func GetConversation(session model.Session, convID int) (model.Conversation, error) {
	return dao.GetConversation(convID, session.UserId)
}

// GetMessages returns a page of history; only participants can read it.
func GetMessages(session model.Session, convID int, beforeID int64, size int) ([]model.Message, error) {
	if err := requireParticipant(convID, session.UserId); err != nil {
		return nil, err
	}
	if size < 1 {
		size = defaultMessagePageSize
	}
	if size > maxMessagePageSize {
		size = maxMessagePageSize
	}
	return dao.GetMessages(convID, beforeID, size)
}

// OpenDirectConversation gets or creates the 1:1 chat about a post. It is
// allowed only between the post author and a user interested in the post.
func OpenDirectConversation(session model.Session, req model.DirectConversationRequest) (model.Conversation, error) {
	if req.UserId == 0 || req.UserId == session.UserId || req.PostId == 0 {
		return model.Conversation{}, model.NewError(itn.ErrorDirectChatNotAllowed, 400)
	}
	post, err := dao.GetPost(req.PostId)
	if err != nil {
		return model.Conversation{}, err
	}

	var donor int
	switch post.AuthorId {
	case session.UserId:
		donor = req.UserId
	case req.UserId:
		donor = session.UserId
	default:
		return model.Conversation{}, model.NewError(itn.ErrorDirectChatNotAllowed, 403)
	}
	interested, err := dao.HasInterest(post.Id, donor)
	if err != nil {
		return model.Conversation{}, err
	}
	if !interested {
		return model.Conversation{}, model.NewError(itn.ErrorDirectChatNotAllowed, 403)
	}

	id, err := dao.GetOrCreateDirectConversation(post.Id, session.UserId, req.UserId)
	if err != nil {
		return model.Conversation{}, err
	}
	return dao.GetConversation(id, session.UserId)
}

// SendMessage stores a message and pushes it to every participant
// (including the sender's other tabs). clientID lets the sender match the
// echo with its optimistic message.
func SendMessage(userID, convID int, body, clientID string) (model.Message, error) {
	body = strings.TrimSpace(body)
	if body == "" || utf8.RuneCountInString(body) > model.MaxMessageLength {
		return model.Message{}, model.NewError(itn.ErrorMessageInvalid, 400)
	}
	if len(clientID) > 64 {
		clientID = clientID[:64]
	}
	if err := requireParticipant(convID, userID); err != nil {
		return model.Message{}, err
	}

	msg, err := dao.InsertMessage(convID, userID, body)
	if err != nil {
		return model.Message{}, err
	}

	ids, err := dao.GetParticipantIDs(convID)
	if err == nil && M != nil {
		M.SendToUsers(ids, model.WSEvent{Type: "message", ConversationId: convID, Message: &msg, ClientId: clientID})
	}
	return msg, nil
}

// BroadcastTyping tells the other participants that the user is typing.
func BroadcastTyping(userID, convID int) {
	if M == nil || requireParticipant(convID, userID) != nil {
		return
	}
	ids, err := dao.GetParticipantIDs(convID)
	if err != nil {
		return
	}
	others := make([]int, 0, len(ids))
	for _, id := range ids {
		if id != userID {
			others = append(others, id)
		}
	}
	M.SendToUsers(others, model.WSEvent{Type: "typing", ConversationId: convID, UserId: userID})
}

// MarkConversationRead moves the user's read marker and syncs their
// other tabs.
func MarkConversationRead(userID, convID int, messageID int64) error {
	if err := requireParticipant(convID, userID); err != nil {
		return err
	}
	if err := dao.MarkRead(convID, userID, messageID); err != nil {
		return err
	}
	if M != nil {
		M.SendToUsers([]int{userID}, model.WSEvent{Type: "read", ConversationId: convID, MessageId: messageID})
	}
	return nil
}

func requireParticipant(convID, userID int) error {
	ok, err := dao.IsParticipant(convID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return model.NewError(itn.ErrorConversationNotFound, 404)
	}
	return nil
}
