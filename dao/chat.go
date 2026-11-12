package dao

import (
	"database/sql"
	"errors"

	itn "github.com/mdhasib01/go-rest-starter/itn"
	model "github.com/mdhasib01/go-rest-starter/model"
	"github.com/mdhasib01/go-rest-starter/pkg/logger"
)

// createBankConversation returns the bank's group chat, creating it if needed.
func createBankConversation(tx *sql.Tx, bankID int) (int, error) {
	var id int
	err := tx.QueryRow(`INSERT INTO conversations (type, blood_bank_id) VALUES ('bank', $1)
		ON CONFLICT (blood_bank_id) DO UPDATE SET blood_bank_id = EXCLUDED.blood_bank_id
		RETURNING id`, bankID).Scan(&id)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	return id, nil
}

// addParticipant adds the user to a conversation. Messages sent before
// they joined are treated as read.
func addParticipant(tx *sql.Tx, convID, userID int) error {
	_, err := tx.Exec(`INSERT INTO conversation_participants (conversation_id, user_id, last_read_message_id)
		VALUES ($1, $2, COALESCE((SELECT MAX(id) FROM messages WHERE conversation_id = $1), 0))
		ON CONFLICT (conversation_id, user_id) DO NOTHING`, convID, userID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

// GetOrCreateDirectConversation returns the 1:1 chat about a post between
// two users, creating it (with both participants) when needed.
func GetOrCreateDirectConversation(postID, userA, userB int) (int, error) {
	low, high := userA, userB
	if low > high {
		low, high = high, low
	}

	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	defer tx.Rollback()

	var id int
	err = tx.QueryRow(`INSERT INTO conversations (type, post_id, direct_user_low, direct_user_high)
		VALUES ('direct', $1, $2, $3)
		ON CONFLICT (post_id, direct_user_low, direct_user_high) WHERE type = 'direct'
		DO UPDATE SET post_id = EXCLUDED.post_id
		RETURNING id`, postID, low, high).Scan(&id)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	for _, u := range []int{low, high} {
		if err = addParticipant(tx, id, u); err != nil {
			return 0, err
		}
	}
	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	return id, nil
}

// IsParticipant reports whether the user is in the conversation.
func IsParticipant(convID, userID int) (bool, error) {
	var ok bool
	err := DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM conversation_participants
		WHERE conversation_id = $1 AND user_id = $2)`, convID, userID).Scan(&ok)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.UnknownError
	}
	return ok, nil
}

// GetParticipantIDs returns the user ids of a conversation's participants.
func GetParticipantIDs(convID int) ([]int, error) {
	return queryInts(`SELECT user_id FROM conversation_participants WHERE conversation_id = $1`, convID)
}

// InsertMessage stores a message and bumps the conversation and the
// sender's read marker.
func InsertMessage(convID, senderID int, body string) (model.Message, error) {
	m := model.Message{ConversationId: convID, SenderId: senderID, Body: body}

	tx, err := DB.Begin()
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return m, model.UnknownError
	}
	defer tx.Rollback()

	err = tx.QueryRow(`INSERT INTO messages (conversation_id, sender_id, body) VALUES ($1, $2, $3)
		RETURNING id, created_at`, convID, senderID, body).Scan(&m.Id, &m.CreatedAt)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return m, model.UnknownError
	}
	if _, err = tx.Exec(`UPDATE conversations SET last_message_at = $2 WHERE id = $1`, convID, m.CreatedAt); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return m, model.UnknownError
	}
	if _, err = tx.Exec(`UPDATE conversation_participants SET last_read_message_id = $3
		WHERE conversation_id = $1 AND user_id = $2`, convID, senderID, m.Id); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return m, model.UnknownError
	}
	if err = tx.QueryRow(`SELECT TRIM(COALESCE(first_name, '') || ' ' || COALESCE(last_name, '')), COALESCE(photo, '')
		FROM users WHERE id = $1`, senderID).Scan(&m.SenderName, &m.SenderPhoto); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return m, model.UnknownError
	}

	if err = tx.Commit(); err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return m, model.UnknownError
	}
	return m, nil
}

// GetMessages returns up to size messages older than beforeID (0 = newest),
// oldest first.
func GetMessages(convID int, beforeID int64, size int) ([]model.Message, error) {
	rows, err := DB.Query(`SELECT * FROM (
			SELECT m.id, m.conversation_id, COALESCE(m.sender_id, 0),
				TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')),
				m.body, m.created_at, COALESCE(u.photo, '')
			FROM messages m LEFT JOIN users u ON u.id = m.sender_id
			WHERE m.conversation_id = $1 AND m.deleted_at IS NULL AND ($2::bigint = 0 OR m.id < $2::bigint)
			ORDER BY m.id DESC LIMIT $3
		) t ORDER BY 1 ASC`, convID, beforeID, size)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	defer rows.Close()

	list := []model.Message{}
	for rows.Next() {
		var m model.Message
		if err := rows.Scan(&m.Id, &m.ConversationId, &m.SenderId, &m.SenderName, &m.Body, &m.CreatedAt, &m.SenderPhoto); err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

// MarkRead moves the user's read marker forward.
func MarkRead(convID, userID int, messageID int64) error {
	_, err := DB.Exec(`UPDATE conversation_participants SET last_read_message_id = GREATEST(last_read_message_id, $3)
		WHERE conversation_id = $1 AND user_id = $2`, convID, userID, messageID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return model.UnknownError
	}
	return nil
}

const conversationColumns = `
	c.id, c.type, COALESCE(c.blood_bank_id, 0), COALESCE(b.name, ''),
	COALESCE(c.post_id, 0), COALESCE(p.title, ''),
	COALESCE(o.id, 0), COALESCE(o.first_name, ''), COALESCE(o.last_name, ''), COALESCE(o.photo, ''),
	lm.id, COALESCE(lm.sender_id, 0),
	TRIM(COALESCE(lu.first_name, '') || ' ' || COALESCE(lu.last_name, '')),
	COALESCE(lm.body, ''), lm.created_at,
	(SELECT COUNT(*) FROM messages um WHERE um.conversation_id = c.id
		AND um.id > cp.last_read_message_id AND um.deleted_at IS NULL)::int,
	c.last_message_at, c.created_at`

const conversationFrom = `
	FROM conversation_participants cp
	JOIN conversations c ON c.id = cp.conversation_id
	LEFT JOIN blood_banks b ON b.id = c.blood_bank_id
	LEFT JOIN donation_posts p ON p.id = c.post_id
	LEFT JOIN users o ON c.type = 'direct'
		AND o.id = CASE WHEN c.direct_user_low = cp.user_id THEN c.direct_user_high ELSE c.direct_user_low END
	LEFT JOIN LATERAL (SELECT id, sender_id, body, created_at FROM messages
		WHERE conversation_id = c.id AND deleted_at IS NULL ORDER BY id DESC LIMIT 1) lm ON true
	LEFT JOIN users lu ON lu.id = lm.sender_id`

func scanConversation(row rowScanner) (model.Conversation, error) {
	var c model.Conversation
	var other model.ChatUser
	var lmID sql.NullInt64
	var lmSender int
	var lmSenderName, lmBody string
	var lmAt, lastAt sql.NullTime
	err := row.Scan(&c.Id, &c.Type, &c.BloodBankId, &c.BloodBankName, &c.PostId, &c.PostTitle,
		&other.Id, &other.FirstName, &other.LastName, &other.Photo,
		&lmID, &lmSender, &lmSenderName, &lmBody, &lmAt,
		&c.UnreadCount, &lastAt, &c.CreatedAt)
	if err != nil {
		return c, err
	}
	if other.Id != 0 {
		c.OtherUser = &other
	}
	if lmID.Valid {
		c.LastMessage = &model.Message{Id: lmID.Int64, ConversationId: c.Id, SenderId: lmSender,
			SenderName: lmSenderName, Body: lmBody, CreatedAt: lmAt.Time}
	}
	c.LastMessageAt = nullTimePtr(lastAt)
	return c, nil
}

// GetConversations lists the user's conversations, most recent first.
func GetConversations(userID int) ([]model.Conversation, error) {
	rows, err := DB.Query(`SELECT `+conversationColumns+conversationFrom+`
		WHERE cp.user_id = $1
		ORDER BY COALESCE(c.last_message_at, c.created_at) DESC, c.id DESC`, userID)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return nil, model.UnknownError
	}
	defer rows.Close()

	list := []model.Conversation{}
	for rows.Next() {
		c, err := scanConversation(rows)
		if err != nil {
			logger.GetLogger().LogErrors(err, nil)
			return nil, model.UnknownError
		}
		list = append(list, c)
	}
	return list, rows.Err()
}

// GetConversation returns one of the user's conversations.
func GetConversation(convID, userID int) (model.Conversation, error) {
	c, err := scanConversation(DB.QueryRow(`SELECT `+conversationColumns+conversationFrom+`
		WHERE cp.user_id = $1 AND c.id = $2`, userID, convID))
	if errors.Is(err, sql.ErrNoRows) {
		return c, model.NewError(itn.ErrorConversationNotFound, 404)
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return c, model.UnknownError
	}
	return c, nil
}

// GetBankConversationID returns the id of the bank's group chat (0 if none).
func GetBankConversationID(bankID int) (int, error) {
	var id int
	err := DB.QueryRow(`SELECT id FROM conversations WHERE blood_bank_id = $1`, bankID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return 0, model.UnknownError
	}
	return id, nil
}

// HasInterest reports whether the user tapped "interested" on the post.
func HasInterest(postID, userID int) (bool, error) {
	var ok bool
	err := DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM donation_post_interests WHERE post_id = $1 AND user_id = $2)`,
		postID, userID).Scan(&ok)
	if err != nil {
		logger.GetLogger().LogErrors(err, nil)
		return false, model.UnknownError
	}
	return ok, nil
}
