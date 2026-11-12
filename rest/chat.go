package rest

import (
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	controller "github.com/mdhasib01/go-rest-starter/controller"
	"github.com/mdhasib01/go-rest-starter/model"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// The socket is authenticated by the bearer token in the query string,
	// not by cookies, so a cross-site page cannot ride the user's session.
	// CORS for the REST API is "*" as well.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ServeWebsocket godoc
// @Summary realtime websocket (chat + notifications)
// @Description browsers cannot set headers on websockets, so the token is sent as ?token=.
// @Description client -> server: {type: send|typing|read|ping, conversation_id, body, client_id, message_id}
// @Description server -> client: {type: message|typing|read|notification|error|pong, ...}
// @Tags chat
// @Param token query string true "access token"
// @Router /ws [get]
func ServeWebsocket(w http.ResponseWriter, r *http.Request) {
	session, err := controller.IsAuthorized(r.URL.Query().Get("token"), anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote the HTTP error
	}
	controller.M.Serve(conn, session)
}

// GetConversations godoc
// @Summary the caller's conversations with last message and unread count
// @Tags chat
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Success 200 {array} model.Conversation
// @Router /conversations [get]
func GetConversations(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	list, err := controller.GetConversations(session)
	JSON(w, list, err)
}

// GetConversation godoc
// @Summary one of the caller's conversations
// @Tags chat
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "conversation id"
// @Success 200 {object} model.Conversation
// @Router /conversation/{id} [get]
func GetConversation(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	c, err := controller.GetConversation(session, id)
	JSON(w, c, err)
}

// GetMessages godoc
// @Summary message history, oldest first
// @Tags chat
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "conversation id"
// @Param before_id query int false "only messages older than this id"
// @Param size query int false "max 100" default(50)
// @Success 200 {array} model.Message
// @Router /conversation/{id}/messages [get]
func GetMessages(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	q := r.URL.Query()
	before, _ := strconv.ParseInt(q.Get("before_id"), 10, 64)
	size, _ := strconv.Atoi(q.Get("size"))
	list, err := controller.GetMessages(session, id, before, size)
	JSON(w, list, err)
}

// OpenDirectConversation godoc
// @Summary open a 1:1 chat about a post
// @Description allowed between the post author and a user interested in the post
// @Tags chat
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param body body model.DirectConversationRequest true "post_id, user_id"
// @Success 200 {object} model.Conversation
// @Failure 403 {string} string
// @Router /conversation/direct [post]
func OpenDirectConversation(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var req model.DirectConversationRequest
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	c, err := controller.OpenDirectConversation(session, req)
	JSON(w, c, err)
}

// SendMessage godoc
// @Summary send a message (REST fallback of the websocket "send" event)
// @Tags chat
// @Accept  json
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "conversation id"
// @Param body body model.SendMessageRequest true "body, client_id"
// @Success 200 {object} model.Message
// @Router /conversation/{id}/message [post]
func SendMessage(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	var req model.SendMessageRequest
	if err := readRequestBody(r, &req); err != nil {
		JSON(w, nil, err)
		return
	}
	msg, err := controller.SendMessage(session.UserId, id, req.Body, req.ClientId)
	JSON(w, msg, err)
}

// MarkConversationRead godoc
// @Summary mark messages up to message_id as read
// @Tags chat
// @Produce  json
// @Security Bearer
// @Param Authorization header string true "Bearer Token" default("Bearer YOUR_ACCESS_TOKEN")
// @Param id path int true "conversation id"
// @Param message_id query int true "last read message id"
// @Success 200 {string} string
// @Router /conversation/{id}/read [put]
func MarkConversationRead(w http.ResponseWriter, r *http.Request) {
	session, err := isAuthorized(r, anyAuthenticatedUser)
	if err != nil {
		JSON(w, nil, err)
		return
	}
	id, err := pathInt(r, "id")
	if err != nil {
		JSON(w, nil, err)
		return
	}
	msgID, _ := strconv.ParseInt(r.URL.Query().Get("message_id"), 10, 64)
	err = controller.MarkConversationRead(session.UserId, id, msgID)
	JSON(w, "ok", err)
}
