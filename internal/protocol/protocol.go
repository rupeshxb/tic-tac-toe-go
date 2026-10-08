// Package protocol defines the JSON message envelope and payload types
// exchanged between the game server and the browser client.
package protocol

import "encoding/json"

// MessageType identifies what kind of message an envelope carries.
type MessageType string

// Server -> Client message types.
const (
	// MsgWaiting tells a connected player to wait for an opponent.
	MsgWaiting MessageType = "WAITING"
	// MsgStart announces the game has begun (payload: StartPayload).
	MsgStart MessageType = "START"
	// MsgYourTurn tells the player it is now their turn to move.
	MsgYourTurn MessageType = "YOUR_TURN"
	// MsgOpponentMove reports the opponent's move (payload: MovePayload).
	MsgOpponentMove MessageType = "OPPONENT_MOVE"
	// MsgGameOver announces the game has ended (payload: GameOverPayload).
	MsgGameOver MessageType = "GAME_OVER"
	// MsgError reports a rejected action or failure (payload: ErrorPayload).
	MsgError MessageType = "ERROR"
)

// Client -> Server message types.
const (
	// MsgMove is sent by a player to make a move (payload: MovePayload).
	MsgMove MessageType = "MOVE"
)

// Message is the envelope for every message sent over the WebSocket.
// Payload holds the raw JSON of one of the payload types below, so the
// receiver can decode it once Type is known. It is omitted when empty.
type Message struct {
	Type    MessageType     `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// StartPayload is sent with MsgStart to tell a player their symbol and
// whether they move first.
type StartPayload struct {
	Symbol    string `json:"symbol"` // "X" or "O"
	GoesFirst bool   `json:"goesFirst"`
}

// MovePayload carries a board position, used by both MsgMove and MsgOpponentMove.
type MovePayload struct {
	Position int `json:"position"` // 0-8, row-major
}

// GameOverPayload is sent with MsgGameOver to report the outcome from the
// receiving player's perspective along with the final board.
type GameOverPayload struct {
	Result string    `json:"result"` // "WIN", "LOSE" or "DRAW"
	Reason string    `json:"reason,omitempty"`
	Board  [9]string `json:"board"`
}

// ErrorPayload is sent with MsgError to describe what went wrong.
type ErrorPayload struct {
	Message string `json:"message"`
}
