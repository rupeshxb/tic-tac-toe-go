# Tic-Tac-Toe over WebSockets

A multiplayer tic-tac-toe game playable over the network, written in Go. It ships two binaries, a server and a terminal client, plus a browser client that the server serves as a static file.

> **Two players are required.** The server pairs players as they connect, so a game only starts once **two clients are connected at the same time**. In the browser, open the page in **two tabs** (the first tab shows "Waiting for opponent..." until the second one opens). With the terminal client, start two clients in two terminals.

## Project structure

```
server/main.go         WebSocket server, matchmaking, game loop
client/main.go         Terminal client binary
static/index.html      Browser client served by the server
internal/game/         Pure game logic, no I/O (Board, ApplyMove, CheckWinner, IsDraw)
internal/protocol/     Shared message types using json.RawMessage
.github/workflows/     GitHub Actions CI
Dockerfile             Multi-stage build for Railway deployment
```

## Design

### Transport: WebSockets, not raw TCP

- Sits naturally behind a reverse proxy, which is how it would run in Kubernetes behind an ingress.
- Railway and other cloud platforms speak HTTP natively, so no special TCP port configuration is needed.
- The browser becomes a zero-install client anyone can use.
- Raw TCP would need custom message framing; WebSockets provide it.

### Matchmaking

- `waitingPool` is a buffered channel of capacity 1. It acts as an atomic slot for one waiting player.
- The first player puts a `waiter{client, matchCh}` into the pool and blocks on `matchCh`.
- The second player takes the waiter from the pool and sends itself over `matchCh`, which wakes the first player.
- The first player then starts the game. The first player is X and moves first.
- No mutex is needed. The channel provides the atomicity.
- Each game runs in its own goroutine, so the server runs any number of concurrent games.

### Concurrency model

- One goroutine per WebSocket connection runs `handleWS`.
- Once two players are paired, `runGame` owns both connections. No state is shared between goroutines after pairing.
- If a player disconnects mid-game, the surviving player receives `GAME_OVER` with result `WIN` and reason `opponent disconnected`.

### Protocol

JSON messages over WebSocket. Every message is an envelope:

```json
{ "type": "MOVE", "payload": { "position": 4 } }
```

`Payload` is a `json.RawMessage`, so each side decodes only the payload it needs once it has checked `type`.

| Direction | Type | Payload |
|---|---|---|
| server → client | `WAITING` | none |
| server → client | `START` | `{symbol, goesFirst}` |
| server → client | `YOUR_TURN` | none |
| server → client | `OPPONENT_MOVE` | `{position}` |
| server → client | `GAME_OVER` | `{result, reason, board}` (`result` is `WIN`, `LOSE` or `DRAW`) |
| server → client | `ERROR` | `{message}` |
| client → server | `MOVE` | `{position}` (0-8, row-major) |

The server validates every move (bounds, turn, occupied cell). A rejected move gets `ERROR`, followed by another `YOUR_TURN`. The server does not echo a player's own move back, so clients record it locally.

## Running locally

Requires Go 1.22 or later.

```sh
# Start the server on port 8080 (override with PORT)
go run ./server
```

**Browser:** open http://localhost:8080 in **two tabs**. Each tab is one player.

**Terminal client:** run in two separate terminals:

```sh
go run ./client -server ws://localhost:8080/ws
```

The two kinds of client can play each other.

Run the tests:

```sh
go test ./...
```

## Running against the live deployment

**Browser:** open https://tic-tac-toe-go-by-rupesh.up.railway.app/ in **two tabs in the same broswer** (or on two different devices).

**Terminal:**

```sh
go run ./client -server wss://tic-tac-toe-go-by-rupesh.up.railway.app/ws
```

## CI

GitHub Actions (`.github/workflows/ci.yml`) runs `go vet`, `go build` and `go test -race` on every push and pull request.

## Deployment

The `Dockerfile` is a multi-stage build. The builder stage compiles the server binary, and the final stage is minimal Alpine containing only the binary and the `static/` folder. It is deployed on Railway, and the server reads the listen port from the `PORT` environment variable (default 8080).

## Known limitations

- A player who disconnects while waiting for an opponent is not detected. The next player is paired with the dead connection and wins immediately by "opponent disconnected".
- A disconnect is noticed only when the server next reads from or writes to that player, not while it is the other player's turn.
- Only `internal/game` has unit tests. The server and client are not covered by automated tests.
