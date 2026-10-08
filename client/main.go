// Command client is a terminal tic-tac-toe client for the WebSocket server.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/rupeshxb/tic-tac-toe-go/internal/protocol"
)

// errQuit is returned when stdin is closed and the player cannot move.
var errQuit = errors.New("input closed")

// client holds the connection and local game state.
type client struct {
	conn    *websocket.Conn
	stdin   *bufio.Scanner
	board   [9]string
	symbol  string
	pending int // position of our last move awaiting server acceptance, or -1
}

// main parses flags, connects to the server and runs the game.
func main() {
	server := flag.String("server", "ws://localhost:8080/ws", "WebSocket server URL")
	flag.Parse()

	conn, _, err := websocket.DefaultDialer.Dial(*server, nil)
	if err != nil {
		log.Fatalf("connect to %s: %v", *server, err)
	}
	defer conn.Close()

	c := &client{conn: conn, stdin: bufio.NewScanner(os.Stdin), pending: -1}
	if err := c.run(); err != nil && !errors.Is(err, errQuit) {
		log.Fatal(err)
	}
}

// run reads server messages and dispatches them until the game ends.
func (c *client) run() error {
	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("connection lost: %w", err)
		}
		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("ignoring malformed message: %v", err)
			continue
		}

		switch msg.Type {
		case protocol.MsgWaiting:
			fmt.Println("Waiting for opponent...")

		case protocol.MsgStart:
			var p protocol.StartPayload
			if err := decode(msg, &p); err != nil {
				return err
			}
			c.symbol = p.Symbol
			order := "Opponent goes first."
			if p.GoesFirst {
				order = "You go first."
			}
			fmt.Printf("You are %s. %s\n", p.Symbol, order)

		case protocol.MsgYourTurn:
			c.pending = -1
			if err := c.takeTurn(); err != nil {
				return err
			}

		case protocol.MsgOpponentMove:
			var p protocol.MovePayload
			if err := decode(msg, &p); err != nil {
				return err
			}
			if p.Position < 0 || p.Position > 8 {
				return fmt.Errorf("server sent invalid position %d", p.Position)
			}
			c.board[p.Position] = opposite(c.symbol)
			fmt.Printf("Opponent played at %d\n", p.Position)

		case protocol.MsgGameOver:
			var p protocol.GameOverPayload
			if err := decode(msg, &p); err != nil {
				return err
			}
			c.board = p.Board
			fmt.Println()
			fmt.Print(render(c.board))
			announce(p)
			return nil

		case protocol.MsgError:
			var p protocol.ErrorPayload
			if err := decode(msg, &p); err != nil {
				return err
			}
			// The server re-sends YOUR_TURN after a rejected move, which
			// prompts for input again. Undo our tentative mark first.
			if c.pending >= 0 {
				c.board[c.pending] = ""
				c.pending = -1
			}
			fmt.Printf("Error: %s\n", p.Message)

		default:
			log.Printf("ignoring unknown message type %q", msg.Type)
		}
	}
}

// takeTurn prints the board, prompts until a valid position is entered
// and sends it to the server.
func (c *client) takeTurn() error {
	fmt.Println()
	fmt.Print(render(c.board))
	for {
		fmt.Print("Your move (0-8): ")
		if !c.stdin.Scan() {
			return errQuit
		}
		pos, err := strconv.Atoi(strings.TrimSpace(c.stdin.Text()))
		if err != nil || pos < 0 || pos > 8 {
			fmt.Println("Please enter a whole number from 0 to 8.")
			continue
		}
		if c.board[pos] != "" {
			fmt.Printf("Cell %d is already taken.\n", pos)
			continue
		}

		payload, err := json.Marshal(protocol.MovePayload{Position: pos})
		if err != nil {
			return err
		}
		out, err := json.Marshal(protocol.Message{Type: protocol.MsgMove, Payload: payload})
		if err != nil {
			return err
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, out); err != nil {
			return fmt.Errorf("send move: %w", err)
		}
		// The server does not echo our own moves, so record it locally.
		c.board[pos] = c.symbol
		c.pending = pos
		return nil
	}
}

// decode unmarshals a message's raw JSON payload into v.
func decode(msg protocol.Message, v interface{}) error {
	if err := json.Unmarshal(msg.Payload, v); err != nil {
		return fmt.Errorf("decode %s payload: %w", msg.Type, err)
	}
	return nil
}

// announce prints the game result (and reason, if any).
func announce(p protocol.GameOverPayload) {
	switch p.Result {
	case "WIN":
		fmt.Println("You win!")
	case "LOSE":
		fmt.Println("You lose.")
	case "DRAW":
		fmt.Println("It's a draw.")
	default:
		fmt.Printf("Game over: %s\n", p.Result)
	}
	if p.Reason != "" {
		fmt.Printf("Reason: %s\n", p.Reason)
	}
}

// render draws the board as a 3x3 grid, showing the position number in
// empty cells and X or O in played cells.
func render(b [9]string) string {
	cell := func(i int) string {
		if b[i] == "" {
			return strconv.Itoa(i)
		}
		return b[i]
	}
	var sb strings.Builder
	for row := 0; row < 3; row++ {
		i := row * 3
		fmt.Fprintf(&sb, " %s | %s | %s\n", cell(i), cell(i+1), cell(i+2))
		if row < 2 {
			sb.WriteString("---+---+---\n")
		}
	}
	return sb.String()
}

// opposite returns the other player's symbol.
func opposite(symbol string) string {
	if symbol == "X" {
		return "O"
	}
	return "X"
}
