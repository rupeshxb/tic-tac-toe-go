// Command server runs the WebSocket tic-tac-toe server.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"github.com/rupeshxb/tic-tac-toe-go/internal/game"
	"github.com/rupeshxb/tic-tac-toe-go/internal/protocol"
)

// upgrader turns HTTP requests into WebSocket connections. CheckOrigin
// allows every origin, which development and Railway deployment require.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Client wraps a player's WebSocket connection.
type Client struct {
	conn *websocket.Conn
}

// send marshals msg to JSON and writes it as a text message.
func (c *Client) send(msg protocol.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

// read blocks for the next message. A connection failure is returned as
// connErr; a message that is not valid JSON is returned as parseErr so the
// caller can reject it without ending the game.
func (c *Client) read() (msg protocol.Message, connErr, parseErr error) {
	_, data, err := c.conn.ReadMessage()
	if err != nil {
		return msg, err, nil
	}
	return msg, nil, json.Unmarshal(data, &msg)
}

// waiter is a client waiting for an opponent. The client that pairs with it
// delivers itself on matchCh.
type waiter struct {
	client  *Client
	matchCh chan *Client
}

// waitingPool holds at most one waiting player at a time.
var waitingPool = make(chan *waiter, 1)

// newMessage builds a Message with the payload marshaled to JSON.
// Marshaling the protocol payload types cannot fail, so the error is ignored.
func newMessage(t protocol.MessageType, payload interface{}) protocol.Message {
	msg := protocol.Message{Type: t}
	if payload != nil {
		msg.Payload, _ = json.Marshal(payload)
	}
	return msg
}

// handleWS upgrades the connection, tells the player to wait, then either
// pairs with a waiting player or waits for an opponent to arrive.
func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade failed: %v", err)
		return
	}
	c := &Client{conn: conn}
	log.Printf("connected: %s", conn.RemoteAddr())

	if err := c.send(newMessage(protocol.MsgWaiting, nil)); err != nil {
		log.Printf("send waiting to %s failed: %v", conn.RemoteAddr(), err)
		conn.Close()
		return
	}

	for {
		select {
		case wt := <-waitingPool:
			// Hand ourselves to the waiter, who starts the game.
			wt.matchCh <- c
			return
		default:
		}

		wt := &waiter{client: c, matchCh: make(chan *Client, 1)}
		select {
		case waitingPool <- wt:
			opponent := <-wt.matchCh
			log.Printf("paired: X=%s O=%s", c.conn.RemoteAddr(), opponent.conn.RemoteAddr())
			go runGame(c, opponent)
			return
		default:
			// Another player took the slot first; try picking them up.
		}
	}
}

// runGame plays one full game between the two clients and closes both
// connections when it finishes.
func runGame(clientX, clientO *Client) {
	defer clientX.conn.Close()
	defer clientO.conn.Close()

	g := game.New()
	players := map[string]*Client{"X": clientX, "O": clientO}

	// finish tells both players the result and logs the outcome.
	finish := func(winner string) {
		for sym, c := range players {
			result := "DRAW"
			if winner != "" {
				result = "LOSE"
				if sym == winner {
					result = "WIN"
				}
			}
			payload := protocol.GameOverPayload{Result: result, Board: [9]string(g.Board)}
			if err := c.send(newMessage(protocol.MsgGameOver, payload)); err != nil {
				log.Printf("send game over to %s failed: %v", sym, err)
			}
		}
		if winner == "" {
			log.Printf("game over: draw")
		} else {
			log.Printf("game over: %s wins", winner)
		}
	}

	// abandon tells the survivor they won because the other player left.
	abandon := func(gone string) {
		survivor := players[other(gone)]
		log.Printf("player %s disconnected; %s wins", gone, other(gone))
		payload := protocol.GameOverPayload{
			Result: "WIN",
			Reason: "opponent disconnected",
			Board:  [9]string(g.Board),
		}
		if err := survivor.send(newMessage(protocol.MsgGameOver, payload)); err != nil {
			log.Printf("send game over to survivor failed: %v", err)
		}
	}

	for sym, c := range players {
		start := protocol.StartPayload{Symbol: sym, GoesFirst: sym == "X"}
		if err := c.send(newMessage(protocol.MsgStart, start)); err != nil {
			abandon(sym)
			return
		}
	}
	log.Printf("game started")

	for {
		turn := g.CurrentTurn
		cur, opp := players[turn], players[other(turn)]

		if err := cur.send(newMessage(protocol.MsgYourTurn, nil)); err != nil {
			abandon(turn)
			return
		}

		msg, connErr, parseErr := cur.read()
		if connErr != nil {
			abandon(turn)
			return
		}

		position, err := decodeMove(msg, parseErr)
		if err == nil {
			err = g.ApplyMove(position, turn)
		}
		if err != nil {
			log.Printf("rejected move from %s: %v", turn, err)
			errMsg := newMessage(protocol.MsgError, protocol.ErrorPayload{Message: err.Error()})
			if sendErr := cur.send(errMsg); sendErr != nil {
				abandon(turn)
				return
			}
			continue // resend YourTurn and let them retry
		}
		log.Printf("move: %s -> %d", turn, position)

		moveMsg := newMessage(protocol.MsgOpponentMove, protocol.MovePayload{Position: position})
		if err := opp.send(moveMsg); err != nil {
			abandon(other(turn))
			return
		}

		if g.IsOver() {
			finish(g.CheckWinner())
			return
		}
	}
}

// decodeMove extracts the board position from a MsgMove message.
func decodeMove(msg protocol.Message, parseErr error) (int, error) {
	if parseErr != nil {
		return 0, errString("invalid message")
	}
	if msg.Type != protocol.MsgMove {
		return 0, errString("expected a MOVE message")
	}
	var p protocol.MovePayload
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return 0, errString("invalid move payload")
	}
	return p.Position, nil
}

// errString is a minimal error type for client-facing validation messages.
type errString string

// Error returns the message text.
func (e errString) Error() string { return string(e) }

// other returns the opposing symbol.
func other(symbol string) string {
	if symbol == "X" {
		return "O"
	}
	return "X"
}

// main reads PORT (default 8080), registers the routes and starts serving.
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/ws", handleWS)

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
